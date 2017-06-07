package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/spf13/viper"
)

// Globals that multiple tests will use
var cloudTrailEvents CloudTrailLogFile

func TestMain(m *testing.M) {
	wd, _ := os.Getwd()
	dataDir := wd + "/../../testdata"

	// Use a test config file
	viper.SetConfigName("test-config")
	viper.AddConfigPath(dataDir + "/config")
	if verr := viper.ReadInConfig(); verr != nil {
		log.Printf("Error reading config file %s:\n%s\n", viper.ConfigFileUsed(), verr.Error())
		os.Exit(1)
	}

	// Init CloudTrail test data
	ctFileName := dataDir + "/parse/cloudtrail-test-data-input.json"
	ctf, cerr := ioutil.ReadFile(ctFileName)
	if cerr != nil {
		fmt.Printf("Error opening %s: %s", ctFileName, cerr)
		os.Exit(1)
	}

	if cuerr := json.Unmarshal(ctf, &cloudTrailEvents); cuerr != nil {
		fmt.Println("Error marshalling cloudtrail events:", cuerr.Error())
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// Set stdout to pipe and capture printed output of a Print event
func captureStdOut(f func(interface{}), v interface{}) string {
	oldStdOut := os.Stdout
	r, w, perr := os.Pipe()
	if perr != nil {
		return ""
	}

	os.Stdout = w

	pipeOut := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		pipeOut <- buf.String()
	}()

	// Execute any f that takes an interface{} argument
	f(v)

	w.Close()
	os.Stdout = oldStdOut
	return <-pipeOut
}

func TestPrintCloudTrailEvents(t *testing.T) {
	wd, _ := os.Getwd()

	cases := []struct {
		InputPatterns []string
		ExpectedFile  string
	}{
		{[]string{}, wd + "/../../testdata/parse/cloudtrail-test-data-output.json"},
		{[]string{
			"{CreatedBy: .userIdentity.arn}",
			"{TaggedAt: \"2017-05-31\"}",
			"{ExpiresAt: (1497253282) | strftime(\"%Y-%m-%d\")}",
		}, wd + "/../../testdata/parse/cloudtrail-test-data-output-tagged.json"},
	}

	var ctJSON string
	for _, c := range cases {
		viper.Set("grafiti.tagPatterns", c.InputPatterns)

		f := func(v interface{}) {
			printEvents(v.([]*cloudtrail.Event))
		}

		ctJSON = captureStdOut(f, cloudTrailEvents.Events)

		// Get desired output of printCloudTrailEvent from file
		want, oerr := ioutil.ReadFile(c.ExpectedFile)
		if oerr != nil {
			fmt.Println("Failed to open", c.ExpectedFile)
			t.Fail()
		}

		// NOTE: non-deterministic pass. jq eval will occasionally fail for some reason.
		if string(want) != ctJSON {
			t.Errorf("printCloudTrailEvent failed\nwanted\n%s\n\ngot\n%s\n", string(want), ctJSON)
		}
	}
}

func TestGetTags(t *testing.T) {
	tags := []string{
		"{CreatedBy: .userIdentity.arn}",
		"{TaggedAt: \"2017-05-31\"}",
		"{ExpiresAt: (1497253282) | strftime(\"%Y-%m-%d\")}",
	}

	mockTags := map[string]string{
		"CreatedBy": "arn:aws:iam::123456789101:user/test-user",
		"TaggedAt":  "2017-05-31",
		"ExpiresAt": "2017-06-12",
	}

	viper.Set("grafiti.tagPatterns", tags)

	for _, e := range cloudTrailEvents.Events {
		te := getTags(e)
		if te != nil && !reflect.DeepEqual(te, mockTags) {
			t.Errorf("getTags failed\nwanted:\n%s,\n\ngot:\n%s\n\n", mockTags, te)
		}
	}

}

func TestMatchFilter(t *testing.T) {
	filters := []string{
		".TaggingMetadata.ResourceType == \"AWS::EC2::Instance\"",
		".TaggingMetadata.ResourceType == \"AWS::ElasticLoadBalancing::LoadBalancer\"",
	}

	viper.Set("grafiti.filterPatterns", filters)

	o1 := []string{`{"TaggingMetadata":
	{"ResourceName":"i-0d521e398c491f95a",
	"ResourceType":"AWS::EC2::Instance",
	"ResourceARN":"arn:aws:ec2:us-west-2:123456789101:instance/i-0d521e398c491f95a",
	"CreatorARN":"arn:aws:iam::123456789101:root",
	"CreatorName":"test-user"},
	"Tags":{}}`,
		`{"TaggingMetadata":
	{"ResourceName":"aws-master-423-api-internal",
	"ResourceType":"AWS::ElasticLoadBalancing::LoadBalancer",
	"ResourceARN":"arn:aws:elasticloadbalancing:us-west-2:123456789101:loadbalancer/aws-master-423-api-internal",
	"CreatorARN":"arn:aws:iam::123456789101:user/test-user",
	"CreatorName":"test-user"},
	"Tags":{}}`,
	}

	var bo []byte
	for _, o := range o1 {
		bo = []byte(o)
		if !matchFilter(&bo) {
			t.Errorf("Filter did not match output:\n%s\n", o)
		}
	}

	o2 := `{"TaggingMetadata":{"ResourceName":"vpc-34dcc053",
	"ResourceType":"AWS::EC2::VPC",
	"ResourceARN":"arn:aws:ec2:us-west-2:123456789101:vpc/vpc-34dcc053",
	"CreatorARN":"arn:aws:iam::123456789101:user/test-user",
	"CreatorName":"test-user"},
	"Tags":{}}`

	bo = []byte(o2)
	if matchFilter(&bo) {
		t.Errorf("Filter should not have matched output:\n%s\n", o2)
	}

}

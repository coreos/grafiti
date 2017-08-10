package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/spf13/viper"
)

type mockCloudTrailAPIEvents struct {
	Events []*cloudtrail.Event `json:"Records"`
}

// Globals that multiple tests will use
var (
	cloudTrailAPIEvents     mockCloudTrailAPIEvents
	cloudTrailLogFileEvents CloudTrailLogFile
)

func TestMain(m *testing.M) {
	wd, _ := os.Getwd()
	dataDir := wd + "/../../testdata"
	// Use a test config file
	viper.SetConfigName("test-config")
	viper.AddConfigPath(dataDir + "/config")
	if err := viper.ReadInConfig(); err != nil {
		exitWithError(fmt.Errorf("read config file %q: %s", viper.ConfigFileUsed(), err))
	}

	// Init CloudTrail API test data
	ctAPIInputFile := dataDir + "/parse/cloudtrail-api-input.json"
	ctAPIInputBytes, err := ioutil.ReadFile(ctAPIInputFile)
	if err != nil {
		exitWithError(fmt.Errorf("open file %q: %s", ctAPIInputFile, err))
	}

	if err := json.Unmarshal(ctAPIInputBytes, &cloudTrailAPIEvents); err != nil {
		exitWithError(fmt.Errorf("marshal cloudtrail API events: %s", err))
	}

	// Init CloudTrail logfile test data
	ctLogFileInputFile := dataDir + "/parse/cloudtrail-logfile-input.json"
	ctLogFileInputBytes, err := ioutil.ReadFile(ctLogFileInputFile)
	if err != nil {
		exitWithError(fmt.Errorf("open file %q: %s", ctLogFileInputFile, err))
	}

	if err := json.Unmarshal(ctLogFileInputBytes, &cloudTrailLogFileEvents); err != nil {
		exitWithError(fmt.Errorf("marshal cloudtrail logfile: %s", err))
	}

	os.Exit(m.Run())
}

func TestCalcTimeWindowFromHourRange(t *testing.T) {
	cases := []struct {
		InputStart   int
		InputEnd     int
		ExpectedDiff time.Duration
	}{
		{-8, 0, time.Duration(8) * time.Hour},
		{-25, -17, time.Duration(8) * time.Hour},
		{0, -8, time.Duration(0)},
		{0, 0, time.Duration(0)},
	}

	for i, c := range cases {
		st, et := calcTimeWindowFromHourRange(c.InputStart, c.InputEnd)

		if c.InputStart >= c.InputEnd {
			if st != nil || et != nil {
				t.Errorf("calcTimeWindow case %d failed\nwanted et=nil, st=nil\ngot st=%s, et=%s\n", i+1, st, et)
			}
			continue
		}

		diff := (*et).Sub(*st)
		if c.ExpectedDiff != diff {
			t.Errorf("calcTimeWindow case %d failed\nwanted diff=%s\ngot diff=%s\n", i+1, c.ExpectedDiff, diff)
		}
	}
}

func TestCalcTimeWindowFromTimeStamp(t *testing.T) {
	cases := []struct {
		InputStart   string
		InputEnd     string
		ExpectedDiff time.Duration
	}{
		{"2017-06-14T01:01:01Z", "2017-06-14T09:01:01Z", time.Duration(8) * time.Hour},
		{"2017-06-13T23:01:01Z", "2017-06-14T07:01:01Z", time.Duration(8) * time.Hour},
		{"2017-06-14T09:01:01Z", "2017-06-14T01:01:01Z", time.Duration(0)},
		{"2017-06-14T01:01:01Z", "2017-06-14T01:01:01Z", time.Duration(0)},
	}

	for i, c := range cases {
		st, et := calcTimeWindowFromTimeStamp(c.InputStart, c.InputEnd)

		// Sorting two timestamp strings will put the earlier stamp first
		sorted := []string{c.InputStart, c.InputEnd}
		sort.Strings(sorted)

		if sorted[0] == c.InputEnd {
			if st != nil || et != nil {
				t.Errorf("calcTimeWindow case %d failed\nwanted et=nil, st=nil\ngot st=%s, et=%s\n", i+1, st, et)
			}
			continue
		}

		diff := (*et).Sub(*st)
		if c.ExpectedDiff != diff {
			t.Errorf("calcTimeWindow case %d failed\nwanted diff=%s\ngot diff=%s\n", i+1, c.ExpectedDiff, diff)
		}
	}
}

func TestParseCloudTrailEvent(t *testing.T) {
	wd, _ := os.Getwd()

	cases := []struct {
		InputPatterns []string
		ExpectedFile  string
	}{
		{[]string{}, wd + "/../../testdata/parse/parsed-logfile-output.json"},
		{[]string{
			"{CreatedBy: .userIdentity.arn}",
			"{TaggedAt: \"2017-05-31\"}",
			"{ExpiresAt: (1497253282) | strftime(\"%Y-%m-%d\")}",
		}, wd + "/../../testdata/parse/parsed-logfile-output-tagged.json"},
	}

	var event []byte
	for i, c := range cases {
		viper.Set("tagPatterns", c.InputPatterns)

		// Get desired output of parseRawCloudTrailEvent from file
		want, err := ioutil.ReadFile(c.ExpectedFile)
		if err != nil {
			t.Fatal("Failed to open", c.ExpectedFile)
		}

		var gotStr string
		for _, eventData := range cloudTrailLogFileEvents.Events {
			event, err = eventData.MarshalJSON()
			if err != nil {
				continue
			}

			gotStr += parseRawCloudTrailEvent(string(event)) + "\n"
		}

		// NOTE: non-deterministic pass. jq eval will occasionally fail for some reason.
		if string(want) != gotStr {
			t.Errorf("printCloudTrailEvent case %d failed\nwanted\n%s\n\ngot\n%s\n", i+1, string(want), gotStr)
		}
	}
}

// Set stdout to pipe and capture printed output of a Print event
func captureStdOut(f func(interface{}), v interface{}) string {
	oldStdOut := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
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
		{[]string{}, wd + "/../../testdata/parse/parsed-api-output.json"},
		{[]string{
			"{CreatedBy: .userIdentity.arn}",
			"{TaggedAt: \"2017-05-31\"}",
			"{ExpiresAt: (1497253282) | strftime(\"%Y-%m-%d\")}",
		}, wd + "/../../testdata/parse/parsed-api-output-tagged.json"},
	}

	var ctJSON string
	for i, c := range cases {
		viper.Set("tagPatterns", c.InputPatterns)

		f := func(v interface{}) {
			printEvents(v.([]*cloudtrail.Event))
		}

		ctJSON = captureStdOut(f, cloudTrailAPIEvents.Events)

		// Get desired output of printCloudTrailEvent from file
		want, err := ioutil.ReadFile(c.ExpectedFile)
		if err != nil {
			t.Fatal("Failed to open", c.ExpectedFile)
		}

		// NOTE: non-deterministic pass. jq eval will occasionally fail for some reason.
		if string(want) != ctJSON {
			t.Errorf("printCloudTrailEvent case %d failed\nwanted\n%s\n\ngot\n%s\n", i+1, string(want), ctJSON)
		}
	}
}

func TestGetTags(t *testing.T) {
	tags := []string{
		"{CreatedBy: .userIdentity.arn}",
		"{TaggedAt: \"2017-05-31\"}",
		"{ExpiresAt: (1497253282) | strftime(\"%Y-%m-%d\")}",
	}

	viper.Set("tagPatterns", tags)

	mockTags := map[string]string{
		"CreatedBy": "arn:aws:iam::123456789101:user/test-user",
		"TaggedAt":  "2017-05-31",
		"ExpiresAt": "2017-06-12",
	}

	for _, e := range cloudTrailAPIEvents.Events {
		te := getTags(aws.StringValue(e.CloudTrailEvent))
		if te != nil && !reflect.DeepEqual(te, mockTags) {
			t.Errorf("getTags failed\nwanted:\n%s,\n\ngot:\n%s\n\n", mockTags, te)
		}
	}

}

func TestMatchFilter(t *testing.T) {
	cases := []struct {
		InputFilters  []string
		InputObject   string
		ExpectedMatch bool
	}{
		{
			InputFilters: []string{
				".TaggingMetadata.ResourceType == \"AWS::EC2::Instance\"",
				".TaggingMetadata.CreatorARN == \"arn:aws:iam::123456789101:root\"",
			},
			InputObject: `{"TaggingMetadata":
			{"ResourceName":"i-0d521e398c491f95a",
			"ResourceType":"AWS::EC2::Instance",
			"ResourceARN":"arn:aws:ec2:us-west-2:123456789101:instance/i-0d521e398c491f95a",
			"CreatorARN":"arn:aws:iam::123456789101:root",
			"CreatorName":"test-user"},
			"Tags":{}}`,
			ExpectedMatch: true,
		}, {
			InputFilters: []string{
				".TaggingMetadata.ResourceType == \"AWS::EC2::Instance\"",
				".TaggingMetadata.CreatorARN == \"arn:aws:iam::123456789101:root\"",
			},
			InputObject: `{"TaggingMetadata":
			{"ResourceName":"i-0d521e398c491f95a",
			"ResourceType":"AWS::EC2::Instance",
			"ResourceARN":"arn:aws:ec2:us-west-2:123456789101:instance/i-0d521e398c491f95a",
			"CreatorARN":"arn:aws:iam::123456789101:user/test-user",
			"CreatorName":"test-user"},
			"Tags":{}}`,
			ExpectedMatch: false,
		},
	}

	for i, c := range cases {
		viper.Set("filterPatterns", c.InputFilters)
		if matchFilter([]byte(c.InputObject)) != c.ExpectedMatch {
			t.Errorf("matchFilter case %d failed\nFilter did not match output:\n%s\n", i+1, c.InputObject)
		}
	}

}

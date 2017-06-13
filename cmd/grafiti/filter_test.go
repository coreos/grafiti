package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	rgta "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	rgtaiface "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/spf13/viper"
)

// Mock RGTA API type for AWS requests
type mockRGTAGetResources struct {
	rgtaiface.ResourceGroupsTaggingAPIAPI
	Resp rgta.GetResourcesOutput
}

func (m *mockRGTAGetResources) GetResourcesWithContext(ctx aws.Context, in *rgta.GetResourcesInput, os ...request.Option) (*rgta.GetResourcesOutput, error) {
	return &m.Resp, nil
}

// Set stdout to pipe and capture printed output of a Print event
func captureFilterStdOut(f func(rgtaiface.ResourceGroupsTaggingAPIAPI, io.Reader, io.Reader) error, svc rgtaiface.ResourceGroupsTaggingAPIAPI, v1, v2 io.Reader) string {
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
	f(svc, v1, v2)

	w.Close()
	os.Stdout = oldStdOut
	return <-pipeOut
}

func TestFilter(t *testing.T) {
	wd, _ := os.Getwd()
	dataDir := wd + "/../../testdata"

	viper.Set("grafiti.tagPatterns", []string{})
	viper.Set("grafiti.resourceTypes", []string{})

	cases := []struct {
		InputFilePath    string
		InputTagFilePath string
		InputSVC         *mockRGTAGetResources
		ExpectedFilePath string
	}{
		{
			InputFilePath:    dataDir + "/filter/input-filter.json",
			InputTagFilePath: dataDir + "/filter/input-ignore-tags.json",
			InputSVC: &mockRGTAGetResources{
				Resp: rgta.GetResourcesOutput{
					ResourceTagMappingList: []*rgta.ResourceTagMapping{
						{
							ResourceARN: aws.String("arn:aws:ec2:us-west-2:123456789101:instance/i-0e846a0fc3863dddd"),
						}, {
							ResourceARN: aws.String("arn:aws:autoscaling:us-west-2:123456789101:autoScalingGroup:big-long-string:autoScalingGroupName/demo-master-1"),
						},
					},
				},
			},
			ExpectedFilePath: dataDir + "/filter/output-filter.json",
		},
	}

	var wi bytes.Buffer

	for i, c := range cases {
		// Parse input data
		ifb, ierr := ioutil.ReadFile(c.InputFilePath)
		if ierr != nil {
			fmt.Println("Could not read", c.InputFilePath)
			t.FailNow()
		}
		wi.Write(ifb)

		// Ignore tag file
		tf, terr := os.OpenFile(c.InputTagFilePath, os.O_RDWR, 0644)
		if terr != nil {
			fmt.Println("Could not open", c.InputTagFilePath)
			t.FailNow()
		}

		filteredOutput := captureFilterStdOut(filter, c.InputSVC, &wi, tf)
		filteredOutput = strings.Trim(filteredOutput, "\n")

		ob, rerr := ioutil.ReadFile(c.ExpectedFilePath)
		if rerr != nil {
			fmt.Println("Could not read from", c.ExpectedFilePath)
			t.FailNow()
		}

		if !reflect.DeepEqual(string(ob), filteredOutput) {
			t.Errorf("filter test %d failed\nwanted\n%s\ngot\n%s\n", i+1, string(ob), filteredOutput)
		}

		wi.Reset()
	}
}

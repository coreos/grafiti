package main

import (
	"bytes"
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
func captureFilterStdOut(f func(rgtaiface.ResourceGroupsTaggingAPIAPI, io.Reader, io.Reader) error, svc rgtaiface.ResourceGroupsTaggingAPIAPI, v1, v2 io.Reader) (string, error) {
	oldStdOut := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	os.Stdout = w

	pipeOut := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		pipeOut <- buf.String()
	}()

	// Execute any f that takes an interface{} argument
	if err := f(svc, v1, v2); err != nil {
		w.Close()
		os.Stdout = oldStdOut
		return "", err
	}

	w.Close()
	os.Stdout = oldStdOut
	return <-pipeOut, nil
}

func TestFilter(t *testing.T) {
	wd, _ := os.Getwd()
	dataDir := wd + "/../../testdata"

	viper.Set("tagPatterns", []string{})
	viper.Set("resourceTypes", []string{})

	cases := []struct {
		InputFilePath    string
		InputTagFilePath string
		InputResp        rgta.GetResourcesOutput
		ExpectedFilePath string
	}{
		{
			InputFilePath:    dataDir + "/filter/input-filter.json",
			InputTagFilePath: dataDir + "/filter/input-ignore-tags.json",
			InputResp: rgta.GetResourcesOutput{
				ResourceTagMappingList: []*rgta.ResourceTagMapping{
					{
						ResourceARN: aws.String("arn:aws:ec2:us-west-2:123456789101:instance/i-0e846a0fc3863dddd"),
					}, {
						ResourceARN: aws.String("arn:aws:s3:::bucket/s3-bucket-name-1"),
					}, {
						ResourceARN: aws.String("arn:aws:autoscaling:us-west-2:123456789101:autoScalingGroup:big-long-string:autoScalingGroupName/autoscaling-group-1"),
					}, {
						ResourceARN: aws.String("arn:aws:route53:::hostedzone/HOSTEDZONEEXAMPLE1"),
					},
				},
			},
			ExpectedFilePath: dataDir + "/filter/output-filter.json",
		},
	}

	var wi bytes.Buffer
	for i, c := range cases {
		svc := &mockRGTAGetResources{
			Resp: c.InputResp,
		}

		// Parse input data
		ifb, err := ioutil.ReadFile(c.InputFilePath)
		if err != nil {
			t.Fatal("Could not read", c.InputFilePath)
		}
		wi.Write(ifb)

		// Ignore tag file
		tf, err := os.OpenFile(c.InputTagFilePath, os.O_RDWR, 0644)
		if err != nil {
			t.Fatal("Could not open", c.InputTagFilePath)
		}

		filteredOutput, err := captureFilterStdOut(filter, svc, &wi, tf)
		if err != nil {
			t.Fatal("Failed to capture stdout:", err)
		}
		filteredOutput = strings.Trim(filteredOutput, "\n")

		ob, err := ioutil.ReadFile(c.ExpectedFilePath)
		if err != nil {
			t.Fatal("Could not read from", c.ExpectedFilePath)
		}

		if !reflect.DeepEqual(strings.Trim(string(ob), "\n"), filteredOutput) {
			t.Errorf("filter test %d failed\nwanted\n%s\ngot\n%s\n", i+1, string(ob), filteredOutput)
		}

		wi.Reset()
	}
}

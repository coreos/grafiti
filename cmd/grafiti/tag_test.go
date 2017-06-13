package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	rgta "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	rgtaiface "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/coreos/grafiti/arn"
)

// Set stdout to pipe and capture printed output of a Print event
func captureRGTAStdOut(f func(rgtaiface.ResourceGroupsTaggingAPIAPI, []string, Tag) error, i rgtaiface.ResourceGroupsTaggingAPIAPI, as []string, t Tag) string {
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

	// Execute any f that takes (rgtaiface.ResourceGroupsTaggingAPIAPI, string, Tag) as arguments
	f(i, as, t)

	w.Close()
	os.Stdout = oldStdOut
	return <-pipeOut
}

// Mock API types for AWS requests
type mockTagResources struct {
	rgtaiface.ResourceGroupsTaggingAPIAPI
	Resp rgta.TagResourcesOutput
}

func (tr *mockTagResources) TagResources(in *rgta.TagResourcesInput) (*rgta.TagResourcesOutput, error) {
	return &tr.Resp, nil
}

func TestTagARNBucket(t *testing.T) {

	caseTable := []struct {
		Resp     rgta.TagResourcesOutput
		TestARNs []string
		TestTag  Tag
		Expected string
	}{
		{
			Resp: rgta.TagResourcesOutput{},
			TestARNs: []string{
				"arn:aws:ec2:us-east-1:123456789101:security-group/sg-a59ca0db",
				"arn:aws:ec2:us-east-1:123456789101:network-interface/eni-3fec2ff7",
				"arn:aws:elasticloadbalancing:us-east-1:123456789101:loadbalancer/aws-pr-780-90123456-api-internal",
				"arn:aws:ec2:us-east-1:123456789101:subnet/subnet-01188d49",
				"arn:aws:ec2:us-east-1:123456789101:vpc/vpc-aeda0dd7",
				"arn:aws:ec2:us-east-1:123456789101:internet-gateway/igw-622bab04",
			},
			TestTag: Tag{"TaggedAt", "2017-05-31"},
			Expected: fmt.Sprint(`{"ResourceARNList":[`,
				`"arn:aws:ec2:us-east-1:123456789101:security-group/sg-a59ca0db",`,
				`"arn:aws:ec2:us-east-1:123456789101:network-interface/eni-3fec2ff7",`,
				`"arn:aws:elasticloadbalancing:us-east-1:123456789101:loadbalancer/aws-pr-780-90123456-api-internal",`,
				`"arn:aws:ec2:us-east-1:123456789101:subnet/subnet-01188d49",`,
				`"arn:aws:ec2:us-east-1:123456789101:vpc/vpc-aeda0dd7",`,
				`"arn:aws:ec2:us-east-1:123456789101:internet-gateway/igw-622bab04"],`,
				`"Tags":{"TaggedAt":"2017-05-31"}}`, "\n"),
		}, {
			Resp:     rgta.TagResourcesOutput{},
			TestARNs: []string{},
			TestTag:  Tag{"TaggedAt", "2017-05-31"},
			Expected: `{"ResourceARNList":[],"Tags":{"TaggedAt":"2017-05-31"}}` + "\n",
		},
	}

	for _, c := range caseTable {
		tr := &mockTagResources{
			Resp: c.Resp,
		}

		outString := captureRGTAStdOut(tagARNBucket, tr, c.TestARNs, c.TestTag)
		if outString != c.Expected {
			t.Errorf("tagARNBucket failed\nwanted\n%s\ngot\n%s", c.Expected, outString)
		}
	}
}

// Set stdout to pipe and capture printed output of a Print event
func captureRGTAUnsupportedStdOut(f func(interface{}, string, string, Tags), i interface{}, rt, rn string, t Tags) string {
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

	// Execute any f that takes (interface{}, string, string, Tags) as arguments
	f(i, rt, rn, t)

	w.Close()
	os.Stdout = oldStdOut
	return <-pipeOut
}

// Mock AutoScaling API type for AWS requests
type mockTagAutoScalingResources struct {
	autoscalingiface.AutoScalingAPI
	Resp autoscaling.CreateOrUpdateTagsOutput
}

func (tr *mockTagAutoScalingResources) CreateOrUpdateTagsWithContext(ctx aws.Context, in *autoscaling.CreateOrUpdateTagsInput, opts ...request.Option) (*autoscaling.CreateOrUpdateTagsOutput, error) {
	return &tr.Resp, nil
}

func TestTagAutoScalingResource(t *testing.T) {

	caseTable := []struct {
		Resp      autoscaling.CreateOrUpdateTagsOutput
		InputName string
		InputTags Tags
		Expected  string
	}{
		{
			Resp:      autoscaling.CreateOrUpdateTagsOutput{},
			InputName: "demo-master",
			InputTags: map[string]string{"TaggedAt": "2017-05-31"},
			Expected: fmt.Sprint(`{"Tags":[`,
				`{"Key":"TaggedAt","PropagateAtLaunch":true,"ResourceId":"demo-master",`,
				`"ResourceType":"auto-scaling-group","Value":"2017-05-31"}]}`, "\n"),
		}, {
			Resp:      autoscaling.CreateOrUpdateTagsOutput{},
			InputName: "",
			InputTags: map[string]string{"TaggedAt": "2017-05-31"},
			Expected:  "",
		},
	}

	f := func(v interface{}, rt, arn string, t Tags) {
		tagAutoScalingResource(v.(autoscalingiface.AutoScalingAPI), rt, arn, t)
	}

	for _, c := range caseTable {
		tr := &mockTagAutoScalingResources{
			Resp: c.Resp,
		}

		outString := captureRGTAUnsupportedStdOut(f, tr, arn.AutoScalingGroupRType, c.InputName, c.InputTags)
		if outString != c.Expected {
			t.Errorf("tagAutoScalingResource failed\nwanted\n%s\ngot\n%s", c.Expected, outString)
		}
	}
}

// Mock Route53 API type for AWS requests
type mockRoute53TagResources struct {
	route53iface.Route53API
	Resp route53.ChangeTagsForResourceOutput
}

func (tr *mockRoute53TagResources) ChangeTagsForResourceWithContext(ctx aws.Context, in *route53.ChangeTagsForResourceInput, opts ...request.Option) (*route53.ChangeTagsForResourceOutput, error) {
	return &tr.Resp, nil
}

func TestTagRoute53Resource(t *testing.T) {

	caseTable := []struct {
		Resp      route53.ChangeTagsForResourceOutput
		InputName string
		InputTags Tags
		Expected  string
	}{
		{
			Resp:      route53.ChangeTagsForResourceOutput{},
			InputName: "Z148QEXAMPLE8V",
			InputTags: map[string]string{"TaggedAt": "2017-05-31"},
			Expected: fmt.Sprint(`{"AddTags":[{"Key":"TaggedAt","Value":"2017-05-31"}],`,
				`"RemoveTagKeys":null,"ResourceId":"Z148QEXAMPLE8V","ResourceType":"hostedzone"}`,
				"\n"),
		}, {
			Resp:      route53.ChangeTagsForResourceOutput{},
			InputName: "",
			InputTags: map[string]string{"TaggedAt": "2017-05-31"},
			Expected:  "",
		},
	}

	f := func(v interface{}, rt, arn string, t Tags) {
		tagRoute53Resource(v.(route53iface.Route53API), rt, arn, t)
	}

	for _, c := range caseTable {
		tr := &mockRoute53TagResources{
			Resp: c.Resp,
		}

		outString := captureRGTAUnsupportedStdOut(f, tr, arn.Route53HostedZoneRType, c.InputName, c.InputTags)
		if outString != c.Expected {
			t.Errorf("TestTagRoute53Resource failed\nwanted\n%s\ngot\n%s", c.Expected, outString)
		}
	}
}

func TestDecodeInput(t *testing.T) {
	wd, _ := os.Getwd()
	dataDir := wd + "/../../testdata"

	cases := []struct {
		InputFilePath string
		Expected      []TagInput
	}{
		{
			InputFilePath: dataDir + "/tag/test-data-input.json",
			Expected: []TagInput{
				{
					TaggingMetadata: TaggingMetadata{
						ResourceName: "demo-master",
						ResourceType: arn.AutoScalingGroupRType,
						ResourceARN:  "arn:aws:autoscaling:us-west-2:123456789101:autoScalingGroup:big-long-string:autoScalingGroupName/demo-master",
						CreatorARN:   "arn:aws:iam::123456789101:user/test-user",
						CreatorName:  "test-user",
					},
					Tags: map[string]string{
						"CreatedBy": "arn:aws:iam::123456789101:user/test-user",
						"ExpiresAt": "2017-06-12",
						"TaggedAt":  "2017-05-31",
					},
				}, {
					TaggingMetadata: TaggingMetadata{
						ResourceName: "i-0e846a0fc386398df",
						ResourceType: arn.EC2InstanceRType,
						ResourceARN:  "arn:aws:ec2:us-west-2:123456789101:instance/i-0e846a0fc386398df",
						CreatorARN:   "arn:aws:iam::123456789101:user/test-user",
						CreatorName:  "test-user",
					},
					Tags: map[string]string{
						"CreatedBy": "arn:aws:iam::123456789101:user/test-user",
						"ExpiresAt": "2017-06-12",
						"TaggedAt":  "2017-05-31",
					},
				}, {
					TaggingMetadata: TaggingMetadata{
						ResourceName: "ZHZDDDD1GKNAC",
						ResourceType: arn.Route53HostedZoneRType,
						ResourceARN:  "arn:aws:route53:::hostedzone/ZHZDDDD1GKNAC",
						CreatorARN:   "",
						CreatorName:  "",
					},
					Tags: map[string]string{
						"CreatedBy": "arn:aws:iam::123456789101:user/test-user",
						"ExpiresAt": "2017-06-12",
						"TaggedAt":  "2017-05-31",
					},
				},
			},
		},
		{
			InputFilePath: dataDir + "/tag/test-data-input-no-tags.json",
			Expected: []TagInput{
				{
					TaggingMetadata: TaggingMetadata{
						ResourceName: "demo-master",
						ResourceType: arn.AutoScalingGroupRType,
						ResourceARN:  "arn:aws:autoscaling:us-west-2:123456789101:autoScalingGroup:big-long-string:autoScalingGroupName/demo-master",
						CreatorARN:   "arn:aws:iam::123456789101:user/test-user",
						CreatorName:  "test-user",
					},
					Tags: map[string]string{},
				}, {
					TaggingMetadata: TaggingMetadata{
						ResourceName: "i-0e846a0fc386398df",
						ResourceType: arn.EC2InstanceRType,
						ResourceARN:  "arn:aws:ec2:us-west-2:123456789101:instance/i-0e846a0fc386398df",
						CreatorARN:   "arn:aws:iam::123456789101:user/test-user",
						CreatorName:  "test-user",
					},
					Tags: map[string]string{},
				}, {
					TaggingMetadata: TaggingMetadata{
						ResourceName: "ZHZDDDD1GKNAC",
						ResourceType: arn.Route53HostedZoneRType,
						ResourceARN:  "arn:aws:route53:::hostedzone/ZHZDDDD1GKNAC",
						CreatorARN:   "",
						CreatorName:  "",
					},
					Tags: map[string]string{},
				},
			},
		},
	}

	for _, c := range cases {
		f, ferr := os.Open(c.InputFilePath)
		if ferr != nil {
			t.Fatal("Failed to open", c.InputFilePath)
		}
		defer f.Close()

		dec := json.NewDecoder(bufio.NewReader(f))
		for i := 0; ; i++ {
			ti, isEOF, derr := decodeInput(dec)
			if derr != nil {
				t.Fatal("Failed to decode:", derr.Error())
			}
			if isEOF {
				break
			}

			if !reflect.DeepEqual(ti.TaggingMetadata, c.Expected[i].TaggingMetadata) {
				t.Errorf("decodeInput failed\nwanted\n%s\ngot\n%s\n", c.Expected[i].TaggingMetadata, ti.TaggingMetadata)
			}
			if !reflect.DeepEqual(ti.Tags, c.Expected[i].Tags) {
				t.Errorf("decodeInput failed\nwanted\n%s\ngot\n%s\n", c.Expected[i].Tags, ti.Tags)
			}
		}
	}
}

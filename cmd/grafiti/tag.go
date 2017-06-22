// Copyright © 2017 grafiti/predator authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	rgta "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	rgtaiface "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/coreos/grafiti/arn"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tagFile string

// TaggingMetadata is the data required to find and tag a resource
type TaggingMetadata struct {
	ResourceName arn.ResourceName
	ResourceType arn.ResourceType
	ResourceARN  arn.ResourceARN
	CreatorARN   arn.ResourceARN
	CreatorName  arn.ResourceName
}

// Tags are an alias for mapping Tag.Key -> Tag.Value
type Tags map[string]string

// Tag holds user-defined tag values from a TOML file
type Tag struct {
	Key   string
	Value string
}

// TagInput holds all Data describing a resource
type TagInput struct {
	TaggingMetadata TaggingMetadata
	Tags            Tags
}

// ARNSet maps an ARN to a general struct
type ARNSet map[arn.ResourceARN]struct{}

// NewARNSet creates a new ARNSet
func NewARNSet() ARNSet {
	return make(map[arn.ResourceARN]struct{})
}

// AddARN adds an empty struct to an ARNSet
func (a *ARNSet) AddARN(ARN arn.ResourceARN) {
	(*a)[ARN] = struct{}{}
}

// ToList generates a list of ARNSet's from a map
func (a *ARNSet) ToList() arn.ResourceARNs {
	var arnList = make(arn.ResourceARNs, 0, len(*a))
	for k := range *a {
		arnList = append(arnList, k)
	}
	return arnList
}

// ARNSetBucket maps a tag to a set of ARN strings
type ARNSetBucket map[Tag]ARNSet

// NewARNSetBucket creates a new map of Tag -> ARNSet
func NewARNSetBucket() ARNSetBucket {
	return make(map[Tag]ARNSet)
}

// AddARNToBuckets adds tags to an ARNSet, or creates a new set if one does not
// exist
func (b *ARNSetBucket) AddARNToBuckets(ARN arn.ResourceARN, tags map[string]string) {
	if ARN == "" {
		return
	}
	for tagKey, tagValue := range tags {
		tag := Tag{tagKey, tagValue}
		arnSet, found := (*b)[tag]
		if !found {
			arnSet = NewARNSet()
			(*b)[tag] = arnSet
		}
		arnSet.AddARN(ARN)
	}
}

// ClearBucket creates a new ARNSet for a specific Tag
func (b *ARNSetBucket) ClearBucket(bucket Tag) {
	(*b)[bucket] = NewARNSet()
}

func init() {
	RootCmd.AddCommand(tagCmd)
	tagCmd.PersistentFlags().StringVarP(&tagFile, "tag-file", "f", "", "File containing JSON objects of taggable resource ARN's and tag key/value pairs. Format is the output format of grafiti parse.")
}

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Tag resources in AWS",
	Long:  "Tag resources in AWS. By default, talks to the configured aws account and reads directly from CloudTrail.",
	RunE:  runTagCommand,
}

func runTagCommand(cmd *cobra.Command, args []string) error {
	if tagFile != "" {
		return tagFromFile(tagFile)
	}
	if err := tagFromStdIn(); err != nil {
		return err
	}
	return nil
}

func tagFromFile(fname string) error {
	file, err := os.Open(fname)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(file)
	return tag(reader)
}

func tagFromStdIn() error {
	return tag(os.Stdin)
}

func tag(reader io.Reader) error {
	svc := rgta.New(session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(viper.GetString("grafiti.region")),
		},
	)))
	dec := json.NewDecoder(reader)

	ARNBuckets := NewARNSetBucket()
	otherResBuckets := make(map[arn.ResourceType]arn.ResourceNames)

	for {
		t, isEOF, err := decodeInput(dec)
		if err != nil {
			return err
		}
		if t == nil {
			continue
		}

		// AutoScaling and Route53 resources have their own tagging API's
		tm := t.TaggingMetadata
		if _, ok := arn.RGTAUnsupportedResourceTypes[tm.ResourceType]; ok {
			if _, rok := otherResBuckets[tm.ResourceType]; !rok {
				otherResBuckets[tm.ResourceType] = append(otherResBuckets[tm.ResourceType], tm.ResourceName)
			}
		} else {
			ARNBuckets.AddARNToBuckets(tm.ResourceARN, t.Tags)
		}

		for tag, bucket := range ARNBuckets {
			if len(bucket) == 20 || isEOF {
				if err := tagARNBucket(svc, bucket.ToList(), tag); err != nil {
					return err
				}
				ARNBuckets.ClearBucket(tag)
			}
		}

		if len(otherResBuckets[tm.ResourceType]) == 20 || isEOF {
			tagUnsupportedResourceType(tm.ResourceType, otherResBuckets[tm.ResourceType], t.Tags)
			delete(otherResBuckets, tm.ResourceType)
		}

		if isEOF {
			break
		}
	}

	return nil
}

func tagUnsupportedResourceType(rt arn.ResourceType, rns arn.ResourceNames, tags Tags) {
	if len(rns) == 0 {
		return
	}

	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(viper.GetString("grafiti.region")),
		},
	))

	switch arn.NamespaceForResource(rt) {
	case arn.AutoScalingNamespace:
		svc := autoscaling.New(sess)
		for _, n := range rns {
			tagAutoScalingResource(svc, rt, n, tags)
		}
	case arn.Route53Namespace:
		svc := route53.New(sess)
		for _, n := range rns {
			tagRoute53Resource(svc, rt, n, tags)
		}
	}
	return
}

func tagAutoScalingResource(svc autoscalingiface.AutoScalingAPI, rt arn.ResourceType, rn arn.ResourceName, tags Tags) {
	// Only AutoScaling Groups support tagging
	if rt != arn.AutoScalingGroupRType || rn == "" {
		return
	}

	asTags := make([]*autoscaling.Tag, 0, len(tags))
	for tk, tv := range tags {
		asTags = append(asTags, &autoscaling.Tag{
			Key:               aws.String(tk),
			Value:             aws.String(tv),
			ResourceType:      aws.String("auto-scaling-group"),
			ResourceId:        rn.AWSString(),
			PropagateAtLaunch: aws.Bool(true),
		})
	}

	params := &autoscaling.CreateOrUpdateTagsInput{
		Tags: asTags,
	}

	ctx := aws.BackgroundContext()
	if _, err := svc.CreateOrUpdateTagsWithContext(ctx, params); err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
	} else {
		pj, _ := json.Marshal(params)
		fmt.Println(string(pj))
	}

	return
}

func tagRoute53Resource(svc route53iface.Route53API, rt arn.ResourceType, rn arn.ResourceName, tags Tags) {
	// Only hostedzones (and healthchecks, but they are not supported yet) allow
	// tagging
	if rt != arn.Route53HostedZoneRType || rn == "" {
		return
	}

	hzTags := make([]*route53.Tag, 0, len(tags))
	for tk, tv := range tags {
		hzTags = append(hzTags, &route53.Tag{
			Key:   aws.String(tk),
			Value: aws.String(tv),
		})
	}

	params := &route53.ChangeTagsForResourceInput{
		AddTags:      hzTags,
		ResourceId:   rn.AWSString(),
		ResourceType: aws.String("hostedzone"),
	}

	ctx := aws.BackgroundContext()
	if _, err := svc.ChangeTagsForResourceWithContext(ctx, params); err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
	} else {
		pj, _ := json.Marshal(params)
		fmt.Println(string(pj))
	}

	return
}

func tagARNBucket(svc rgtaiface.ResourceGroupsTaggingAPIAPI, bucket arn.ResourceARNs, tag Tag) error {
	params := &rgta.TagResourcesInput{
		ResourceARNList: bucket.AWSStringSlice(),
		Tags:            map[string]*string{tag.Key: aws.String(tag.Value)},
	}

	paramsJSON, _ := json.Marshal(params)
	fmt.Println(string(paramsJSON))

	if dryRun {
		return nil
	}
	// Rate limit error is returned if no pause between requests
	time.Sleep(time.Duration(2) * time.Second)
	if _, err := svc.TagResources(params); err != nil {
		if ignoreErrors {
			fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
			return nil
		}
		return err

	}
	return nil
}

func decodeInput(decoder *json.Decoder) (*TagInput, bool, error) {
	var decoded TagInput
	if err := decoder.Decode(&decoded); err != nil {
		if err == io.EOF {
			return &decoded, true, nil
		}
		if ignoreErrors {
			fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
			return nil, false, nil
		}
		return nil, false, err
	}
	return &decoded, false, nil
}

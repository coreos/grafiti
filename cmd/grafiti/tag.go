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
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	rgta "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/coreos/grafiti/arn"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	tagFile  string
	tagsOnly bool
)

// TaggingMetadata is the data required to find and tag a resource
type TaggingMetadata struct {
	ResourceName string
	ResourceType string
	ResourceARN  string
	CreatorARN   string
	CreatorName  string
	CreatedAt    string
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
type ARNSet map[string]struct{}

// NewARNSet creates a new ARNSet
func NewARNSet() ARNSet {
	return make(map[string]struct{})
}

// AddARN adds an empty struct to an ARNSet
func (a *ARNSet) AddARN(ARN string) {
	(*a)[ARN] = struct{}{}
}

// ToList generates a list of ARNSet's from a map
func (a *ARNSet) ToList() []string {
	var arnList = make([]string, 0, len(*a))
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
func (b *ARNSetBucket) AddARNToBuckets(ARN string, tags map[string]string) {
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
	tagCmd.PersistentFlags().StringVarP(&tagFile, "log-data-file", "f", "", "CloudTrail Log input file.")
	tagCmd.PersistentFlags().BoolVarP(&tagsOnly, "tags-only", "t", false, "Only print a JSON list of tags used.")
}

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "tag resources in AWS",
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

type tagFilterOutput struct {
	TagFilters []tagOutput
}

type tagOutput struct {
	Key    string
	Values []string
}

func tag(reader io.Reader) error {
	region := viper.GetString("grafiti.az")
	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(region),
		},
	))
	svc := rgta.New(sess)
	dec := json.NewDecoder(reader)

	ARNBuckets := NewARNSetBucket()

	allTags := make(Tags)

	for {
		t, isEOF, err := decodeInput(dec)
		if err != nil {
			return err
		}
		if t == nil {
			continue
		}

		resARN := t.TaggingMetadata.ResourceARN

		// AutoScalingGroups have their own tagging API that must be used
		if strings.HasPrefix(resARN, "arn:aws:autoscaling") {
			tagAutoScalingGroup(resARN, t.Tags)
		} else {
			ARNBuckets.AddARNToBuckets(resARN, t.Tags)
		}

		if tagsOnly {
			for tk, tv := range t.Tags {
				allTags[tk] = tv
			}
		}

		for tag, bucket := range ARNBuckets {
			if len(bucket) == 20 || isEOF {
				if err := tagARNBucket(svc, bucket.ToList(), tag); err != nil {
					return err
				}
				ARNBuckets.ClearBucket(tag)
			}
		}

		if isEOF {
			break
		}
	}

	if tagsOnly {
		tags := make([]tagOutput, 0, len(allTags))
		for tk, tv := range allTags {
			tags = append(tags, tagOutput{tk, []string{tv}})
		}

		tagsJSON, err := json.Marshal(tagFilterOutput{tags})
		if err != nil {
			return err
		}
		fmt.Println(string(tagsJSON))
	}

	return nil
}

func tagAutoScalingGroup(ARN string, tags Tags) {
	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(viper.GetString("grafiti.az")),
		},
	))
	svc := autoscaling.New(sess)

	rType, rName := arn.MapARNToRTypeAndRName(ARN)

	if rType != arn.AutoScalingGroupRType {
		return
	}

	asTags := make([]*autoscaling.Tag, 0, len(tags))
	// Only AutoScaling Groups support tagging
	for tk, tv := range tags {
		tag := autoscaling.Tag{
			Key:               aws.String(tk),
			Value:             aws.String(tv),
			ResourceType:      aws.String("auto-scaling-group"),
			ResourceId:        aws.String(rName),
			PropagateAtLaunch: aws.Bool(true),
		}
		tj, _ := json.Marshal(tag)
		if !tagsOnly {
			fmt.Println(string(tj))
		}
		asTags = append(asTags, &tag)
	}

	params := &autoscaling.CreateOrUpdateTagsInput{
		Tags: asTags,
	}

	ctx := aws.BackgroundContext()
	_, err := svc.CreateOrUpdateTagsWithContext(ctx, params)
	if err != nil {
		fmt.Println(err.Error())
		// fmt.Println("Failed to tag", ARN)
	}
	return
}

func tagARNBucket(svc *rgta.ResourceGroupsTaggingAPI, bucket []string, tag Tag) error {
	params := &rgta.TagResourcesInput{
		ResourceARNList: aws.StringSlice(bucket),
		Tags:            map[string]*string{tag.Key: aws.String(tag.Value)},
	}
	paramsJSON, _ := json.Marshal(params)
	if !tagsOnly {
		fmt.Println(string(paramsJSON))
	}
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

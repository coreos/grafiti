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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tagFile string

// TaggingMetadata is the data required to find and tag a resource
type TaggingMetadata struct {
	ResourceName string
	ResourceType string
	ResourceARN  string
	CreatorARN   string
	CreatorName  string
}

type Tags map[string]string

type Tag struct {
	Key   string
	Value string
}

type TagInput struct {
	TaggingMetadata TaggingMetadata
	Tags            Tags
}

type ARNSet map[string]struct{}

func NewARNSet() ARNSet {
	return make(map[string]struct{})
}

func (a *ARNSet) AddARN(ARN string) {
	(*a)[ARN] = struct{}{}
}

func (a *ARNSet) ToList() []string {
	var arnList = make([]string, 0, len(*a))
	for k, _ := range *a {
		arnList = append(arnList, k)
	}
	return arnList
}

//ARNSetBucket maps a tag to a set of ARN strings
type ARNSetBucket map[Tag]ARNSet

func NewARNSetBucket() ARNSetBucket {
	return make(map[Tag]ARNSet)
}

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

func (b *ARNSetBucket) ClearBucket(bucket Tag) {
	(*b)[bucket] = NewARNSet()
}

func init() {
	RootCmd.AddCommand(tagCmd)
	tagCmd.PersistentFlags().StringVarP(&tagFile, "tagFile", "t", "", "CloudTrail Log input")
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

func tagFromFile(tagFile string) error {
	file, err := os.Open(tagFile)
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
	region := viper.GetString("grafiti.az")
	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(region),
		},
	))
	svc := resourcegroupstaggingapi.New(sess)
	dec := json.NewDecoder(reader)

	ARNBuckets := NewARNSetBucket()

	for {
		t, isEOF, err := decodeInput(dec)
		if err != nil {
			return err
		}
		if t == nil {
			continue
		}
		ARNBuckets.AddARNToBuckets(t.TaggingMetadata.ResourceARN, t.Tags)

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

	return nil
}

func tagARNBucket(svc *resourcegroupstaggingapi.ResourceGroupsTaggingAPI, bucket []string, tag Tag) error {
	params := &resourcegroupstaggingapi.TagResourcesInput{
		ResourceARNList: aws.StringSlice(bucket),
		Tags:            map[string]*string{tag.Key: aws.String(tag.Value)},
	}
	paramsJson, _ := json.Marshal(params)
	fmt.Println(string(paramsJson))
	if dryRun {
		return nil
	}
	if _, err := svc.TagResources(params); err != nil {
		if ignoreErrors {
			fmt.Printf(`{"error": "%s"}\n`, err.Error())
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
			fmt.Printf(`{"error": "%s"}\n`, err.Error())
			return nil, false, nil
		}
		return nil, false, err
	}
	return &decoded, false, nil
}

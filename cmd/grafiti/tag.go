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
	"encoding/json"
	"os"

	"io"

	"fmt"

	"bufio"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tagFile string
var ignoreErrors bool

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

type ARNBucketSet map[Tag][]string

func NewARNBucketSet() ARNBucketSet {
	return make(map[Tag][]string)
}

func (b *ARNBucketSet) AddARNToBuckets(ARN string, tags map[string]string) {
	if ARN == "" {
		return
	}
	for tagKey, tagValue := range tags {
		tag := Tag{tagKey, tagValue}
		if _, found := (*b)[tag]; found {
			continue
		}
		(*b)[tag] = append((*b)[tag], ARN)
	}
}

func (b *ARNBucketSet) ClearBucket(bucket Tag) {
	(*b)[bucket] = []string{}
}

func init() {
	RootCmd.AddCommand(tagCmd)
	tagCmd.PersistentFlags().StringVarP(&tagFile, "tagFile", "t", "", "CloudTrail Log input")
	tagCmd.PersistentFlags().BoolVarP(&ignoreErrors, "ignoreErrors", "e", false, "Continue processing even when there are API errors when tagging.")
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

	ARNBuckets := NewARNBucketSet()

	for {
		var t TagInput
		isEOF, err := decodeInput(&t, dec)
		if err != nil {
			return err
		}

		ARNBuckets.AddARNToBuckets(t.TaggingMetadata.ResourceARN, t.Tags)

		for tag, bucket := range ARNBuckets {
			if len(bucket) == 20 || isEOF {
				if err := tagARNBucket(svc, bucket, tag); err != nil {
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

func decodeInput(decoded *TagInput, decoder *json.Decoder) (bool, error) {
	if err := decoder.Decode(&decoded); err != nil {
		if err == io.EOF {
			return true, nil
		}
		if ignoreErrors {
			fmt.Printf(`{"error": "%s"}\n`, err.Error())
			return false, nil
		}
		return false, err
	}
	return false, nil
}

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

type TagInput struct {
	TaggingMetadata TaggingMetadata
	Tags            map[string]*string
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
	for {
		var t TagInput
		if err := dec.Decode(&t); err != nil {
			if err == io.EOF {
				break
			}
			if ignoreErrors {
				fmt.Printf("Ignoring error: %s\n", err.Error())
				continue
			}
			return err
		}

		//TODO: slurp up input
		params := &resourcegroupstaggingapi.TagResourcesInput{
			ResourceARNList: []*string{aws.String(t.TaggingMetadata.ResourceARN)},
			Tags:            t.Tags,
		}

		if _, err := svc.TagResources(params); err != nil {
			return err
		}
	}

	return nil
}

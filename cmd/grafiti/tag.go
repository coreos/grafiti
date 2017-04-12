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
	"strings"

	"encoding/json"
	"os"

	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"fmt"
)

var tagFile string

func init() {
	RootCmd.AddCommand(tagCmd)
	parseCmd.PersistentFlags().StringVarP(&tagFile, "tagFile", "t", "", "CloudTrail Log input")
}

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "tag resources in AWS",
	Long:  "Tag resources in AWS. By default, talks to the configured aws account and reads directly from CloudTrail.",
	RunE:  runTagCommand,
}

func runTagCommand(cmd *cobra.Command, args []string) error {
	//if tagFile != "" {
	//	return tagFromFile(tagFile)
	//}
	if err := tagFromStdIn(); err != nil {
		return err
	}
	return nil
}

type TagInput struct {
	Resource cloudtrail.Resource
	Tags     map[string]string
}

func tagFromStdIn() error {
	region := viper.GetString("grafiti.az")
	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(region),
		},
	))

	dec := json.NewDecoder(os.Stdin)
	for {
		var t TagInput
		if err := dec.Decode(&t); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if err := TagByResourceType(sess, t.Tags, *t.Resource.ResourceName, *t.Resource.ResourceType); err != nil {
			return err
		}
	}

	return nil
}

func TagByResourceType(sess *session.Session, tags map[string]string, resourceName, resourceType string) error {
	switch {
	case strings.HasPrefix(resourceType, "AWS::EC2::"):
		svc := ec2.New(sess)

		var awsTags []*ec2.Tag
		for k, v := range tags {
			awsTags = append(awsTags, &ec2.Tag{Key: aws.String(k), Value: aws.String(v)})
		}

		params := &ec2.CreateTagsInput{
			Resources: []*string{
				aws.String(resourceName),
			},
			Tags:   awsTags,
			DryRun: aws.Bool(dryRun),
		}
		if _, err := svc.CreateTags(params); err != nil {
			return err
		}
		fmt.Printf("Tagging EC2 Instance %s\n", resourceName)

		return nil
	case strings.HasPrefix(resourceType, "AWS::AutoScaling::"):
		return nil
	case strings.HasPrefix(resourceType, "AWS::ACM::"):
		return nil
	case strings.HasPrefix(resourceType, "AWS::CloudTrail::"):
		return nil
	case strings.HasPrefix(resourceType, "AWS::CodePipeline::"):
		return nil
	case strings.HasPrefix(resourceType, "AWS::ElasticLoadBalancing::"):
		return nil
	case strings.HasPrefix(resourceType, "AWS::IAM::"):
		return nil
	case strings.HasPrefix(resourceType, "AWS::Redshift::"):
		return nil
	case strings.HasPrefix(resourceType, "AWS::RDS::"):
		return nil
	case strings.HasPrefix(resourceType, "AWS::S3::"):
		return nil
	}
	return nil
}

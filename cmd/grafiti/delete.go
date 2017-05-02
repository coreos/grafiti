// Copyright © 2017 grafiti authors
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
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/coreos/grafiti/arn"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deleteFile string

type DeleteInput struct {
	TagFilters []*resourcegroupstaggingapi.TagFilter
}

func init() {
	RootCmd.AddCommand(deleteCmd)
	tagCmd.PersistentFlags().StringVarP(&tagFile, "deleteFile", "i", "", "ARNs that should be deleted")
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete resources in AWS",
	Long:  "Delete resources in AWS. Uses the configured delete filters to decide which resources to delete.",
	RunE:  runDeleteCommand,
}

func runDeleteCommand(cmd *cobra.Command, args []string) error {
	if tagFile != "" {
		return deleteFromFile(deleteFile)
	}
	if err := deleteFromStdIn(); err != nil {
		return err
	}
	return nil
}
func deleteFromFile(tagFile string) error {
	file, err := os.Open(tagFile)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(file)
	return delete(reader)
}

func deleteFromStdIn() error {
	return delete(os.Stdin)
}

func delete(reader io.Reader) error {
	region := viper.GetString("grafiti.az")
	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(region),
		},
	))
	svc := resourcegroupstaggingapi.New(sess)
	dec := json.NewDecoder(reader)

	for {
		t, isEOF, err := decodeDeleteInput(dec)
		if err != nil {
			return err
		}
		if t == nil {
			continue
		}
		// get ARNs of matching tags
		params := &resourcegroupstaggingapi.GetResourcesInput{
			PaginationToken: nil,
			ResourceTypeFilters: []*string{
				aws.String(arn.ServiceNameForResource(viper.GetString("grafiti.resourceType"))),
			},
			TagFilters:  t.TagFilters,
			TagsPerPage: aws.Int64(100),
		}

		for {
			// Get batch of matching resources
			req, resp := svc.GetResourcesRequest(params)
			if err := req.Send(); err != nil {
				return err
			}
			arns := make([]string, len(resp.ResourceTagMappingList))
			for _, r := range resp.ResourceTagMappingList {
				if r.ResourceARN != nil && *r.ResourceARN != "" {
					arns = append(arns, *r.ResourceARN)
				}
			}
			if resp.PaginationToken == nil {
				break
			}
			params.PaginationToken = resp.PaginationToken

			if len(arns) == 0 {
				fmt.Println("No resources match the specified tag filters")
				return nil
			}

			fmt.Println(arns)

			// Delete batch of matching resources
			if err := deleteARNs(arns); err != nil {
				return err
			}
		}

		if isEOF {
			break
		}
	}

	return nil
}

func deleteARNs(ARNs []string) error {
	switch viper.GetString("grafiti.resourceType") {
	case "AWS::EC2::Instance":
		return deleteEC2Instances(ARNs)
	}
	fmt.Println("ResourceType not yet supported")
	return nil
}

func deleteEC2Instances(ARNs []string) error {
	instanceIDs := make([]*string, len(ARNs))
	for _, a := range ARNs {
		id := arn.InstanceIDFromARN(a)
		if id != "" {
			instanceIDs = append(instanceIDs, aws.String(id))
		}
	}

	region := viper.GetString("grafiti.az")
	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(region),
		},
	))
	svc := ec2.New(sess)
	params := &ec2.TerminateInstancesInput{
		InstanceIds: instanceIDs,
		DryRun:      aws.Bool(dryRun),
	}
	_, err := svc.TerminateInstances(params)

	if err != nil {
		if ignoreErrors {
			fmt.Printf(`{"error": "%s"}\n`, err.Error())
			return nil
		}
		return err
	}
	return nil
}

func decodeDeleteInput(decoder *json.Decoder) (*DeleteInput, bool, error) {
	var decoded DeleteInput
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

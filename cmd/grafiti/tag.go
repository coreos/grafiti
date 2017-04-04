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
	"fmt"
	"io/ioutil"
	"os"

	"text/tabwriter"

	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/spf13/cobra"
)

var inputFile string
var bucket string

func init() {
	RootCmd.AddCommand(tagCmd)
	tagCmd.PersistentFlags().StringVarP(&inputFile, "inputFile", "i", "", "CloudTrail Log input")
	tagCmd.PersistentFlags().StringVarP(&bucket, "bucket", "b", "", "S3 bucket with CloudTrail Logs")
}

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "tag resources by reading CloudTrail logs",
	Long:  "Parse a CloudTrail Log and tag resources. By default, talks to the configured aws account and reads directly from CloudTrail.",
	RunE:  runTagCommand,
}

func runTagCommand(cmd *cobra.Command, args []string) error {
	if inputFile != "" {
		return tagFromFile(inputFile)
	}
	if bucket != "" {
		return fmt.Errorf("bucket support not implemented")
	}
	if err := tagFromCloudTrail(); err != nil {
		return err
	}
	return nil
}

type CloudTrailLogFile struct {
	Events []*cloudtrail.Event `json:"Records"`
}

func tagFromFile(logFileName string) error {
	raw, err := ioutil.ReadFile(logFileName)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	var logFile CloudTrailLogFile
	if err := json.Unmarshal(raw, &logFile); err != nil {
		return err
	}
	printEvents(logFile.Events)

	//TODO: actually tag events
	return nil
}

func tagFromCloudTrail() error {
	// TODO: arguments for this, cleanup
	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String("us-east-1"),
		},
	))
	svc := cloudtrail.New(sess)
	params := &cloudtrail.LookupEventsInput{
		EndTime: aws.Time(time.Now()),
		LookupAttributes: []*cloudtrail.LookupAttribute{
			{
				AttributeKey:   aws.String("ResourceType"),
				AttributeValue: aws.String("AWS::EC2::Instance"),
			},
		},
		MaxResults: aws.Int64(50),
		StartTime:  aws.Time(time.Now().AddDate(0, 0, -1)),
	}

	var events []*cloudtrail.Event
	req, resp := svc.LookupEventsRequest(params)
	if err := req.Send(); err != nil {
		return err
	}
	events = append(events, resp.Events...)

	// Loop through pages
	for resp.NextToken != nil {
		params.NextToken = resp.NextToken
		req, resp := svc.LookupEventsRequest(params)
		if err := req.Send(); err != nil {
			return err
		}
		events = append(events, resp.Events...)
		fmt.Println("fetching next batch...")
	}
	printEvents(events)

	//TODO: actually tag stuff
	return nil
}

func printEvents(events []*cloudtrail.Event) {
	w := tabwriter.NewWriter(os.Stdout, 8, 8, 8, ' ', 0)
	for _, e := range events {
		fmt.Println(e)
		for _, r := range e.Resources {
			fmt.Fprintf(w, "%s\t%s\t%s\t\n", *r.ResourceName, *r.ResourceType, *e.Username)
		}
	}
	w.Flush()
}

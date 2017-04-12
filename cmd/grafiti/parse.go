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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var inputFile string

func init() {
	RootCmd.AddCommand(parseCmd)
	parseCmd.PersistentFlags().StringVarP(&inputFile, "inputFile", "i", "", "CloudTrail Log input")
}

var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "parse and output resources by reading CloudTrail logs",
	Long:  "Parse a CloudTrail Log and output resources. By default, talks to the configured aws account and reads directly from CloudTrail.",
	RunE:  runParseCommand,
}

func runParseCommand(cmd *cobra.Command, args []string) error {
	if inputFile != "" {
		return parseFromFile(inputFile)
	}
	if err := parseFromCloudTrail(); err != nil {
		return err
	}
	return nil
}

type CloudTrailLogFile struct {
	Events []*cloudtrail.Event `json:"Records"`
}

func parseFromFile(logFileName string) error {
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

	return nil
}

func parseFromCloudTrail() error {
	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(viper.GetString("grafiti.az")),
		},
	))
	svc := cloudtrail.New(sess)
	params := &cloudtrail.LookupEventsInput{
		EndTime: aws.Time(time.Now()),
		LookupAttributes: []*cloudtrail.LookupAttribute{
			{
				AttributeKey:   aws.String("ResourceType"),
				AttributeValue: aws.String(viper.GetString("grafiti.resourceType")),
			},
		},
		MaxResults: aws.Int64(50),
		StartTime:  aws.Time(time.Now().Add(time.Duration(viper.GetInt("grafiti.hours")) * time.Hour)),
	}

	for {
		req, resp := svc.LookupEventsRequest(params)
		if err := req.Send(); err != nil {
			return err
		}
		printEvents(resp.Events)
		if resp.NextToken == nil {
			break
		}
		params.NextToken = resp.NextToken
	}

	return nil
}

type Output struct {
	Event *cloudtrail.Event
	cloudtrail.Resource
}

func printEvents(events []*cloudtrail.Event) {
	for _, e := range events {
		printEvent(e)
	}
}

func printEvent(event *cloudtrail.Event) {
	output := Output{
		Event: event,
	}
	resourceJson, err := json.Marshal(output)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println(string(resourceJson))
}

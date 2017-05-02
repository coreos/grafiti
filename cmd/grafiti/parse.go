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
	jq "github.com/threatgrid/jqpipe-go"
	"github.com/tidwall/gjson"

	"github.com/coreos/grafiti/arn"
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

type NotTaggedFilter struct {
	Type string `json:"type"`
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

type OutputWithEvent struct {
	Event           *cloudtrail.Event
	TaggingMetadata *TaggingMetadata
	Tags            map[string]string
}

type Output struct {
	TaggingMetadata *TaggingMetadata
	Tags            map[string]string
}

func filterOutput(output []byte) []byte {
	for _, filter := range viper.GetStringSlice("grafiti.filterPatterns") {
		results, err := jq.Eval(string(output), filter)
		if err != nil {
			return nil
		}
		if len(results) == 0 {
			return nil
		}
		json, _ := results[0].MarshalJSON()
		if string(json) != "true" {
			return nil
		}
	}
	return output
}

func printEvents(events []*cloudtrail.Event) {
	for _, e := range events {
		parsedEvent := gjson.Parse(*e.CloudTrailEvent)
		printEvent(e, parsedEvent)
	}
}

func printEvent(event *cloudtrail.Event, parsedEvent gjson.Result) {
	includeEvent := viper.GetBool("grafiti.includeEvent")
	for _, r := range event.Resources {
		tags := getTags(event)
		output := getOutput(includeEvent, event, tags, &TaggingMetadata{
			ResourceName: *r.ResourceName,
			ResourceType: *r.ResourceType,
			ResourceARN:  arn.ARNForResource(r, parsedEvent),
			CreatorARN:   parsedEvent.Get("userIdentity.arn").Str,
			CreatorName:  parsedEvent.Get("userIdentity.userName").Str,
		})
		resourceJson, err := json.Marshal(output)
		resourceJson = filterOutput(resourceJson)
		if resourceJson == nil {
			continue
		}
		if err != nil {
			fmt.Println(fmt.Sprintf(`{"error": "%s"}`, err))
		}
		fmt.Println(string(resourceJson))
	}
}

func getTags(event *cloudtrail.Event) map[string]string {
	allTags := make(map[string]string)
	for _, tagPattern := range viper.GetStringSlice("grafiti.tagPatterns") {
		results, err := jq.Eval(*event.CloudTrailEvent, tagPattern)
		if err != nil {
			fmt.Println(fmt.Sprintf(`{"error": "%s"}`, err))
			return nil
		}
		for _, r := range results {
			rBytes, err := r.MarshalJSON()
			if err != nil {
				break
			}
			tagMap, ok := gjson.Parse(string(rBytes)).Value().(map[string]interface{})

			if !ok {
				break
			}

			for k, v := range tagMap {
				allTags[k] = v.(string)
			}
		}
	}
	return allTags
}

func getOutput(includeEvent bool, event *cloudtrail.Event, tags map[string]string, taggingMetadata *TaggingMetadata) interface{} {
	if includeEvent {
		return OutputWithEvent{
			Event:           event,
			TaggingMetadata: taggingMetadata,
			Tags:            tags,
		}
	}
	return Output{
		TaggingMetadata: taggingMetadata,
		Tags:            tags,
	}
}

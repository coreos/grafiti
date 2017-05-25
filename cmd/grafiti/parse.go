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
	"github.com/coreos/grafiti/arn"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	jq "github.com/threatgrid/jqpipe-go"
	"github.com/tidwall/gjson"
)

var (
	inputFile string
)

func init() {
	RootCmd.AddCommand(parseCmd)
	parseCmd.PersistentFlags().StringVarP(&inputFile, "input-file", "f", "", "CloudTrail Log input file.")
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

// CloudTrailLogFile holds the array of Records returned in a CloudTrail API
// response
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

// NotTaggedFilter holds the resource types of all resources not tagged
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
		EndTime:    aws.Time(time.Now()),
		MaxResults: aws.Int64(50),
		StartTime:  aws.Time(time.Now().Add(time.Duration(viper.GetInt("grafiti.hours")) * time.Hour)),
	}

	// Get all resourceTypes from config file. If no resourceTypes are listed, all
	// types are requested
	rts := viper.GetStringSlice("grafiti.resourceTypes")
	las := make([]*cloudtrail.LookupAttribute, 0, len(rts))
	for _, rt := range rts {
		la := &cloudtrail.LookupAttribute{
			AttributeKey:   aws.String("ResourceType"),
			AttributeValue: aws.String(rt),
		}
		las = append(las, la)
	}
	params.SetLookupAttributes(las)

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

// OutputWithEvent holds all data associated with a resource when the
// 'includeEvent' TOML field is set to 'true'
type OutputWithEvent struct {
	Event           *cloudtrail.Event
	TaggingMetadata *TaggingMetadata
	Tags            map[string]string
}

// Output holds all data associated with a resource when the 'includeEvent' TOML
// field is set to 'false'
type Output struct {
	TaggingMetadata *TaggingMetadata
	Tags            map[string]string
}

func matchFilter(output *[]byte) bool {
	filters := viper.GetStringSlice("grafiti.filterPatterns")
	if filters == nil {
		return true
	}
	matched := false
	for _, filter := range filters {
		results, err := jq.Eval(string(*output), filter)

		if err != nil || len(results) == 0 {
			continue
		}

		json, _ := results[0].MarshalJSON()
		if string(json) != "true" {
			continue
		}

		matched = true
	}

	return matched
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
		if r.ResourceName == nil || r.ResourceType == nil {
			continue
		}
		tags := getTags(event)
		output := getOutput(includeEvent, event, tags, &TaggingMetadata{
			ResourceName: *r.ResourceName,
			ResourceType: *r.ResourceType,
			ResourceARN:  arn.MapResourceTypeToARN(r, parsedEvent),
			CreatorARN:   parsedEvent.Get("userIdentity.arn").Str,
			CreatorName:  parsedEvent.Get("userIdentity.userName").Str,
			CreatedAt:    parsedEvent.Get("eventTime").Str,
		})
		resourceJSON, err := json.Marshal(output)
		if err != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
		}
		if jsonMatch := matchFilter(&resourceJSON); !jsonMatch {
			continue
		}
		fmt.Println(string(resourceJSON))
	}
}

func getTags(event *cloudtrail.Event) map[string]string {
	allTags := make(map[string]string)
	for _, tagPattern := range viper.GetStringSlice("grafiti.tagPatterns") {
		results, err := jq.Eval(*event.CloudTrailEvent, tagPattern)
		if err != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
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

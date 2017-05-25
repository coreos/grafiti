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
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/aws/aws-sdk-go/service/route53"
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
	if err := parse(); err != nil {
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

func parse() error {
	// Get all resourceTypes from config file. If no resourceTypes are listed, all
	// types are requested
	rts := viper.GetStringSlice("grafiti.resourceTypes")
	otherRts, ctRts := make([]string, 0), make([]string, 0)
	for _, rt := range rts {
		if _, ok := arn.CTUnsupportedResourceTypes[rt]; ok {
			otherRts = append(otherRts, rt)
		} else {
			ctRts = append(ctRts, rt)
		}
	}

	// If no CloudTrail-supported resource types were specified in resourceTypes,
	// don't bother calling parseCloudTrailEvents (unless no resources are specified)
	switch {
	case len(ctRts) > 0:
		if len(otherRts) > 0 {
			for _, rt := range otherRts {
				switch rt {
				case arn.Route53HostedZoneRType:
					_ = parseRoute53Events()
					break
				}
			}
		}
		if len(otherRts) == len(rts) {
			break
		}
		_ = parseCloudTrailEvents(ctRts...)
		break
	case len(ctRts) == 0:
		_ = parseCloudTrailEvents()
		_ = parseRoute53Events()
		break
	}

	return nil
}

func parseCloudTrailEvents(rts ...string) error {
	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(viper.GetString("grafiti.az")),
		},
	))
	// Get all resourceTypes from config file. If no resourceTypes are listed, all
	// types are requested
	var paramsSlice []*cloudtrail.LookupEventsInput
	if rts != nil {
		paramsSlice = make([]*cloudtrail.LookupEventsInput, 0, len(rts))
		for _, rt := range rts {
			paramsSlice = append(paramsSlice, &cloudtrail.LookupEventsInput{
				EndTime:    aws.Time(time.Now()),
				MaxResults: aws.Int64(50),
				StartTime:  aws.Time(time.Now().Add(time.Duration(viper.GetInt("grafiti.hours")) * time.Hour)),
				LookupAttributes: []*cloudtrail.LookupAttribute{
					{AttributeKey: aws.String("ResourceType"), AttributeValue: aws.String(rt)},
				},
			})
		}
	} else {
		paramsSlice = []*cloudtrail.LookupEventsInput{
			&cloudtrail.LookupEventsInput{
				EndTime:    aws.Time(time.Now()),
				MaxResults: aws.Int64(50),
				StartTime:  aws.Time(time.Now().Add(time.Duration(viper.GetInt("grafiti.hours")) * time.Hour)),
			},
		}
	}

	svc := cloudtrail.New(sess)

	for _, params := range paramsSlice {
		for {
			req, resp := svc.LookupEventsRequest(params)
			if err := req.Send(); err != nil {
				return err
			}

			printEvents(resp.Events)

			if resp.NextToken == nil || *resp.NextToken == "" {
				break
			}
			params.SetNextToken(*resp.NextToken)
		}
	}

	return nil
}

func parseRoute53Events() error {
	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(viper.GetString("grafiti.az")),
		},
	))
	svc := route53.New(sess)
	params := &route53.ListHostedZonesInput{
		MaxItems: aws.String("100"),
	}

	for {
		req, resp := svc.ListHostedZonesRequest(params)
		if err := req.Send(); err != nil {
			return err
		}

		printEvents(resp.HostedZones)

		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}
		params.SetMarker(*resp.NextMarker)
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

func printEvents(events interface{}) {
	var parsedEvent gjson.Result
	switch events.(type) {
	case []*cloudtrail.Event:
		for _, e := range events.([]*cloudtrail.Event) {
			parsedEvent = gjson.Parse(*e.CloudTrailEvent)
			printCloudTrailEvent(e, parsedEvent)
		}
	case []*route53.HostedZone:
		for _, e := range events.([]*route53.HostedZone) {
			je, _ := json.Marshal(*e)
			parsedEvent = gjson.Parse(string(je))
			printRoute53Event(e, parsedEvent)
		}
	}
}

func printCloudTrailEvent(event *cloudtrail.Event, parsedEvent gjson.Result) {
	includeEvent := viper.GetBool("grafiti.includeEvent")
	for _, r := range event.Resources {
		ARN := arn.MapResourceTypeToARN(r, parsedEvent)
		if r.ResourceName == nil || r.ResourceType == nil || ARN == "" {
			continue
		}
		tags := getTags(event)
		output := getOutput(includeEvent, event, tags, &TaggingMetadata{
			ResourceName: *r.ResourceName,
			ResourceType: *r.ResourceType,
			ResourceARN:  ARN,
			CreatorARN:   parsedEvent.Get("userIdentity.arn").Str,
			CreatorName:  parsedEvent.Get("userIdentity.userName").Str,
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

func printRoute53Event(event *route53.HostedZone, parsedEvent gjson.Result) {
	hzSplit := strings.Split(*event.Id, "/hostedzone/")
	if len(hzSplit) != 2 {
		return
	}
	tm := &TaggingMetadata{
		ResourceName: hzSplit[1],
		ResourceType: arn.Route53HostedZoneRType,
		ResourceARN:  fmt.Sprintf("arn:aws:route53:::hostedzone/%s", hzSplit[1]),
	}
	resourceJSON, err := json.Marshal(Output{
		TaggingMetadata: tm,
		Tags:            getTags(event),
	})
	if err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
	}
	if jsonMatch := matchFilter(&resourceJSON); !jsonMatch {
		return
	}
	fmt.Println(string(resourceJSON))
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

func getTags(event interface{}) map[string]string {
	eventString := ""
	switch event.(type) {
	case *route53.HostedZone:
		eventBytes, _ := json.Marshal(event)
		eventString = string(eventBytes)
		break
	case *cloudtrail.Event:
		eventString = *event.(*cloudtrail.Event).CloudTrailEvent
		break
	default:
		return nil
	}
	allTags := make(map[string]string)
	for _, tagPattern := range viper.GetStringSlice("grafiti.tagPatterns") {
		results, err := jq.Eval(eventString, tagPattern)
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

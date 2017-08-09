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
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	rgta "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	rgtaiface "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/coreos/grafiti/arn"
	"github.com/coreos/grafiti/deleter"
	"github.com/coreos/grafiti/graph"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	deleteFile string
	delAllDeps bool
	wantReport bool
)

// DeleteOrder contains the REVERSE order of deletion for all resource types
var DeleteOrder = arn.ResourceTypes{
	arn.EC2VPCRType,
	arn.EC2VPNGatewayRType, // Deletes EC2 VPN Gateway Attachments
	arn.EC2SecurityGroupRType,
	arn.EC2RouteTableRType, // Deletes EC2 Route Table Routes
	arn.EC2SubnetRType,
	arn.EC2VolumeRType,
	arn.EC2CustomerGatewayRType,
	arn.EC2VPNConnectionRType, // Deletes EC2 VPN Connection Routes
	arn.EC2NetworkACLRType,
	arn.EC2NetworkInterfaceRType,
	arn.EC2InternetGatewayRType,
	arn.IAMUserRType,
	arn.IAMRoleRType, // Deletes IAM Role Policies
	arn.IAMInstanceProfileRType,
	arn.AutoScalingLaunchConfigurationRType,
	arn.EC2EIPRType,
	arn.EC2EIPAssociationRType,
	arn.EC2NatGatewayRType,
	arn.ElasticLoadBalancingLoadBalancerRType,
	arn.AutoScalingGroupRType,
	arn.EC2InstanceRType,
	arn.EC2RouteTableAssociationRType,
	arn.Route53HostedZoneRType, // Deletes Route53 RecordSets
	arn.S3BucketRType,          // Delete S3 Objects
}

// TagFileInput holds a list of all tags to be deleted
type TagFileInput struct {
	TagFilters []*rgta.TagFilter
}

func init() {
	RootCmd.AddCommand(deleteCmd)
	deleteCmd.PersistentFlags().StringVarP(&deleteFile, "delete-file", "f", "", "File of tags of resources to delete.")
	deleteCmd.PersistentFlags().BoolVar(&delAllDeps, "all-deps", false, "Delete all dependencies of all tagged resourcs.")
	deleteCmd.PersistentFlags().BoolVar(&wantReport, "report", false, "Pretty-print a report of resource deletion errors, if any.")
}

var deleteCmd = &cobra.Command{
	Use:           "delete",
	Short:         "Delete resources in AWS by tag.",
	Long:          "Delete resources in AWS by tags specified in 'delete-file'.",
	RunE:          runDeleteCommand,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func runDeleteCommand(cmd *cobra.Command, args []string) error {
	// We decode tags from deleteFile that resources `grafiti delete` should
	// delete are tagged with.
	if deleteFile != "" {
		if err := deleteFromFile(deleteFile); err != nil {
			return fmt.Errorf("delete: %s", err)
		}
		return nil
	}

	// Same data as that in deleteFile but passed by stdin.
	if err := deleteFromStdIn(); err != nil {
		return fmt.Errorf("delete: %s", err)
	}

	return nil
}

func deleteFromFile(fname string) error {
	file, err := os.Open(fname)
	if err != nil {
		return fmt.Errorf("open delete file: %s", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	return deleteFromTags(reader)
}

func deleteFromStdIn() error {
	return deleteFromTags(os.Stdin)
}

func deleteFromTags(reader io.Reader) error {
	dec := json.NewDecoder(reader)
	// ARNs for all resources tagged with key:values encoded in tagFile.
	var arns arn.ResourceARNs

	svc := rgta.New(session.Must(session.NewSession(
		&aws.Config{},
	)))

	for {
		t, isEOF, err := decodeTagFileInput(dec)
		if err != nil {
			return err
		}
		if isEOF {
			break
		}
		if t == nil {
			continue
		}

		// Request all RGTA-taggable resources tagged with key:values encoded in
		// tagFile.
		if arns, err = getARNsForResource(svc, t.TagFilters, arns); err != nil {
			return err
		}
		for rtk := range arn.RGTAUnsupportedResourceTypes {
			// Request all RGTA-unsupported resources tagged with key:values encoded
			// in tagFile.
			if arns, err = getARNsForUnsupportedResource(rtk, t.TagFilters, arns); err != nil {
				return err
			}
		}
	}

	// Delete batch of matching resources
	return deleteARNs(arns)
}

func getARNsForResource(svc rgtaiface.ResourceGroupsTaggingAPIAPI, tags []*rgta.TagFilter, arnList arn.ResourceARNs) (arn.ResourceARNs, error) {
	// Get ARNs of matching tags
	params := &rgta.GetResourcesInput{
		TagFilters:  tags,
		TagsPerPage: aws.Int64(100),
	}

	// If available, get all resourceTypes from config file
	rts := viper.GetStringSlice("resourceTypes")
	if len(rts) != 0 {
		frts := make([]*string, 0, len(rts))
		for _, t := range rts {
			rt := arn.ResourceType(t)
			if _, ok := arn.RGTAUnsupportedResourceTypes[rt]; ok {
				continue
			}
			frts = append(frts, aws.String(arn.NamespaceForResource(rt)))
		}
		params.ResourceTypeFilters = frts
	}

	for {
		// Request a batch of matching resources
		ctx := aws.BackgroundContext()
		resp, err := svc.GetResourcesWithContext(ctx, params)
		if err != nil {
			if ignoreErrors {
				logger.Debugln("rgta: get resources:", err)
				return arnList, nil
			}
			return arnList, fmt.Errorf("rgta: get resources: %s", err)
		}

		if len(resp.ResourceTagMappingList) == 0 {
			return arnList, nil
		}

		for _, r := range resp.ResourceTagMappingList {
			if arnStr := aws.StringValue(r.ResourceARN); arnStr != "" {
				arnList = append(arnList, arn.ResourceARN(arnStr))
			}
		}

		if aws.StringValue(resp.PaginationToken) == "" {
			break
		}

		params.PaginationToken = resp.PaginationToken
	}

	return arnList, nil
}

func getARNsForUnsupportedResource(rt arn.ResourceType, tags []*rgta.TagFilter, arnList arn.ResourceARNs) (arn.ResourceARNs, error) {
	sess := session.Must(session.NewSession(
		&aws.Config{},
	))

	switch arn.NamespaceForResource(rt) {
	case arn.AutoScalingNamespace:
		return getAutoScalingResourcesByTags(autoscaling.New(sess), rt, tags, arnList)
	case arn.Route53Namespace:
		return getRoute53ResourcesByTags(route53.New(sess), rt, tags, arnList)
	}

	return arnList, nil
}

func getAutoScalingResourcesByTags(svc autoscalingiface.AutoScalingAPI, rt arn.ResourceType, rgtaTags []*rgta.TagFilter, arnList arn.ResourceARNs) (arn.ResourceARNs, error) {
	if len(rgtaTags) == 0 || len(arnList) == 0 {
		return arnList, nil
	}

	// Currently only AutoScaling Groups support tagging
	if rt != arn.AutoScalingGroupRType {
		if ignoreErrors {
			logger.Debugf("autoscaling: ResourceType %q not supported", rt)
			return arnList, nil
		}
		return arnList, fmt.Errorf("autoscaling: ResourceType %q not supported", rt)
	}

	asgTags := make([]*autoscaling.Filter, 0)
	for _, tag := range rgtaTags {
		asgTags = append(asgTags, &autoscaling.Filter{
			Name:   aws.String("key"),
			Values: aws.StringSlice([]string{*tag.Key}),
		})
		if len(tag.Values) > 0 {
			asgTags = append(asgTags, &autoscaling.Filter{
				Name:   aws.String("value"),
				Values: tag.Values,
			})
		}
	}

	params := &autoscaling.DescribeTagsInput{
		Filters:    asgTags,
		MaxRecords: aws.Int64(100),
	}

	asgNames := make(arn.ResourceNames, 0)
	for {
		ctx := aws.BackgroundContext()
		resp, err := svc.DescribeTagsWithContext(ctx, params)
		if err != nil {
			if ignoreErrors {
				logger.Debugln("autoscaling: describe tags: %s", err)
				return arnList, nil
			}
			return arnList, fmt.Errorf("autoscaling: describe tags: %s", err)
		}

		if len(resp.Tags) == 0 {
			return arnList, nil
		}

		for _, t := range resp.Tags {
			asgNames = append(asgNames, arn.ToResourceName(t.ResourceId))
		}

		if aws.StringValue(resp.NextToken) == "" {
			break
		}

		params.NextToken = resp.NextToken
	}

	asgDel := deleter.AutoScalingGroupDeleter{
		Client:        svc,
		ResourceNames: asgNames,
	}
	asgs, err := asgDel.RequestAutoScalingGroups()
	if err != nil {
		if ignoreErrors {
			logger.Debugln("autoscaling: request ASGs: %s", err)
			return arnList, nil
		}
		return arnList, fmt.Errorf("autoscaling: request ASGs: %s", err)
	}

	for _, asg := range asgs {
		arnList = append(arnList, arn.ToResourceARN(asg.AutoScalingGroupARN))
	}

	return arnList, nil
}

func getRoute53ResourcesByTags(svc route53iface.Route53API, rt arn.ResourceType, rgtaTags []*rgta.TagFilter, arnList arn.ResourceARNs) (arn.ResourceARNs, error) {
	if len(rgtaTags) == 0 || len(arnList) == 0 {
		return arnList, nil
	}

	// Currently only Route53 HostedZones support tagging
	if rt != arn.Route53HostedZoneRType {
		if ignoreErrors {
			logger.Debugf("route53: ResourceType %q not supported", rt)
			return arnList, nil
		}
		return arnList, fmt.Errorf("route53: ResourceType %q not supported", rt)
	}

	rd := deleter.Route53HostedZoneDeleter{Client: svc}
	hzs, err := rd.RequestAllRoute53HostedZones()
	if err != nil || len(hzs) == 0 {
		if ignoreErrors {
			logger.Debugln("route53: request hosted zones: %s", err)
			return arnList, nil
		}
		return arnList, fmt.Errorf("route53: request hosted zones: %s", err)
	}

	var hzIDs arn.ResourceNames
	for _, hz := range hzs {
		hzIDs = append(hzIDs, arn.SplitHostedZoneID(aws.StringValue(hz.Id)))
	}

	// Create a map[string][]string from []*rgta.TagFilter so we can filter hosted
	// zones by tag keys and values
	tagMap := createHostedZoneTagMap(rgtaTags)
	size, chunk := len(hzIDs), 10
	var filteredHZIDs arn.ResourceNames
	// Can only tag hosted zones in batches of 10
	for i := 0; i < size; i += chunk {
		stop := deleter.CalcChunk(i, size, chunk)
		params := &route53.ListTagsForResourcesInput{
			ResourceType: aws.String("hostedzone"),
			ResourceIds:  hzIDs[i:stop].AWSStringSlice(),
		}

		ctx := aws.BackgroundContext()
		resp, err := svc.ListTagsForResourcesWithContext(ctx, params)
		if err != nil {
			if ignoreErrors {
				logger.Debugln("route53: list resource tags: %s", err)
				return arnList, nil
			}
			return arnList, fmt.Errorf("route53: list resource tags: %s", err)
		}

		filteredHZIDs = filterHostedZones(filteredHZIDs, resp.ResourceTagSets, tagMap)
	}

	for _, id := range filteredHZIDs {
		hzARN := arn.MapResourceTypeToARN(arn.Route53HostedZoneRType, id)
		arnList = append(arnList, arn.ResourceARN(hzARN))
	}

	return arnList, nil
}

// createHostedZoneTagMap creates a map[string][]string corresponding to tags of
// the form 'key:values' that can be used to filter hosted zone tags by key and
// values.
func createHostedZoneTagMap(rgtaTags []*rgta.TagFilter) map[string][]string {
	tagMap := make(map[string][]string)
	for _, tag := range rgtaTags {
		key := aws.StringValue(tag.Key)
		if _, ok := tagMap[key]; !ok {
			tagMap[key] = []string{}
			for _, v := range tag.Values {
				tagMap[key] = append(tagMap[key], aws.StringValue(v))
			}
		}
	}
	return tagMap
}

// filterHostedZones adds hosted zone IDs to hzIDs if their tag key(s) (and
// values if present) match those provided by those in deleteFile.
func filterHostedZones(hzIDs arn.ResourceNames, tagSets []*route53.ResourceTagSet, tagMap map[string][]string) arn.ResourceNames {
	for _, rts := range tagSets {
		nameStr := arn.ToResourceName(rts.ResourceId)
		for _, tag := range rts.Tags {
			if vals, ok := tagMap[aws.StringValue(tag.Key)]; ok {
				// If no tag values are specified, then we want all hosted zones that
				// match a specific key but have any value. Append all that have key
				if len(vals) == 0 {
					hzIDs = append(hzIDs, nameStr)
					continue
				}
				for _, v := range vals {
					if v == aws.StringValue(tag.Value) {
						hzIDs = append(hzIDs, nameStr)
						break
					}
				}
			}
		}
	}
	return hzIDs
}

// Traverse dependency graph and request all possible ID's of resource
// dependencies, then bucket them according to ResourceType.
func bucketARNs(ARNs arn.ResourceARNs) map[arn.ResourceType]deleter.ResourceDeleter {
	// All ARN's stored here. Key is some arn.*RType, value is a slice of ARN's
	resMap := make(map[arn.ResourceType]deleter.ResourceDeleter)
	seen := map[arn.ResourceName]struct{}{}

	// Initialize with all ID's from ARN's tagged in CloudTrail logs
	for _, a := range ARNs {
		rt, rn := arn.MapARNToRTypeAndRName(a)
		// Remove duplicates and nil resources
		if _, ok := seen[rn]; ok || rt == "" || rn == "" {
			continue
		}
		seen[rn] = struct{}{}

		if _, ok := resMap[rt]; !ok {
			resMap[rt] = deleter.InitResourceDeleter(rt)
		}
		resMap[rt].AddResourceNames(rn)
	}

	// Unless the user specifies the --all-deps flag, do not find/delete
	// dependencies of resources
	if delAllDeps {
		graph.FillDependencyGraph(resMap)
	}

	return resMap
}

type delResMap struct {
	Type     string
	Deleters deleter.ResourceDeleter
}

func deleteARNs(ARNs arn.ResourceARNs) error {
	// Create a slice of ARN's for every ResourceType in ARNs
	resMap := bucketARNs(ARNs)
	if len(resMap) == 0 {
		return nil
	}

	// Ensure deletion order. Most resources have dependencies, so a dependency
	// graph must be constructed and executed. See README for deletion order.
	sorted := organizeByDelOrder(resMap)

	cfg := &deleter.DeleteConfig{
		IgnoreErrors: ignoreErrors,
		DryRun:       dryRun,
		Logger:       logger,
	}

	// Delete all ARN's in a slice mapped by ResourceType. Iterate in reverse to
	// delete all non-dependent resources first
	for i := len(sorted) - 1; i >= 0; i-- {
		if err := sorted[i].Deleters.DeleteResources(cfg); err != nil {
			// DeleteResources should only return an error when ignoreErrors == false,
			// so we want to return this err if one is encountered.
			return fmt.Errorf("delete resources: %s", err)
		}
	}

	// Print all failed deletion logs in report format at end of deletion cycle
	if wantReport && logger.LogFile != "" {
		f, err := os.Open(logger.LogFile)
		if err != nil {
			return fmt.Errorf("open log file: %s", err)
		}
		defer f.Close()
		fmt.Println(logHead)
		deleter.PrintLogFileReport(bufio.NewReader(f), formatReportLogEntry)
		fmt.Println(logTail)
	}

	return nil
}

func organizeByDelOrder(resMap map[arn.ResourceType]deleter.ResourceDeleter) []delResMap {
	sorted := make([]delResMap, 0, len(resMap))

	// Append ARN's to sorted in deletion order
	for _, rt := range DeleteOrder {
		if dels, ok := resMap[rt]; ok {
			sorted = append(sorted, delResMap{
				Type:     rt.String(),
				Deleters: dels,
			})
			delete(resMap, rt)
		}
	}

	// Add the remaining ARN's
	for rt, dels := range resMap {
		sorted = append(sorted, delResMap{
			Type:     rt.String(),
			Deleters: dels,
		})
	}

	return sorted
}

func decodeTagFileInput(decoder *json.Decoder) (*TagFileInput, bool, error) {
	var decoded TagFileInput
	if err := decoder.Decode(&decoded); err != nil {
		if err == io.EOF {
			return &decoded, true, nil
		}
		if ignoreErrors {
			logger.Debugln("decode delete input:", err)
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("decode delete input: %s", err)
	}
	return &decoded, false, nil
}

// Beginning and end of log reports
const logTail = `=================================================`

const logHead = logTail + "\n== Log Report: Failed Resource Deletion Events ==\n" + logTail

func formatReportLogEntry(e *deleter.LogEntry) (m string) {
	if e.Error == nil {
		return ""
	}

	m = fmt.Sprintf("Failed to delete %s %s", e.ResourceType, e.ResourceName)

	if e.ParentResourceName != "" {
		m = fmt.Sprintf("%s from %s %s", m, e.ParentResourceType, e.ParentResourceName)
	}

	switch {
	case e.AWSErrorCode != "":
		// AWS error messages are verbose and should be logged to a log file instead
		// of printed
		m = fmt.Sprintf("%s (%s)", m, e.AWSErrorCode)
	case e.ErrMsg != "":
		m = fmt.Sprintf("%s (%s)", m, e.ErrMsg)
	}

	return
}

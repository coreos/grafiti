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
	"strings"
	"time"

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
	silent     bool
	delAllDeps bool
)

// DeleteOrder contains the REVERSE order of deletion for all resource types
var DeleteOrder = arn.ResourceTypes{
	arn.EC2VPCRType,
	arn.EC2SecurityGroupRType,
	arn.EC2RouteTableRType, // Deletes RouteTable Routes
	arn.EC2SubnetRType,
	arn.EC2VolumeRType,
	arn.EC2CustomerGatewayRType,
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

// DeleteInput holds a list of all tags to be deleted
type DeleteInput struct {
	TagFilters []*rgta.TagFilter
}

func init() {
	RootCmd.AddCommand(deleteCmd)
	deleteCmd.PersistentFlags().StringVarP(&deleteFile, "delete-file", "f", "", "File of tags of resources to delete.")
	deleteCmd.PersistentFlags().BoolVarP(&silent, "silent", "s", false, "Suppress JSON output.")
	deleteCmd.PersistentFlags().BoolVar(&delAllDeps, "all-deps", false, "Delete all dependencies of all tagged resourcs.")
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete resources in AWS",
	Long:  "Delete resources in AWS. Uses the configured delete filters to decide which resources to delete.",
	RunE:  runDeleteCommand,
}

func runDeleteCommand(cmd *cobra.Command, args []string) error {
	if deleteFile != "" {
		return deleteFromFile(deleteFile)
	}
	if err := deleteFromStdIn(); err != nil {
		return err
	}
	return nil
}

func deleteFromFile(fname string) error {
	file, err := os.Open(fname)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(file)
	return deleteFromTags(reader)
}

func deleteFromStdIn() error {
	return deleteFromTags(os.Stdin)
}

func deleteFromTags(reader io.Reader) error {
	dec := json.NewDecoder(reader)
	// Collect all ARN's
	allARNs := make(arn.ResourceARNs, 0)

	svc := rgta.New(session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(viper.GetString("grafiti.az")),
		},
	)))

	for {
		t, isEOF, err := decodeDeleteInput(dec)
		if err != nil {
			return err
		}
		if isEOF {
			break
		}
		if t == nil {
			continue
		}

		getARNsForResource(svc, t.TagFilters, allARNs)

		for rtk := range arn.RGTAUnsupportedResourceTypes {
			// Request all RGTA-unsupported resources with the same tags
			getARNsForUnsupportedResource(rtk, t.TagFilters, allARNs)
		}
	}

	// Delete batch of matching resources
	if err := deleteARNs(allARNs); err != nil {
		return err
	}

	if !silent {
		arnsJSON, _ := json.MarshalIndent(allARNs, "", " ")
		fmt.Printf("{\"DeletedARNs\": %s}\n", arnsJSON)
	}

	return nil
}

func getARNsForResource(svc rgtaiface.ResourceGroupsTaggingAPIAPI, tags []*rgta.TagFilter, arnList arn.ResourceARNs) {
	// Get ARNs of matching tags
	params := &rgta.GetResourcesInput{
		TagFilters:  tags,
		TagsPerPage: aws.Int64(100),
	}

	// If available, get all resourceTypes from config file
	rts := viper.GetStringSlice("grafiti.resourceTypes")
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
		req, resp := svc.GetResourcesRequest(params)
		if err := req.Send(); err != nil {
			return
		}

		if len(resp.ResourceTagMappingList) == 0 {
			fmt.Println("No resources match the specified tag filters")
			return
		}

		for _, r := range resp.ResourceTagMappingList {
			if r.ResourceARN != nil && *r.ResourceARN != "" {
				arnList = append(arnList, arn.ResourceARN(*r.ResourceARN))
			}
		}

		if resp.PaginationToken == nil || *resp.PaginationToken == "" {
			break
		}

		params.PaginationToken = resp.PaginationToken
	}
}

func getARNsForUnsupportedResource(rt arn.ResourceType, tags []*rgta.TagFilter, arnList arn.ResourceARNs) {
	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(viper.GetString("grafiti.az")),
		},
	))

	switch arn.NamespaceForResource(rt) {
	case arn.AutoScalingNamespace:
		getAutoScalingResourcesByTags(autoscaling.New(sess), rt, tags, arnList)
	case arn.Route53Namespace:
		getRoute53ResourcesByTags(route53.New(sess), rt, tags, arnList)
	}
}

func getAutoScalingResourcesByTags(svc autoscalingiface.AutoScalingAPI, rt arn.ResourceType, rgtaTags []*rgta.TagFilter, arnList arn.ResourceARNs) {
	if len(rgtaTags) == 0 {
		return
	}

	// Currently only AutoScaling Groups support tagging
	if rt != arn.AutoScalingGroupRType {
		return
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
		resp, rerr := svc.DescribeTagsWithContext(ctx, params)
		if rerr != nil {
			return
		}

		if resp.Tags == nil {
			break
		}
		for _, t := range resp.Tags {
			asgNames = append(asgNames, arn.ResourceName(*t.ResourceId))
		}

		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}

		params.NextToken = resp.NextToken
	}

	asgDel := deleter.AutoScalingGroupDeleter{ResourceNames: asgNames}
	asgs, aerr := asgDel.RequestAutoScalingGroups()
	if aerr != nil {
		return
	}

	for _, asg := range asgs {
		arnList = append(arnList, arn.ResourceARN(*asg.AutoScalingGroupARN))
	}

	return
}

func getRoute53ResourcesByTags(svc route53iface.Route53API, rt arn.ResourceType, rgtaTags []*rgta.TagFilter, arnList arn.ResourceARNs) {
	if len(rgtaTags) == 0 {
		return
	}

	// Currently only Route53 HostedZones support tagging
	if rt != arn.Route53HostedZoneRType {
		return
	}

	tagKeyMap := make(map[string][]string)
	for _, tag := range rgtaTags {
		if _, ok := tagKeyMap[*tag.Key]; !ok {
			tagKeyMap[*tag.Key] = make([]string, 0, len(tag.Values))
			for _, v := range tag.Values {
				tagKeyMap[*tag.Key] = append(tagKeyMap[*tag.Key], *v)
			}
		}
	}

	rd := deleter.Route53HostedZoneDeleter{Client: svc}
	hzs, rerr := rd.RequestAllRoute53HostedZones()
	if rerr != nil || len(hzs) == 0 {
		return
	}

	hzIDs := make(arn.ResourceNames, 0, len(hzs))
	for _, hz := range hzs {
		hzSplit := strings.Split(*hz.Id, "/hostedzone/")
		if len(hzSplit) != 2 {
			continue
		}
		hzIDs = append(hzIDs, arn.ResourceName(hzSplit[1]))
	}

	params := &route53.ListTagsForResourcesInput{
		ResourceType: aws.String("hostedzone"),
	}

	size := len(hzIDs)
	filteredIDs := make(arn.ResourceNames, 0, len(hzIDs))
	// Can only tag hosted zones in batches of 10
	for i := 0; i < size; i += 10 {
		stop := i + 10
		if size-stop < 0 {
			stop = i + size%10
		}
		params.ResourceIds = hzIDs[i:stop].AWSStringSlice()

		ctx := aws.BackgroundContext()
		resp, rerr := svc.ListTagsForResourcesWithContext(ctx, params)
		if rerr != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", rerr.Error())
			return
		}

		for _, rts := range resp.ResourceTagSets {
			for _, tag := range rts.Tags {
				if vals, ok := tagKeyMap[*tag.Key]; ok {
					// If no tag values are specified, then we want all hosted zones that
					// match a specific key but have any value. Append all that have key
					if vals == nil || len(vals) == 0 {
						filteredIDs = append(filteredIDs, arn.ResourceName(*rts.ResourceId))
						continue
					}
					for _, v := range vals {
						if v == *tag.Value {
							filteredIDs = append(filteredIDs, arn.ResourceName(*rts.ResourceId))
							break
						}
					}
				}
			}
		}
	}

	for _, id := range filteredIDs {
		hzARN := fmt.Sprintf("arn:aws:route53:::hostedzone/%s", id)
		arnList = append(arnList, arn.ResourceARN(hzARN))
	}

	return
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

	// Delete all ARN's in a slice mapped by ResourceType. Iterate in reverse to
	// delete all non-dependent resources first
	cfg := &deleter.DeleteConfig{
		IgnoreErrors: ignoreErrors,
		DryRun:       dryRun,
		BackoffTime:  time.Duration(viper.GetInt("grafiti.backoffTime")) * time.Millisecond,
	}
	for i := len(sorted) - 1; i >= 0; i-- {
		if err := sorted[i].Deleters.DeleteResources(cfg); err != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
		}
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

func decodeDeleteInput(decoder *json.Decoder) (*DeleteInput, bool, error) {
	var decoded DeleteInput
	if err := decoder.Decode(&decoded); err != nil {
		if err == io.EOF {
			return &decoded, true, nil
		}
		if ignoreErrors {
			fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
			return nil, false, nil
		}
		return nil, false, err
	}
	return &decoded, false, nil
}

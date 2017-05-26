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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	rgta "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/coreos/grafiti/arn"
	"github.com/coreos/grafiti/deleter"
	"github.com/coreos/grafiti/describe"
	"github.com/coreos/grafiti/graph"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	deleteFile string
	silent     bool
)

// DeleteOrder contains the REVERSE order of deletion for all resource types
var DeleteOrder = []string{
	arn.EC2VPCRType,
	arn.EC2SecurityGroupRType,
	arn.EC2RouteTableRType,
	arn.EC2SubnetRType,
	arn.EC2VolumeRType,
	arn.EC2CustomerGatewayRType,
	arn.EC2NetworkACLRType,
	arn.EC2NetworkInterfaceRType,
	arn.EC2InternetGatewayRType,
	arn.IAMUserRType,
	arn.IAMRoleRType,
	arn.IAMInstanceProfileRType,
	arn.AutoScalingLaunchConfigurationRType,
	arn.EC2EIPRType,
	arn.EC2EIPAssociationRType,
	arn.EC2NatGatewayRType,
	arn.ElasticLoadBalancingLoadBalancerRType,
	arn.AutoScalingGroupRType,
	arn.EC2InstanceRType,
	// Delete SecurityGroup Rule
	// Delete RouteTable Routes
	arn.EC2RouteTableAssociationRType,
	arn.Route53HostedZoneRType,
	// Delete Route53 RecordSets
	// Delete IAM Role Policies
	arn.S3BucketRType,
	// Delete S3 Objects
}

// DeleteInput holds a list of all tags to be deleted
type DeleteInput struct {
	TagFilters []*rgta.TagFilter
}

func init() {
	RootCmd.AddCommand(deleteCmd)
	deleteCmd.PersistentFlags().StringVarP(&deleteFile, "delete-file", "f", "", "File of tags of resources to delete.")
	deleteCmd.PersistentFlags().BoolVarP(&silent, "silent", "s", false, "Suppress JSON output.")
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
	region := viper.GetString("grafiti.az")
	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(region),
		},
	))
	svc := rgta.New(sess)
	dec := json.NewDecoder(reader)
	// Collect all ARN's
	allARNs := make([]string, 0)

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

		// Get ARNs of matching tags
		params := &rgta.GetResourcesInput{
			TagFilters:  t.TagFilters,
			TagsPerPage: aws.Int64(100),
		}

		// If available, get all resourceTypes from config file
		rts := viper.GetStringSlice("grafiti.resourceTypes")
		if rts != nil {
			rtfs := make([]*string, 0, len(rts))
			for _, rt := range rts {
				if _, ok := arn.RGTAUnsupportedResourceTypes[rt]; ok {
					continue
				}
				rtfs = append(rtfs, aws.String(arn.NamespaceForResource(rt)))
			}
			params.SetResourceTypeFilters(rtfs)
		}

		for {
			// Request a batch of matching resources
			req, resp := svc.GetResourcesRequest(params)
			if err := req.Send(); err != nil {
				return err
			}

			if len(resp.ResourceTagMappingList) == 0 {
				fmt.Println("No resources match the specified tag filters")
				return nil
			}

			for _, r := range resp.ResourceTagMappingList {
				if r.ResourceARN != nil && *r.ResourceARN != "" {
					allARNs = append(allARNs, *r.ResourceARN)
				}
			}

			if resp.PaginationToken == nil || *resp.PaginationToken == "" {
				break
			}
			params.SetPaginationToken(*resp.PaginationToken)
		}

		// Request all AutoScalingGroups with the same tags, as these cannot be
		// retrieved using the resource group tagging API
		asgs := getAutoScalingGroupsByTags(&t.TagFilters)
		if asgs != nil {
			for _, asg := range *asgs {
				allARNs = append(allARNs, *asg.AutoScalingGroupARN)
			}
		}
		// Request all HostedZones with the same tags, as these cannot be
		// retrieved using the resource group tagging API
		hzIDs := getRoute53HostedZoneIDsByTags(&t.TagFilters)
		if hzIDs != nil {
			for _, id := range *hzIDs {
				hzARN := fmt.Sprintf("arn:aws:route53:::hostedzone/%s", id)
				allARNs = append(allARNs, hzARN)
			}
		}
	}

	// Delete batch of matching resources
	if err := deleteARNs(&allARNs); err != nil {
		return err
	}

	if !silent {
		arnsJSON, _ := json.MarshalIndent(allARNs, "", " ")
		fmt.Printf("{\"DeletedARNs\": %s}\n", arnsJSON)
	}

	return nil
}

func getAutoScalingGroupsByTags(rgtaTags *[]*rgta.TagFilter) *[]*autoscaling.Group {
	if rgtaTags == nil {
		return nil
	}
	asgTags := make([]*autoscaling.Filter, 0, 2*len(*rgtaTags))
	for _, tag := range *rgtaTags {
		asgTags = append(asgTags, &autoscaling.Filter{
			Name:   aws.String("key"),
			Values: aws.StringSlice([]string{*tag.Key}),
		})
		asgTags = append(asgTags, &autoscaling.Filter{
			Name:   aws.String("value"),
			Values: tag.Values,
		})
	}

	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(viper.GetString("grafiti.az")),
		},
	))
	svc := autoscaling.New(sess)

	params := &autoscaling.DescribeTagsInput{
		Filters:    asgTags,
		MaxRecords: aws.Int64(100),
	}

	asgNames := make([]string, 0)
	for {
		ctx := aws.BackgroundContext()
		resp, rerr := svc.DescribeTagsWithContext(ctx, params)
		if rerr != nil {
			return nil
		}

		if resp.Tags == nil {
			break
		}
		for _, t := range resp.Tags {
			asgNames = append(asgNames, *t.ResourceId)
		}

		if resp.NextToken != nil && *resp.NextToken != "" {
			params.SetNextToken(*resp.NextToken)
			continue
		}
		break
	}

	asgs, aerr := describe.GetAutoScalingGroupsByNames(&asgNames)
	if aerr != nil {
		return nil
	}

	return asgs
}

func getRoute53HostedZoneIDsByTags(rgtaTags *[]*rgta.TagFilter) *[]string {
	if rgtaTags == nil {
		return nil
	}
	tagKeyMap := make(map[string][]string)
	for _, tag := range *rgtaTags {
		for _, v := range tag.Values {
			if _, ok := tagKeyMap[*tag.Key]; !ok {
				tagKeyMap[*tag.Key] = append(tagKeyMap[*tag.Key], *v)
			}
		}
	}

	hzs, herr := describe.GetRoute53HostedZones()
	if herr != nil || hzs == nil {
		return nil
	}

	hzIDs := make([]string, 0, len(*hzs))
	for _, hz := range *hzs {
		hzSplit := strings.Split(*hz.Id, "/hostedzone/")
		if len(hzSplit) != 2 {
			continue
		}
		hzIDs = append(hzIDs, hzSplit[1])
	}

	params := &route53.ListTagsForResourcesInput{
		ResourceType: aws.String("hostedzone"),
	}

	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(viper.GetString("grafiti.az")),
		},
	))
	svc := route53.New(sess)

	size := len(hzIDs)
	filteredIDs := make([]string, 0, len(hzIDs))
	// Can only request hosted zones in batches of 10
	for i := 0; i < size; i += 10 {
		stop := i + 10
		if size-stop < 0 {
			stop = i + size%10
		}
		params.SetResourceIds(aws.StringSlice(hzIDs[i:stop]))

		ctx := aws.BackgroundContext()
		resp, rerr := svc.ListTagsForResourcesWithContext(ctx, params)
		if rerr != nil {
			fmt.Println(rerr.Error())
			return nil
		}

		for _, rts := range resp.ResourceTagSets {
			for _, tag := range rts.Tags {
				if vals, ok := tagKeyMap[*tag.Key]; ok {
					for _, v := range vals {
						if v == *tag.Value {
							filteredIDs = append(filteredIDs, *rts.ResourceId)
							break
						}
					}
				}
			}
		}
	}

	return &filteredIDs
}

// Traverse dependency graph and request all possible ID's of resource
// dependencies, then bucket them according to ResourceType.
func bucketARNs(ARNs *[]string) *map[string][]string {
	if ARNs == nil {
		return nil
	}
	// All ARN's stored here. Key is some arn.*RType, value is a slice of ARN's
	rTypeToARNMap := make(map[string][]string)
	seen := map[string]struct{}{}

	// Initialize with all ID's from ARN's tagged in CloudTrail logs
	for _, a := range *ARNs {
		rType, rName := arn.MapARNToRTypeAndRName(a)
		// Remove duplicates and nil resources
		if _, ok := seen[rName]; ok || rType == "" || rName == "" {
			continue
		}
		seen[rName] = struct{}{}
		rTypeToARNMap[rType] = append(rTypeToARNMap[rType], rName)
	}

	graph.FillDependencyGraph(&rTypeToARNMap)

	return &rTypeToARNMap
}

type delResMap struct {
	ResourceType string
	IDs          []string
}

func appendToOrderedList(rt string, resMap *map[string][]string, sortedMap *[]*delResMap) {
	if ids, ok := (*resMap)[rt]; ok {
		drm := &delResMap{
			ResourceType: rt,
			IDs:          ids,
		}
		*sortedMap = append(*sortedMap, drm)
		delete(*resMap, rt)
	}
}

func deleteARNs(ARNs *[]string) error {
	awsCfg := &aws.Config{
		Region: aws.String(viper.GetString("grafiti.az")),
	}
	cfg := &deleter.DeleteConfig{
		IgnoreErrors: ignoreErrors,
		DryRun:       dryRun,
	}

	// Create a slice of ARN's for every ResourceType in ARNs
	resMap := bucketARNs(ARNs)
	if resMap == nil {
		return nil
	}
	// Ensure deletion order. Most resources have dependencies, so a dependency
	// graph must be constructed and executed. See README for deletion order.
	sortedByDelOrder := make([]*delResMap, 0, len(*resMap))
	// Append ARN's to sortedByDelOrder in deletion order
	for _, rt := range DeleteOrder {
		appendToOrderedList(rt, resMap, &sortedByDelOrder)
	}
	// Add the remaining ARN's
	var drm *delResMap
	for k, v := range *resMap {
		drm = &delResMap{
			ResourceType: k,
			IDs:          v,
		}
		sortedByDelOrder = append(sortedByDelOrder, drm)
	}

	// Delete all ARN's in a slice mapped by ResourceType. Iterate in reverse to
	// delete all non-dependent resources first
	size := len(sortedByDelOrder)
	for i := range sortedByDelOrder {
		drm = sortedByDelOrder[size-1-i]
		cfg.ResourceType = drm.ResourceType
		cfg.AWSSession = session.Must(session.NewSession(awsCfg))

		if err := deleter.DeleteAWSResourcesByIDs(cfg, &drm.IDs); err != nil {
			fmt.Printf("Error deleting resources of type %s: %s\n", drm.ResourceType, err.Error())
		}

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
			fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
			return nil, false, nil
		}
		return nil, false, err
	}
	return &decoded, false, nil
}

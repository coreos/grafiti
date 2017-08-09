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
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tagFile string

// TaggingMetadata is the data required to find and tag a resource
type TaggingMetadata struct {
	ResourceName arn.ResourceName
	ResourceType arn.ResourceType
	ResourceARN  arn.ResourceARN
	CreatorARN   arn.ResourceARN
	CreatorName  arn.ResourceName
}

// Tags are an alias for mapping Tag.Key -> Tag.Value
type Tags map[string]string

// Tag holds user-defined tag values from a TOML file
type Tag struct {
	Key   string
	Value string
}

// TagInput holds all Data describing a resource
type TagInput struct {
	TaggingMetadata TaggingMetadata
	Tags            Tags
}

func shouldEject(ejectSize, setSize int, createdAt time.Time) bool {
	limitInt := viper.GetInt("bucketEjectLimitSeconds")
	limit := time.Duration(limitInt) * time.Second

	ejectTime := time.Time(createdAt.Add(limit))

	return setSize == ejectSize || ((createdAt.After(ejectTime) || createdAt.Equal(ejectTime)) && setSize > 0)
}

// ARNSet is a set of ARNs (no duplicates)
type ARNSet map[arn.ResourceARN]struct{}

// NewARNSet creates a new ARNSet
func NewARNSet() ARNSet {
	return make(map[arn.ResourceARN]struct{})
}

// AddARN adds an ARN to an ARNSet
func (a *ARNSet) AddARN(ARN arn.ResourceARN) {
	(*a)[ARN] = struct{}{}
}

// ToARNList generates a list of ResourceARNs from a ARNSet
func (a *ARNSet) ToARNList() arn.ResourceARNs {
	var arns = make(arn.ResourceARNs, 0, len(*a))
	for k := range *a {
		arns = append(arns, arn.ResourceARN(k))
	}
	return arns
}

// TrackedARNSet tracks how long since ARNs in a bucket have been ejected. If
// running in daemon mode, buckets may be half full for long periods of time.
// To prevent this, eject ARNs from buckets after a set period of time. Default
// limit of 10 minutes
type TrackedARNSet struct {
	ARNSet
	CreatedAt time.Time
}

// ShouldEject calculates whether a bucket has at least 20 member ARNs or
// CreatedAt is after the user-specified duration
func (s *TrackedARNSet) ShouldEject() bool {
	return shouldEject(20, len(s.ARNSet), s.CreatedAt)
}

// ARNSetBucket maps a tag to a set of tracked ARNs
type ARNSetBucket map[Tag]TrackedARNSet

// NewARNSetBucket creates a new map of Tag -> TrackedARNSet
func NewARNSetBucket() ARNSetBucket {
	return make(map[Tag]TrackedARNSet)
}

// AddARNToBuckets adds tags to an ARNSet, or creates a new set if one does
// not exist
func (b *ARNSetBucket) AddARNToBuckets(ARN arn.ResourceARN, tags map[string]string) {
	if ARN == "" {
		return
	}
	for tagKey, tagValue := range tags {
		tag := Tag{tagKey, tagValue}

		resourceSet, found := (*b)[tag]
		if !found {
			resourceSet = TrackedARNSet{
				ARNSet:    NewARNSet(),
				CreatedAt: time.Now(),
			}
			(*b)[tag] = resourceSet
		}
		resourceSet.AddARN(ARN)
	}
}

// ClearBucket creates a new ARNSet for a specific Tag
func (b *ARNSetBucket) ClearBucket(bucket Tag) {
	(*b)[bucket] = TrackedARNSet{
		ARNSet:    NewARNSet(),
		CreatedAt: time.Now(),
	}
}

// ResourceNameSet is a set of ResourceNames (no duplicates) mapped to a map of
// applied Tags
type ResourceNameSet map[arn.ResourceName]Tags

// NewResourceNameSet creates a new map of ResourceNameSet -> Tags
func NewResourceNameSet() ResourceNameSet {
	return make(map[arn.ResourceName]Tags)
}

// AddResourceName adds an ResourceName to an ResourceNameSet
func (a *ResourceNameSet) AddResourceName(name arn.ResourceName) {
	(*a)[name] = make(map[string]string)
}

// AddTags adds Tags to a ResourceName in a ResourceNameSet
func (a *ResourceNameSet) AddTags(name arn.ResourceName, tags map[string]string) {
	if _, ok := (*a)[name]; !ok {
		a.AddResourceName(name)
	}
	for tagKey, tagValue := range tags {
		(*a)[name][tagKey] = tagValue
	}
}

// TrackedResourceNameSet tracks how long since ResourceNames in a bucket have been ejected. If
// running in daemon mode, buckets may be half full for long periods of time.
// To prevent this, eject ResourceNames from buckets after a set period of time. Default
// limit of 10 minutes
type TrackedResourceNameSet struct {
	ResourceNameSet
	CreatedAt time.Time
}

// ShouldEject calculates whether a bucket has at least 10 member ResourceNames
// or CreatedAt is after the user-specified duration
func (s *TrackedResourceNameSet) ShouldEject() bool {
	return shouldEject(10, len(s.ResourceNameSet), s.CreatedAt)
}

// ResourceNameSetBucket maps a ResourceType to a set of tracked ResourceNames
type ResourceNameSetBucket map[arn.ResourceType]TrackedResourceNameSet

// NewResourceNameSetBucket creates a new map of Tag -> TrackedResourceNameSet
func NewResourceNameSetBucket() ResourceNameSetBucket {
	return make(map[arn.ResourceType]TrackedResourceNameSet)
}

// AddResourceNameToBucket adds tags to an ResourceNameSet, or creates a new
// set if one does not exist
func (b *ResourceNameSetBucket) AddResourceNameToBucket(bucket arn.ResourceType, name arn.ResourceName, tags map[string]string) {
	if bucket == "" || name == "" {
		return
	}

	nameSet, ok := (*b)[bucket]
	if !ok {
		nameSet = TrackedResourceNameSet{
			ResourceNameSet: NewResourceNameSet(),
			CreatedAt:       time.Now(),
		}
		(*b)[bucket] = nameSet
	}
	nameSet.AddTags(name, tags)
}

// ClearBucket creates a new ResourceNameSet for a ResourceType and sets the
// creation time to now
func (b *ResourceNameSetBucket) ClearBucket(bucket arn.ResourceType) {
	(*b)[bucket] = TrackedResourceNameSet{
		ResourceNameSet: NewResourceNameSet(),
		CreatedAt:       time.Now(),
	}
}

func init() {
	RootCmd.AddCommand(tagCmd)
	tagCmd.PersistentFlags().StringVarP(&tagFile, "tag-file", "f", "", "File containing JSON objects of taggable resources and tag key/value pairs. Format is the output format of grafiti parse.")
}

var tagCmd = &cobra.Command{
	Use:           "tag",
	Short:         "Tag resources in AWS.",
	Long:          "Tag resources in AWS using tags created by the 'parse' subcommand.",
	RunE:          runTagCommand,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func runTagCommand(cmd *cobra.Command, args []string) error {
	// tagFile holds data structured in the output format of `grafiti parse`.
	if tagFile != "" {
		if err := tagFromFile(tagFile); err != nil {
			return fmt.Errorf("tag: %s", err)
		}
		return nil
	}

	// Same data as that in tagFile but passed by stdin.
	if err := tagFromStdIn(); err != nil {
		return fmt.Errorf("tag: %s", err)
	}

	return nil
}

func tagFromFile(fname string) error {
	file, err := os.Open(fname)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(file)
	return tag(reader)
}

func tagFromStdIn() error {
	return tag(os.Stdin)
}

func tag(reader io.Reader) error {
	svc := rgta.New(session.Must(session.NewSession(
		&aws.Config{},
	)))
	dec := json.NewDecoder(reader)

	// Holds all ARN's of resources supported by the RGTA
	arnBuckets := NewARNSetBucket()
	// Holds all resource names of resources not supported by the RGTA
	resourceNameBuckets := NewResourceNameSetBucket()

	for {
		t, isEOF, err := decodeInput(dec)
		if err != nil {
			return err
		}
		if t == nil {
			continue
		}

		// Check map that holds all RGTA-unsupported resource types and bucket
		// accordingly
		tm := t.TaggingMetadata
		// Certain resources do not support tagging at all. Skip these.
		if _, ok := arn.UntaggableResourceTypes[tm.ResourceType]; ok {
			continue
		}

		if tm.ResourceType != "" && tm.ResourceName != "" && tm.ResourceARN != "" {
			if _, ok := arn.RGTAUnsupportedResourceTypes[tm.ResourceType]; ok {
				resourceNameBuckets.AddResourceNameToBucket(tm.ResourceType, tm.ResourceName, t.Tags)
			} else {
				arnBuckets.AddARNToBuckets(tm.ResourceARN, t.Tags)
			}
		}

		for tag, bucket := range arnBuckets {
			if bucket.ShouldEject() || (isEOF && len(bucket.ARNSet) > 0) {
				if err := tagARNBucket(svc, bucket.ToARNList(), tag); err != nil {
					return err
				}
				arnBuckets.ClearBucket(tag)
			}
		}

		for rt, buckets := range resourceNameBuckets {
			if buckets.ShouldEject() || (isEOF && len(buckets.ResourceNameSet) > 0) {
				if err := tagUnsupportedResourceType(rt, buckets.ResourceNameSet); err != nil {
					return err
				}
				resourceNameBuckets.ClearBucket(rt)
			}
		}

		if isEOF {
			break
		}
	}

	return nil
}

func tagUnsupportedResourceType(rt arn.ResourceType, nameSet ResourceNameSet) error {
	sess := session.Must(session.NewSession(
		&aws.Config{},
	))

	switch arn.NamespaceForResource(rt) {
	case arn.AutoScalingNamespace:
		svc := autoscaling.New(sess)
		for n, tags := range nameSet {
			if err := tagAutoScalingResources(svc, rt, n, tags); err != nil {
				return err
			}
		}
	case arn.Route53Namespace:
		svc := route53.New(sess)
		for n, tags := range nameSet {
			if err := tagRoute53Resource(svc, rt, n, tags); err != nil {
				return err
			}
		}
	}

	return nil
}

// Tag an autoscaling resource individually. Avoids failures encountered when
// tagging a batch of resources containing one that does not exist in AWS
func tagAutoScalingResources(svc autoscalingiface.AutoScalingAPI, rt arn.ResourceType, rn arn.ResourceName, tags Tags) error {
	// Only autoscaling groups can be tagged
	if rt != arn.AutoScalingGroupRType || rn == "" {
		return nil
	}

	var asgTags []*autoscaling.Tag
	for tk, tv := range tags {
		asgTags = append(asgTags, &autoscaling.Tag{
			Key:               aws.String(tk),
			Value:             aws.String(tv),
			ResourceType:      aws.String("auto-scaling-group"),
			ResourceId:        rn.AWSString(),
			PropagateAtLaunch: aws.Bool(true),
		})
	}
	if len(asgTags) == 0 {
		return nil
	}
	params := &autoscaling.CreateOrUpdateTagsInput{
		Tags: asgTags,
	}

	pj, err := json.Marshal(params)
	if err != nil {
		if ignoreErrors {
			logger.Debugln("marshal autoscaling params:", err)
			return nil
		}
		return fmt.Errorf("marshal autoscaling params: %s", err)
	}
	fmt.Println(string(pj))

	if dryRun {
		return nil
	}

	ctx := aws.BackgroundContext()
	if _, err := svc.CreateOrUpdateTagsWithContext(ctx, params); err != nil {
		if ignoreErrors {
			logger.Debugln("autoscaling: tag resources:", err)
			return nil
		}
		return fmt.Errorf("autoscaling: tag resources %s", err)
	}

	return nil
}

func tagRoute53Resource(svc route53iface.Route53API, rt arn.ResourceType, rn arn.ResourceName, tags Tags) error {
	// Only hostedzones (and healthchecks, but they are not supported by grafiti)
	// can be tagged
	if rt != arn.Route53HostedZoneRType || rn == "" {
		return nil
	}

	hzTags := make([]*route53.Tag, 0, len(tags))
	for tk, tv := range tags {
		hzTags = append(hzTags, &route53.Tag{
			Key:   aws.String(tk),
			Value: aws.String(tv),
		})
	}

	params := &route53.ChangeTagsForResourceInput{
		AddTags:      hzTags,
		ResourceId:   rn.AWSString(),
		ResourceType: aws.String("hostedzone"),
	}

	pj, err := json.Marshal(params)
	if err != nil {
		if ignoreErrors {
			logger.Debugln("marshal route53 params:", err)
			return nil
		}
		return fmt.Errorf("marshal route53 params: %s", err)
	}
	fmt.Println(string(pj))

	if dryRun {
		return nil
	}

	ctx := aws.BackgroundContext()
	if _, err := svc.ChangeTagsForResourceWithContext(ctx, params); err != nil {
		if ignoreErrors {
			logger.Debugln("route53: tag resources:", err)
			return nil
		}
		return fmt.Errorf("route53: tag resources %s", err)
	}

	return nil
}

func tagARNBucket(svc rgtaiface.ResourceGroupsTaggingAPIAPI, bucket arn.ResourceARNs, tag Tag) error {
	params := &rgta.TagResourcesInput{
		ResourceARNList: bucket.AWSStringSlice(),
		Tags:            map[string]*string{tag.Key: aws.String(tag.Value)},
	}

	pj, err := json.Marshal(params)
	if err != nil {
		if ignoreErrors {
			logger.Debugln("marshal rgta params:", err)
			return nil
		}
		return fmt.Errorf("marshal rgta params: %s", err)
	}
	fmt.Println(string(pj))

	if dryRun {
		return nil
	}

	// Rate limit error is returned if no pause between requests
	time.Sleep(time.Duration(2) * time.Second)
	if _, err := svc.TagResources(params); err != nil {
		if ignoreErrors {
			logger.Debugln("rgta: tag resources:", err)
			return nil
		}
		return fmt.Errorf("rgta: tag resources %s", err)
	}
	return nil
}

func decodeInput(decoder *json.Decoder) (*TagInput, bool, error) {
	var decoded TagInput
	if err := decoder.Decode(&decoded); err != nil {
		if err == io.EOF {
			return &decoded, true, nil
		}
		if ignoreErrors {
			logger.Debugln("decode tag input:", err)
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("decode tag input: %s", err)
	}
	return &decoded, false, nil
}

package deleter

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/coreos/grafiti/arn"
)

func isValidationError(err error) bool {
	aerr, ok := err.(awserr.Error)
	return ok && aerr.Code() == ErrCodeValidationError
}

// AutoScalingGroupDeleter represents an AWS autoscaling group
type AutoScalingGroupDeleter struct {
	Client        autoscalingiface.AutoScalingAPI
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *AutoScalingGroupDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *AutoScalingGroupDeleter) GetClient() autoscalingiface.AutoScalingAPI {
	if rd.Client == nil {
		rd.Client = autoscaling.New(setUpAWSSession())
	}
	return rd.Client
}

// AddResourceNames adds autoscaling group names to ResourceNames
func (rd *AutoScalingGroupDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes autoscaling groups from AWS
func (rd *AutoScalingGroupDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	fmtStr := "Deleted AutoScalingGroup"

	var params *autoscaling.DeleteAutoScalingGroupInput
	for _, n := range rd.ResourceNames {
		if cfg.DryRun {
			fmt.Println(drStr, fmtStr, n)
			continue
		}

		params = &autoscaling.DeleteAutoScalingGroupInput{
			AutoScalingGroupName: n.AWSString(),
			ForceDelete:          aws.Bool(true),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteAutoScalingGroupWithContext(ctx, params)
		if err != nil {
			cfg.logRequestError(arn.AutoScalingGroupRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		cfg.logRequestSuccess(arn.AutoScalingGroupRType, n)
		fmt.Println(fmtStr, n)
	}

	return nil
}

// RequestAutoScalingGroups requests resources from the AWS API and returns
// autoscaling groups by names
func (rd *AutoScalingGroupDeleter) RequestAutoScalingGroups() ([]*autoscaling.Group, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	var lcs []*autoscaling.Group
	// Requesting a batch of resources that contains at least one already deleted
	// resource will return an error and no resources. To guard against this,
	// request them one by one
	for _, name := range rd.ResourceNames {
		var err error
		if lcs, err = rd.requestAutoScalingGroup(name, lcs); err != nil && !isValidationError(err) {
			return lcs, err
		}
	}

	return lcs, nil
}

func (rd *AutoScalingGroupDeleter) requestAutoScalingGroup(rn arn.ResourceName, lcs []*autoscaling.Group) ([]*autoscaling.Group, error) {
	params := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{rn.AWSString()},
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeAutoScalingGroupsWithContext(ctx, params)
	if err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err)
		return lcs, err
	}

	lcs = append(lcs, resp.AutoScalingGroups...)

	return lcs, nil
}

// AutoScalingLaunchConfigurationDeleter represents an AWS launch configuration
type AutoScalingLaunchConfigurationDeleter struct {
	Client        autoscalingiface.AutoScalingAPI
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *AutoScalingLaunchConfigurationDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *AutoScalingLaunchConfigurationDeleter) GetClient() autoscalingiface.AutoScalingAPI {
	if rd.Client == nil {
		rd.Client = autoscaling.New(setUpAWSSession())
	}
	return rd.Client
}

// AddResourceNames adds launch configuration names to ResourceNames
func (rd *AutoScalingLaunchConfigurationDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes a launch configurations from AWS
func (rd *AutoScalingLaunchConfigurationDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	fmtStr := "Deleted LaunchConfiguration"

	var params *autoscaling.DeleteLaunchConfigurationInput
	for _, n := range rd.ResourceNames {
		if cfg.DryRun {
			fmt.Println(drStr, fmtStr, n)
			continue
		}

		params = &autoscaling.DeleteLaunchConfigurationInput{
			LaunchConfigurationName: n.AWSString(),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteLaunchConfigurationWithContext(ctx, params)
		if err != nil {
			cfg.logRequestError(arn.AutoScalingLaunchConfigurationRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		cfg.logRequestSuccess(arn.AutoScalingLaunchConfigurationRType, n)
		fmt.Println(fmtStr, n)
	}

	return nil
}

// RequestAutoScalingLaunchConfigurations requests resources from the AWS API and returns launch
// configurations by names
func (rd *AutoScalingLaunchConfigurationDeleter) RequestAutoScalingLaunchConfigurations() ([]*autoscaling.LaunchConfiguration, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	var lcs []*autoscaling.LaunchConfiguration
	// Requesting a batch of resources that contains at least one already deleted
	// resource will return an error and no resources. To guard against this,
	// request them one by one
	for _, name := range rd.ResourceNames {
		var err error
		if lcs, err = rd.requestAutoScalingLaunchConfiguration(name, lcs); err != nil && !isValidationError(err) {
			return lcs, err
		}
	}

	return lcs, nil
}

func (rd *AutoScalingLaunchConfigurationDeleter) requestAutoScalingLaunchConfiguration(rn arn.ResourceName, lcs []*autoscaling.LaunchConfiguration) ([]*autoscaling.LaunchConfiguration, error) {
	params := &autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: []*string{rn.AWSString()},
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeLaunchConfigurationsWithContext(ctx, params)
	if err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err)
		return lcs, err
	}

	lcs = append(lcs, resp.LaunchConfigurations...)

	return lcs, nil
}

// RequestIAMInstanceProfilesFromLaunchConfigurations retrieves instance profiles from
// launch configuration names
func (rd *AutoScalingLaunchConfigurationDeleter) RequestIAMInstanceProfilesFromLaunchConfigurations() ([]*iam.InstanceProfile, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	lcs, rerr := rd.RequestAutoScalingLaunchConfigurations()
	if rerr != nil {
		return nil, rerr
	}

	// We cannot request instance profiles by their ID's so we must search
	// iteratively with a map
	want, iprs := createInstanceProfileMap(lcs), make([]*iam.InstanceProfile, 0)
	params := new(iam.ListInstanceProfilesInput)
	svc := iam.New(setUpAWSSession())
	for {
		ctx := aws.BackgroundContext()
		resp, err := svc.ListInstanceProfilesWithContext(ctx, params)
		if err != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", err)
			return iprs, err
		}

		for _, ipr := range resp.InstanceProfiles {
			if _, ok := want[aws.StringValue(ipr.InstanceProfileName)]; ok {
				iprs = append(iprs, ipr)
			}
		}

		if !aws.BoolValue(resp.IsTruncated) {
			break
		}

		params.Marker = resp.Marker
	}

	return iprs, nil
}

func createInstanceProfileMap(lcs []*autoscaling.LaunchConfiguration) map[string]struct{} {
	want := map[string]struct{}{}
	var iprName string
	for _, lc := range lcs {
		// The docs say that IAMInstanceProfile can be either an ARN or name; if an
		// ARN, parse out name
		iprName = aws.StringValue(lc.IamInstanceProfile)
		if iprSplit := strings.Split(iprName, "instance-profile/"); len(iprSplit) == 2 && iprSplit[1] != "" {
			iprName = iprSplit[1]
		}
		if _, ok := want[iprName]; !ok {
			want[iprName] = struct{}{}
		}
	}

	return want
}

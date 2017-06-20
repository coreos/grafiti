package deleter

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/coreos/grafiti/arn"
)

// AutoScalingGroupDeleter represents an AWS autoscaling group
type AutoScalingGroupDeleter struct {
	Client        autoscalingiface.AutoScalingAPI
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *AutoScalingGroupDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
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
	if cfg.DryRun {
		for _, n := range rd.ResourceNames {
			fmt.Println(drStr, fmtStr, n)
		}
		return nil
	}

	if rd.Client == nil {
		rd.Client = autoscaling.New(setUpAWSSession())
	}

	var params *autoscaling.DeleteAutoScalingGroupInput
	for _, n := range rd.ResourceNames {
		params = &autoscaling.DeleteAutoScalingGroupInput{
			AutoScalingGroupName: n.AWSString(),
			ForceDelete:          aws.Bool(true),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.Client.DeleteAutoScalingGroupWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.AutoScalingGroupRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, n)
		// Prevent throttling
		time.Sleep(cfg.BackoffTime)
	}

	time.Sleep(time.Duration(30) * time.Second)
	return nil
}

// RequestAutoScalingGroups requests autoscaling groups from the AWS API and returns autoscaling
// groups by names
func (rd *AutoScalingGroupDeleter) RequestAutoScalingGroups() ([]*autoscaling.Group, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	if rd.Client == nil {
		rd.Client = autoscaling.New(setUpAWSSession())
	}

	params := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: rd.ResourceNames.AWSStringSlice(),
	}
	asgs := make([]*autoscaling.Group, 0)

	for {
		ctx := aws.BackgroundContext()
		resp, err := rd.Client.DescribeAutoScalingGroupsWithContext(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, asg := range resp.AutoScalingGroups {
			asgs = append(asgs, asg)
		}

		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}

		params.NextToken = resp.NextToken
	}

	return asgs, nil
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
	if cfg.DryRun {
		for _, n := range rd.ResourceNames {
			fmt.Println(drStr, fmtStr, n)
		}
		return nil
	}

	if rd.Client == nil {
		rd.Client = autoscaling.New(setUpAWSSession())
	}

	var params *autoscaling.DeleteLaunchConfigurationInput
	for _, n := range rd.ResourceNames {
		params = &autoscaling.DeleteLaunchConfigurationInput{
			LaunchConfigurationName: n.AWSString(),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.Client.DeleteLaunchConfigurationWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.AutoScalingLaunchConfigurationRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, n)
		// Prevent throttling
		time.Sleep(cfg.BackoffTime)
	}

	return nil
}

// RequestAutoScalingLaunchConfigurations requests resources from the AWS API and returns launch
// configurations by names
func (rd *AutoScalingLaunchConfigurationDeleter) RequestAutoScalingLaunchConfigurations() ([]*autoscaling.LaunchConfiguration, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	if rd.Client == nil {
		rd.Client = autoscaling.New(setUpAWSSession())
	}

	params := &autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: rd.ResourceNames.AWSStringSlice(),
	}
	lcs := make([]*autoscaling.LaunchConfiguration, 0)

	for {
		ctx := aws.BackgroundContext()
		resp, err := rd.Client.DescribeLaunchConfigurationsWithContext(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, lc := range resp.LaunchConfigurations {
			lcs = append(lcs, lc)
		}

		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}

		params.NextToken = resp.NextToken
	}

	return lcs, nil
}

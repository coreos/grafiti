package deleter

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/coreos/grafiti/arn"
	"github.com/coreos/grafiti/describe"
)

const (
	drStr  = "(dry-run)"
	drCode = "DryRunOperation"
)

// DeleteConfig holds configuration info for resource deletion
type DeleteConfig struct {
	IgnoreErrors bool
	DryRun       bool
	ResourceType string
	AWSSession   *session.Session
}

// DelFunc takes a *DeleteConfig and an array of strings and
// returns an error
type DelFunc func(*DeleteConfig, *[]string) error

func deleteAutoScalingGroupsByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	svc := autoscaling.New(cfg.AWSSession)
	fmtStr := "Deleted AutoScalingGroup"
	if cfg.DryRun {
		for _, id := range *ids {
			fmt.Println(drStr, fmtStr, id)
		}
		return nil
	}
	var params *autoscaling.DeleteAutoScalingGroupInput
	for _, id := range *ids {
		params = &autoscaling.DeleteAutoScalingGroupInput{
			AutoScalingGroupName: aws.String(id),
			ForceDelete:          aws.Bool(true),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteAutoScalingGroupWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			return err
		}
		fmt.Println(fmtStr, id)
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	time.Sleep(time.Duration(30) * time.Second)
	return nil
}

func deleteAutoScalingLaunchConfigurationsByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	svc := autoscaling.New(cfg.AWSSession)
	fmtStr := "Deleted LaunchConfiguration"
	if cfg.DryRun {
		for _, id := range *ids {
			fmt.Println(drStr, fmtStr, id)
		}
		return nil
	}
	var params *autoscaling.DeleteLaunchConfigurationInput
	for _, id := range *ids {
		params = &autoscaling.DeleteLaunchConfigurationInput{
			LaunchConfigurationName: aws.String(id),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteLaunchConfigurationWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			return err
		}
		fmt.Println(fmtStr, id)
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	return nil
}

// NOTE: ids is an array of arrays, in which each 2nd dimension has the
// autoscaling group name (id[1]) and policy name (id[2])
func deleteAutoScalingPoliciesByIDs(cfg *DeleteConfig, ids *[][]string) error {
	if ids == nil {
		return nil
	}

	svc := autoscaling.New(cfg.AWSSession)
	fmtStr := "Deleted AutoScalingPolicy"
	if cfg.DryRun {
		for _, id := range *ids {
			fmt.Println(drStr, fmtStr, id[2])
		}
		return nil
	}
	var params *autoscaling.DeletePolicyInput
	for _, id := range *ids {
		params = &autoscaling.DeletePolicyInput{
			AutoScalingGroupName: aws.String(id[1]),
			PolicyName:           aws.String(id[2]),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeletePolicyWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			return err
		}
		fmt.Println(fmtStr, id[2])
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	return nil
}

func deleteEC2InstancesByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	ress, _ := describe.GetEC2InstanceReservationsByIDs(ids)
	iIDs := make([]string, 0)
	if ress != nil {
		for _, res := range *ress {
			for _, ins := range res.Instances {
				code := *ins.State.Code
				// If instance is shutting down (32) or terminated (48), skip
				if code == 32 || code == 48 {
					continue
				}
				iIDs = append(iIDs, *ins.InstanceId)
			}
		}
	}

	if len(iIDs) == 0 {
		return nil
	}

	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Terminated EC2 Instance"

	params := &ec2.TerminateInstancesInput{
		InstanceIds: aws.StringSlice(iIDs),
		DryRun:      aws.Bool(cfg.DryRun),
	}
	ctx := aws.BackgroundContext()
	_, err := svc.TerminateInstancesWithContext(ctx, params)
	if err != nil {
		if cfg.IgnoreErrors {
			fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
			return nil
		}
		aerr, ok := err.(awserr.Error)
		if ok {
			aerrCode := aerr.Code()
			if aerrCode == drCode {
				for _, id := range iIDs {
					fmt.Println(drStr, fmtStr, id)
				}
				return nil
			}
		}
		return err
	}
	for _, id := range iIDs {
		fmt.Println(fmtStr, id)
	}
	// Instances take awhile to shut down
	time.Sleep(time.Duration(2) * time.Minute)
	return nil
}

func deleteRoute53ResourceRecordSets(cfg *DeleteConfig, hzID string, rrs *[]*route53.ResourceRecordSet) error {
	if rrs == nil {
		return nil
	}

	fmtStr := "Deleted Route53 ResourceRecordSet"
	if cfg.DryRun {
		for _, rs := range *rrs {
			fmt.Printf("%s %s %s (HZ %s)\n", drStr, fmtStr, *rs.Name, hzID)
		}
		return nil
	}
	changes := make([]*route53.Change, 0, len(*rrs))
	for _, rs := range *rrs {
		// Cannot delete NS/SOA type record sets
		if *rs.Type == "NS" || *rs.Type == "SOA" {
			continue
		}
		changes = append(changes, &route53.Change{
			Action:            aws.String(route53.ChangeActionDelete),
			ResourceRecordSet: rs,
		})
	}

	if len(changes) == 0 {
		return nil
	}

	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  &route53.ChangeBatch{Changes: changes},
		HostedZoneId: aws.String(hzID),
	}

	svc := route53.New(cfg.AWSSession)
	ctx := aws.BackgroundContext()
	_, err := svc.ChangeResourceRecordSetsWithContext(ctx, params)
	if err != nil {
		if cfg.IgnoreErrors {
			fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
			return nil
		}
		return err
	}
	for _, rs := range *rrs {
		fmt.Printf("%s %s (HZ %s)\n", fmtStr, *rs.Name, hzID)
	}
	return nil
}

// NOTE: must delete all non-default resource record sets before deleting a
// hosted zone. Will receive HostedZoneNotEmpty otherwise
func deleteRoute53HostedZonesByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	for _, id := range *ids {
		rrs, rerr := describe.GetRoute53ResourceRecordSetsByHZID(id)
		if rerr != nil || rrs == nil {
			continue
		}
		_ = deleteRoute53ResourceRecordSets(cfg, id, rrs)
	}

	svc := route53.New(cfg.AWSSession)
	fmtStr := "Deleted Route53 HostedZone"
	if cfg.DryRun {
		for _, id := range *ids {
			fmt.Println(drStr, fmtStr, id)
		}
		return nil
	}
	var params *route53.DeleteHostedZoneInput
	for _, id := range *ids {
		params = &route53.DeleteHostedZoneInput{
			Id: aws.String(id),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteHostedZoneWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			return err
		}
		fmt.Println(fmtStr, id)
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	return nil
}

// Deletes only private hosted zones
func deleteRoute53HostedZonesPrivate(cfg *DeleteConfig, ids *[]string) error {
	hzs, err := describe.GetRoute53HostedZonesByIDs(ids)
	if err != nil && hzs == nil {
		return nil
	}
	hzIDs := make([]string, 0)
	for _, hz := range *hzs {
		if *hz.Config.PrivateZone {
			hzSplit := strings.Split(*hz.Id, "/hostedzone/")
			if len(hzSplit) != 2 {
				continue
			}
			hzIDs = append(hzIDs, hzSplit[1])
		}
	}

	return deleteRoute53HostedZonesByIDs(cfg, &hzIDs)
}

// DeleteAWSResourcesByIDs deletes resources by ID's that are supported by the aws-sdk-go
// CloudTrail API
func DeleteAWSResourcesByIDs(cfg *DeleteConfig, ids *[]string) error {
	switch cfg.ResourceType {
	case arn.AutoScalingGroupRType:
		return deleteAutoScalingGroupsByIDs(cfg, ids)
	case arn.AutoScalingLaunchConfigurationRType:
		return deleteAutoScalingLaunchConfigurationsByIDs(cfg, ids)
	case arn.EC2AmiRType:
	case arn.EC2CustomerGatewayRType:
	case arn.EC2EIPRType:
	case arn.EC2EIPAssociationRType:
	case arn.EC2InstanceRType:
		return deleteEC2InstancesByIDs(cfg, ids)
	case arn.EC2InternetGatewayRType:
	case arn.EC2NatGatewayRType:
	case arn.EC2NetworkACLRType:
	case arn.EC2NetworkInterfaceRType:
	case arn.EC2RouteTableAssociationRType:
	case arn.EC2RouteTableRType:
	case arn.EC2SecurityGroupRType:
	case arn.EC2SnapshotRType:
	case arn.EC2SubnetRType:
	case arn.EC2VolumeRType:
	case arn.EC2VPCRType:
	case arn.ElasticLoadBalancingLoadBalancerRType:
	case arn.IAMInstanceProfileRType:
	case arn.IAMRoleRType:
	case arn.IAMUserRType:
	case arn.S3BucketRType:
	case arn.Route53HostedZoneRType:
		return deleteRoute53HostedZonesPrivate(cfg, ids)
	}
	fmt.Printf("%s is not a supported ResourceType\n", cfg.ResourceType)
	return nil
}

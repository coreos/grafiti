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

func deleteEC2AMIsByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Deregistered EC2 AMI"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}
	var params *ec2.DeregisterImageInput
	for _, id := range *ids {
		params = &ec2.DeregisterImageInput{
			ImageId: aws.String(id),
			DryRun:  aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeregisterImageWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			aerr, ok := err.(awserr.Error)
			if ok {
				aerrCode := aerr.Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
			}
			return err
		}
		fmt.Println(fmtStr, id)
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	return nil
}

func deleteEC2EIPAssocationsByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Disassociated EC2 ElasticIP"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}
	var params *ec2.DisassociateAddressInput
	for _, id := range *ids {
		params = &ec2.DisassociateAddressInput{
			AssociationId: aws.String(id),
			DryRun:        aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DisassociateAddressWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			aerr, ok := err.(awserr.Error)
			if ok {
				aerrCode := aerr.Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
			}
			return err
		}
		fmt.Println(fmtStr, id)
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	return nil
}

func deleteEC2EIPAddresses(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}
	// Format of ids should be
	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Released EC2 ElasticIP"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}
	var params *ec2.ReleaseAddressInput
	for _, id := range *ids {
		params = &ec2.ReleaseAddressInput{
			AllocationId: aws.String(id),
			DryRun:       aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.ReleaseAddressWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			aerr, ok := err.(awserr.Error)
			if ok {
				aerrCode := aerr.Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
			}
			return err
		}
		fmt.Println(fmtStr, id)
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

func deleteEC2InternetGatewayAttachments(cfg *DeleteConfig, igws *[]*ec2.InternetGateway) error {
	if igws == nil {
		return nil
	}

	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Detached EC2 InternetGateway"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	var params *ec2.DetachInternetGatewayInput
	for _, igw := range *igws {
		params = &ec2.DetachInternetGatewayInput{
			InternetGatewayId: igw.InternetGatewayId,
			DryRun:            aws.Bool(cfg.DryRun),
		}

		for _, a := range igw.Attachments {
			params.SetVpcId(*a.VpcId)
			ctx := aws.BackgroundContext()
			_, err := svc.DetachInternetGatewayWithContext(ctx, params)
			if err != nil {
				if cfg.IgnoreErrors {
					fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
					continue
				}
				aerr, ok := err.(awserr.Error)
				if ok {
					aerrCode := aerr.Code()
					if aerrCode == drCode {
						fmt.Println(fmtStr, *igw.InternetGatewayId, "from", *a.VpcId)
						continue
					}
				}
				return err
			}
			fmt.Printf("%s %s from VPC %s\n", fmtStr, *igw.InternetGatewayId, *a.VpcId)
			// Prevent throttling
			time.Sleep(time.Duration(500) * time.Millisecond)
		}
	}
	return nil
}

// NOTE: must detach all internet gateways from vpc's before deletion
func deleteEC2InternetGatewaysByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	igws, ierr := describe.GetEC2InternetGatewaysByIDs(ids)
	if ierr != nil {
		return nil
	}

	// Detach internet gateways from all vpc's
	if verr := deleteEC2InternetGatewayAttachments(cfg, igws); verr != nil {
		return verr
	}

	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Deleted EC2 InternetGateway"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}
	var params *ec2.DeleteInternetGatewayInput
	for _, id := range *ids {
		params = &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: aws.String(id),
			DryRun:            aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteInternetGatewayWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			aerr, ok := err.(awserr.Error)
			if ok {
				aerrCode := aerr.Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
			}
			return err
		}
		fmt.Println(fmtStr, id)
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	return nil
}

func deleteEC2NatGatewaysByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Deleted EC2 NatGateway"
	if cfg.DryRun {
		for _, id := range *ids {
			fmt.Println(drStr, fmtStr, id)
		}
		return nil
	}
	var params *ec2.DeleteNatGatewayInput
	for _, id := range *ids {
		params = &ec2.DeleteNatGatewayInput{
			NatGatewayId: aws.String(id),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteNatGatewayWithContext(ctx, params)
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
	// Wait for NAT Gateways to delete
	time.Sleep(time.Duration(30) * time.Second)
	return nil
}

// Network interfaces require detachment before deletion
func detachEC2NetworkInterfacesByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Detached EC2 NetworkInterface"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}
	var params *ec2.DetachNetworkInterfaceInput

	for _, id := range *ids {
		params = &ec2.DetachNetworkInterfaceInput{
			AttachmentId: aws.String(id),
			Force:        aws.Bool(true),
			DryRun:       aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DetachNetworkInterfaceWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			aerr, ok := err.(awserr.Error)
			if ok {
				aerrCode := aerr.Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
			}
			return err
		}
		fmt.Println(fmtStr, id)
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	return nil
}

func deleteEC2NetworkInterfacesByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	// To delete a network interface, all attachments must be deleted first.
	// DescribeNetworkInterfaces will find attachment ID's.
	enis, nierr := describe.GetEC2NetworkInterfacesByIDs(ids)
	if nierr != nil {
		return nierr
	}

	if enis != nil {
		eniaIDs := make([]string, 0)
		iIDs := make([]string, 0)
		for _, ni := range *enis {
			if ni.Attachment != nil && ni.Attachment.AttachmentId != nil {
				// eth0 is the primary eni and cannot be detached. Must delete underlying
				// instance
				if ni.Attachment.DeviceIndex != nil && *ni.Attachment.DeviceIndex == 0 {
					iIDs = append(iIDs, *ni.Attachment.InstanceId)
					continue
				}
				eniaIDs = append(eniaIDs, *ni.Attachment.AttachmentId)
			}
		}
		// Remove blocking instances
		_ = deleteEC2InstancesByIDs(cfg, &iIDs)

		if eniaIDs != nil {
			if nerr := detachEC2NetworkInterfacesByIDs(cfg, &eniaIDs); nerr != nil {
				return nerr
			}
		}
	}

	// Format of ids should be
	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Deleted EC2 NetworkInterface"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}
	var params *ec2.DeleteNetworkInterfaceInput

	for _, id := range *ids {
		if id == "" {
			continue
		}
		params = &ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: aws.String(id),
			DryRun:             aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteNetworkInterfaceWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			aerr, ok := err.(awserr.Error)
			if ok {
				aerrCode := aerr.Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
			}
			return err
		}
		fmt.Println(fmtStr, id)
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	return nil
}

func deleteEC2SnapshotsByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Deleted EC2 Snapshot"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}
	var params *ec2.DeleteSnapshotInput
	for _, id := range *ids {
		params = &ec2.DeleteSnapshotInput{
			SnapshotId: aws.String(id),
			DryRun:     aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteSnapshotWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			aerr, ok := err.(awserr.Error)
			if ok {
				aerrCode := aerr.Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
			}
			return err
		}
		fmt.Println(fmtStr, id)
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	return nil
}

func deleteEC2VolumesByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Deleted EC2 Volume"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}
	var params *ec2.DeleteVolumeInput
	for _, id := range *ids {
		params = &ec2.DeleteVolumeInput{
			VolumeId: aws.String(id),
			DryRun:   aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteVolumeWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			aerr, ok := err.(awserr.Error)
			if ok {
				aerrCode := aerr.Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
			}
			return err
		}
		fmt.Println(fmtStr, id)
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
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
		return deleteEC2AMIsByIDs(cfg, ids)
	case arn.EC2CustomerGatewayRType:
	case arn.EC2EIPRType:
		return deleteEC2EIPAddresses(cfg, ids)
	case arn.EC2EIPAssociationRType:
		return deleteEC2EIPAssocationsByIDs(cfg, ids)
	case arn.EC2InstanceRType:
		return deleteEC2InstancesByIDs(cfg, ids)
	case arn.EC2InternetGatewayRType:
		return deleteEC2InternetGatewaysByIDs(cfg, ids)
	case arn.EC2NatGatewayRType:
		return deleteEC2NatGatewaysByIDs(cfg, ids)
	case arn.EC2NetworkACLRType:
	case arn.EC2NetworkInterfaceRType:
		return deleteEC2NetworkInterfacesByIDs(cfg, ids)
	case arn.EC2RouteTableAssociationRType:
	case arn.EC2RouteTableRType:
	case arn.EC2SecurityGroupRType:
	case arn.EC2SnapshotRType:
		return deleteEC2SnapshotsByIDs(cfg, ids)
	case arn.EC2SubnetRType:
	case arn.EC2VolumeRType:
		return deleteEC2VolumesByIDs(cfg, ids)
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

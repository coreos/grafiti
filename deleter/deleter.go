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
	"github.com/aws/aws-sdk-go/service/iam"
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

func deleteEC2CustomerGatewaysByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Deleted EC2 CustomerGateway"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}
	var params *ec2.DeleteCustomerGatewayInput
	for _, id := range *ids {
		params = &ec2.DeleteCustomerGatewayInput{
			CustomerGatewayId: aws.String(id),
			DryRun:            aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteCustomerGatewayWithContext(ctx, params)
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

func deleteEC2NetworkACLEntries(cfg *DeleteConfig, acls *[]*ec2.NetworkAcl) error {
	if acls == nil {
		return nil
	}
	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Deleted EC2 NetworkAcl Association"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	var params *ec2.DeleteNetworkAclEntryInput
	for _, acl := range *acls {
		if *acl.IsDefault {
			continue
		}

		for _, a := range acl.Entries {
			params = &ec2.DeleteNetworkAclEntryInput{
				NetworkAclId: acl.NetworkAclId,
				RuleNumber:   a.RuleNumber,
				Egress:       a.Egress,
				DryRun:       aws.Bool(cfg.DryRun),
			}
			ctx := aws.BackgroundContext()
			_, err := svc.DeleteNetworkAclEntryWithContext(ctx, params)
			if err != nil {
				if cfg.IgnoreErrors {
					fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
					continue
				}
				aerr, ok := err.(awserr.Error)
				if ok {
					aerrCode := aerr.Code()
					if aerrCode == drCode {
						fmt.Printf("%s %s Entry %d\n", fmtStr, *acl.NetworkAclId, *a.RuleNumber)
						continue
					}
				}
				return err
			}
			fmt.Printf("%s %s Entry %d\n", fmtStr, *acl.NetworkAclId, *a.RuleNumber)
			// Prevent throttling
			time.Sleep(time.Duration(500) * time.Millisecond)
		}
	}
	return nil
}

func deleteEC2NetworkACLsByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	acls, aerr := describe.GetEC2NetworkACLsByIDs(ids)
	if aerr != nil {
		return aerr
	}

	// First delete network acl entries
	if eerr := deleteEC2NetworkACLEntries(cfg, acls); eerr != nil {
		return eerr
	}

	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Deleted EC2 NetworkAcl"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	var params *ec2.DeleteNetworkAclInput
	for _, acl := range *acls {
		params = &ec2.DeleteNetworkAclInput{
			NetworkAclId: acl.NetworkAclId,
			DryRun:       aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteNetworkAclWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			aerr, ok := err.(awserr.Error)
			if ok {
				aerrCode := aerr.Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, *acl.NetworkAclId)
					continue
				}
			}
			return err
		}
		fmt.Println(fmtStr, *acl.NetworkAclId)
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
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

func deleteEC2RouteTableRoutes(cfg *DeleteConfig, rtID *string, rs *[]*ec2.Route) error {
	if rs == nil {
		return nil
	}
	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Deleted RouteTable Route"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}
	var params *ec2.DeleteRouteInput
	for _, r := range *rs {
		if r.GatewayId != nil && *r.GatewayId == "local" {
			continue
		}
		params = &ec2.DeleteRouteInput{
			DestinationCidrBlock: r.DestinationCidrBlock,
			RouteTableId:         rtID,
			DryRun:               aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteRouteWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			aerr, ok := err.(awserr.Error)
			if ok {
				aerrCode := aerr.Code()
				if aerrCode == drCode {
					fmt.Printf("%s: Dst CIDR Block %s\n", fmtStr, *r.DestinationCidrBlock)
					continue
				}
			}
			return err
		}
		fmt.Printf("%s: Dst CIDR Block %s\n", fmtStr, *r.DestinationCidrBlock)
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	return nil
}

func deleteEC2RouteTableAssociationsByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Deleted RouteTable Association"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}
	var params *ec2.DisassociateRouteTableInput
	for _, id := range *ids {
		params = &ec2.DisassociateRouteTableInput{
			AssociationId: aws.String(id),
			DryRun:        aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DisassociateRouteTableWithContext(ctx, params)
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

// NOTE: can only delete a route table once all subnets have been disassociated,
// and and all routes have been deleted. Cannot delete the main (default) route
// table
func deleteEC2RouteTablesByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	// Ensure all routes are deleted
	rts, err := describe.GetEC2RouteTablesByIDs(ids)
	if err != nil {
		return err
	}
	if rts == nil {
		return nil
	}
	sras := make([]*ec2.RouteTableAssociation, 0)
	for _, rt := range *rts {
		for _, a := range rt.Associations {
			if *a.Main {
				continue
			}
			sras = append(sras, a)
		}
		if rerr := deleteEC2RouteTableRoutes(cfg, rt.RouteTableId, &rt.Routes); rerr != nil {
			return rerr
		}
	}

	// Delete route table
	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Deleted EC2 RouteTable"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}
	var params *ec2.DeleteRouteTableInput
	for _, rt := range *rts {

		params = &ec2.DeleteRouteTableInput{
			RouteTableId: rt.RouteTableId,
			DryRun:       aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteRouteTableWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			aerr, ok := err.(awserr.Error)
			if ok {
				aerrCode := aerr.Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, *rt.RouteTableId)
					continue
				}
			}
			return err
		}
		fmt.Println(fmtStr, *rt.RouteTableId)
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	return nil
}

const defaultGroupName = "default"

func deleteEC2SecurityGroupIngressRules(cfg *DeleteConfig, sgs *[]*ec2.SecurityGroup) error {
	svc := ec2.New(cfg.AWSSession)
	if sgs == nil {
		return nil
	}

	fmtStr := "Deleted EC2 SecurityGroup Ingress Rule"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	var params *ec2.RevokeSecurityGroupIngressInput
	for _, sg := range *sgs {
		if *sg.GroupName == defaultGroupName {
			continue
		}
		params = &ec2.RevokeSecurityGroupIngressInput{
			GroupId:       sg.GroupId,
			IpPermissions: sg.IpPermissions,
			DryRun:        aws.Bool(cfg.DryRun),
		}
		ctx := aws.BackgroundContext()
		_, err := svc.RevokeSecurityGroupIngressWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			aerr, ok := err.(awserr.Error)
			if ok {
				aerrCode := aerr.Code()
				if aerrCode == drCode {
					fmt.Printf("%s for %s\n", fmtStr, *sg.GroupId)
					continue
				}
			}
			return err
		}
		fmt.Printf("%s for %s\n", fmtStr, *sg.GroupId)
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	return nil
}

func deleteEC2SecurityGroupEgressRules(cfg *DeleteConfig, sgs *[]*ec2.SecurityGroup) error {
	svc := ec2.New(cfg.AWSSession)
	if sgs == nil {
		return nil
	}

	fmtStr := "Deleted EC2 SecurityGroup Egress Rule"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	var params *ec2.RevokeSecurityGroupEgressInput
	for _, sg := range *sgs {
		if *sg.GroupName == defaultGroupName {
			continue
		}
		params = &ec2.RevokeSecurityGroupEgressInput{
			GroupId:       sg.GroupId,
			IpPermissions: sg.IpPermissionsEgress,
			DryRun:        aws.Bool(cfg.DryRun),
		}
		ctx := aws.BackgroundContext()
		_, err := svc.RevokeSecurityGroupEgressWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			aerr, ok := err.(awserr.Error)
			if ok {
				aerrCode := aerr.Code()
				if aerrCode == drCode {
					fmt.Printf("%s for %s\n", fmtStr, *sg.GroupId)
					continue
				}
			}
			return err
		}
		fmt.Printf("%s for %s\n", fmtStr, *sg.GroupId)
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	return nil
}

// NOTE: all security group references must be removed before deleting before
// deleting a security group
func deleteEC2SecurityGroupsByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	sgs, rerr := describe.GetEC2SecurityGroupsByIDs(ids)
	if rerr != nil {
		return rerr
	}
	if sgs == nil {
		return nil
	}

	// Delete ingress/egress rules (security group references)
	if ierr := deleteEC2SecurityGroupIngressRules(cfg, sgs); ierr != nil {
		return ierr
	}
	if eerr := deleteEC2SecurityGroupEgressRules(cfg, sgs); eerr != nil {
		return eerr
	}

	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Deleted EC2 SecurityGroup"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}
	var params *ec2.DeleteSecurityGroupInput
	for _, sg := range *sgs {
		if *sg.GroupName == defaultGroupName {
			continue
		}
		params = &ec2.DeleteSecurityGroupInput{
			GroupId: sg.GroupId,
			DryRun:  aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteSecurityGroupWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			aerr, ok := err.(awserr.Error)
			if ok {
				aerrCode := aerr.Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, *sg.GroupId)
					continue
				}
			}
			return err
		}
		fmt.Println(fmtStr, *sg.GroupId)
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

// NOTE: ensure all network interfaces and network acl's are disassociated
func deleteEC2SubnetsByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Deleted EC2 Subnet"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}
	var params *ec2.DeleteSubnetInput
	for _, id := range *ids {
		params = &ec2.DeleteSubnetInput{
			SubnetId: aws.String(id),
			DryRun:   aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteSubnetWithContext(ctx, params)
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

func deleteEC2VPCCIDRBlocks(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	vpcs, verr := describe.GetEC2VPCsByIDs(ids)
	if verr != nil {
		return verr
	}
	if vpcs == nil {
		return nil
	}

	svc := ec2.New(cfg.AWSSession)
	if cfg.DryRun {
		for _, id := range *ids {
			fmt.Printf("%s Deleted EC2 VPC %s CIDRBlockAssociation\n", drStr, id)
		}
		return nil
	}

	var params *ec2.DisassociateVpcCidrBlockInput
	for _, vpc := range *vpcs {
		for _, b := range vpc.Ipv6CidrBlockAssociationSet {
			params = &ec2.DisassociateVpcCidrBlockInput{
				AssociationId: b.AssociationId,
			}
			ctx := aws.BackgroundContext()
			_, err := svc.DisassociateVpcCidrBlockWithContext(ctx, params)
			if err != nil {
				if cfg.IgnoreErrors {
					fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
					continue
				}
				return err
			}
			fmt.Printf("%s Deleted EC2 VPC %s CIDRBlockAssociation %s\n", drStr, *vpc.VpcId, *b.AssociationId)
			// Prevent throttling
			time.Sleep(time.Duration(500) * time.Millisecond)
		}
	}
	return nil
}

func deleteEC2VPCsByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	// Disassociate vpc cidr blocks
	if cerr := deleteEC2VPCCIDRBlocks(cfg, ids); cerr != nil {
		return cerr
	}

	// Now delete VPC itself
	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Deleted EC2 VPC"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}
	var params *ec2.DeleteVpcInput
	for _, id := range *ids {
		params = &ec2.DeleteVpcInput{
			VpcId:  aws.String(id),
			DryRun: aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteVpcWithContext(ctx, params)
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

func removeIAMRolesFromInstanceProfilesByIPrs(cfg *DeleteConfig, iprs *[]*iam.InstanceProfile) error {
	if iprs == nil {
		return nil
	}

	svc := iam.New(cfg.AWSSession)
	if cfg.DryRun {
		for _, ipr := range *iprs {
			fmt.Println(drStr, "Removed Role from IAM InstanceProfile", *ipr.InstanceProfileName)
		}
		return nil
	}
	var params *iam.RemoveRoleFromInstanceProfileInput
	for _, ipr := range *iprs {
		for _, rl := range ipr.Roles {
			params = &iam.RemoveRoleFromInstanceProfileInput{
				InstanceProfileName: ipr.InstanceProfileName,
				RoleName:            rl.RoleName,
			}

			ctx := aws.BackgroundContext()
			_, err := svc.RemoveRoleFromInstanceProfileWithContext(ctx, params)
			if err != nil {
				if cfg.IgnoreErrors {
					fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
					continue
				}
				return err
			}
			fmt.Printf("Removed Role %s from IAM InstanceProfile %s\n", *ipr.InstanceProfileName, *rl.RoleName)
			// Prevent throttling
			time.Sleep(time.Duration(500) * time.Millisecond)
		}
	}
	return nil
}

// NOTE: must delete roles from instance profile before deleting roles. Must
// be done in this step because of only profiles contain role info, not visa versa.
func deleteIAMInstanceProfilesByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	iprs, ierr := describe.GetIAMInstanceProfilesByIDs(ids)
	if ierr != nil {
		return ierr
	}
	if iprs == nil {
		return nil
	}

	// Delete roles from instance profiles
	_ = removeIAMRolesFromInstanceProfilesByIPrs(cfg, iprs)

	svc := iam.New(cfg.AWSSession)
	fmtStr := "Deleted IAM InstanceProfile"
	if cfg.DryRun {
		for _, id := range *ids {
			fmt.Println(drStr, fmtStr, id)
		}
		return nil
	}
	var params *iam.DeleteInstanceProfileInput
	for _, ipr := range *iprs {
		params = &iam.DeleteInstanceProfileInput{
			InstanceProfileName: aws.String(*ipr.InstanceProfileName),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteInstanceProfileWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			return err
		}
		fmt.Println(fmtStr, *ipr.InstanceProfileName)
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	return nil
}

func deleteIAMRolePoliciesByRoles(cfg *DeleteConfig, rpsMap *map[string][]string) error {
	if rpsMap == nil {
		return nil
	}
	svc := iam.New(cfg.AWSSession)
	fmtStr := "Deleted IAM RolePolicy"
	if cfg.DryRun {
		for rn, rpns := range *rpsMap {
			for _, rp := range rpns {
				fmt.Printf("%s %s %s for IAM Role %s\n", drStr, fmtStr, rp, rn)
			}
		}
		return nil
	}
	var params *iam.DeleteRolePolicyInput
	for rn, rpns := range *rpsMap {
		for _, rp := range rpns {
			params = &iam.DeleteRolePolicyInput{
				RoleName:   aws.String(rn),
				PolicyName: aws.String(rp),
			}

			ctx := aws.BackgroundContext()
			_, err := svc.DeleteRolePolicyWithContext(ctx, params)
			if err != nil {
				if cfg.IgnoreErrors {
					fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
					continue
				}
				return err
			}
			fmt.Println(fmtStr, rp)
			// Prevent throttling
			time.Sleep(time.Duration(500) * time.Millisecond)
		}
	}
	return nil
}

func deleteIAMRolesByNames(cfg *DeleteConfig, ns *[]string) error {
	if ns == nil {
		return nil
	}

	// Delete role policies
	rps, lerr := describe.GetIAMRolePoliciesByRoleNames(ns)
	if lerr != nil {
		return lerr
	}
	if perr := deleteIAMRolePoliciesByRoles(cfg, rps); perr != nil {
		return perr
	}

	// Delete roles
	svc := iam.New(cfg.AWSSession)
	fmtStr := "Deleted IAM Role"
	if cfg.DryRun {
		for _, id := range *ns {
			fmt.Println(drStr, fmtStr, id)
		}
		return nil
	}
	var params *iam.DeleteRoleInput
	for _, n := range *ns {
		params = &iam.DeleteRoleInput{
			RoleName: aws.String(n),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteRoleWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			return err
		}
		fmt.Println(fmtStr, n)
		// Prevent throttling
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	return nil
}

func deleteIAMUsersByNames(cfg *DeleteConfig, ns *[]string) error {
	if ns == nil {
		return nil
	}

	svc := iam.New(cfg.AWSSession)
	fmtStr := "Deleted IAM User"
	if cfg.DryRun {
		for _, n := range *ns {
			fmt.Println(drStr, fmtStr, n)
		}
		return nil
	}
	var params *iam.DeleteUserInput
	for _, n := range *ns {
		params = &iam.DeleteUserInput{
			UserName: aws.String(n),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteUserWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
				continue
			}
			return err
		}
		fmt.Println(fmtStr, n)
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
		return deleteEC2CustomerGatewaysByIDs(cfg, ids)
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
		return deleteEC2NetworkACLsByIDs(cfg, ids)
	case arn.EC2NetworkInterfaceRType:
		return deleteEC2NetworkInterfacesByIDs(cfg, ids)
	case arn.EC2RouteTableAssociationRType:
		return deleteEC2RouteTableAssociationsByIDs(cfg, ids)
	case arn.EC2RouteTableRType:
		return deleteEC2RouteTablesByIDs(cfg, ids)
	case arn.EC2SecurityGroupRType:
		return deleteEC2SecurityGroupsByIDs(cfg, ids)
	case arn.EC2SnapshotRType:
		return deleteEC2SnapshotsByIDs(cfg, ids)
	case arn.EC2SubnetRType:
		return deleteEC2SubnetsByIDs(cfg, ids)
	case arn.EC2VolumeRType:
		return deleteEC2VolumesByIDs(cfg, ids)
	case arn.EC2VPCRType:
		return deleteEC2VPCsByIDs(cfg, ids)
	case arn.ElasticLoadBalancingLoadBalancerRType:
	case arn.IAMInstanceProfileRType:
		return deleteIAMInstanceProfilesByIDs(cfg, ids)
	case arn.IAMRoleRType:
		return deleteIAMRolesByNames(cfg, ids)
	case arn.IAMUserRType:
		return deleteIAMUsersByNames(cfg, ids)
	case arn.S3BucketRType:
	case arn.Route53HostedZoneRType:
		return deleteRoute53HostedZonesPrivate(cfg, ids)
	}
	fmt.Printf("%s is not a supported ResourceType\n", cfg.ResourceType)
	return nil
}

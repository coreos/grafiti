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
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/coreos/grafiti/arn"
	"github.com/coreos/grafiti/describe"
)

const (
	drStr    = "(dry-run)"
	drCode   = "DryRunOperation"
	invpCode = "InvalidParameterValue"
	mprmCode = "MissingParameter"
	depvCode = "DependencyViolation"
	nopCode  = "OperationNotPermitted"
	riuCode  = "ResourceInUse"
	nseCode  = "NoSuchEntity"
	dcfCode  = "DeleteConflict"
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

// TODO: add write-to-file step for subsequent removal of resources (parsable JSON)
func handleAWSError(ignoreError bool, err error) (bool, error) {
	if ignoreError {
		fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
		return false, nil
	}
	_, ok := err.(awserr.Error)
	return ok, err
}

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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == depvCode || aerrCode == invpCode || aerrCode == riuCode {
					fmt.Printf("Could not delete AutoScalingGroup %s (%s)\n", id, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == depvCode || aerrCode == invpCode || aerrCode == riuCode {
					fmt.Printf("Could not delete LaunchConfiguration %s (%s)\n", id, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s (%s)\n", id, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id[2])
	}
	return nil
}

func deleteEC2AmisByIDs(cfg *DeleteConfig, ids *[]string) error {
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s (%s)\n", id, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
	}
	// TODO: find and delete any snapshots after deregistering AMI
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s (%s)\n", id, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
	}
	return nil
}

func deleteEC2EIPAssocationsByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}
	// Format of ids should be
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
		_, derr := svc.DisassociateAddressWithContext(ctx, params)
		if derr != nil {
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, derr)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s (%s)\n", id, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
	}
	return nil
}

// NOTE: no ARN
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
		_, rerr := svc.ReleaseAddressWithContext(ctx, params)
		if rerr != nil {
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, rerr)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s (%s)\n", id, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
	}
	return nil
}

func deleteEC2InstancesByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	ress, _ := describe.GetEC2InstanceReservations(ids)
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
		isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
		if isAWSError {
			aerrCode := herr.(awserr.Error).Code()
			if aerrCode == drCode {
				for _, id := range iIDs {
					fmt.Println(drStr, fmtStr, id)
				}
				return nil
			}
			if aerrCode == depvCode || aerrCode == invpCode {
				return nil
			}
		}
		return herr
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
				isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
				if isAWSError {
					aerrCode := herr.(awserr.Error).Code()
					if aerrCode == drCode {
						fmt.Println(fmtStr, *igw.InternetGatewayId, "from", *a.VpcId)
						continue
					}
					if aerrCode == drCode || aerrCode == depvCode || aerrCode == invpCode {
						continue
					}
				}
				return herr
			}
			fmt.Printf("%s %s from VPC %s\n", fmtStr, *igw.InternetGatewayId, *a.VpcId)
		}
	}
	return nil
}

// NOTE: must detach all internet gateways from vpc's before deletion
func deleteEC2InternetGatewaysByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	igws, ierr := describe.GetEC2InternetGateways(ids)
	if ierr != nil {
		return nil
	}

	// First detach internet gateways from all vpc's
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s (%s)\n", id, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
	}
	return nil
}

func deleteEC2NatGatewaysByIDs(cfg *DeleteConfig, ngwIDs *[]string) error {
	if ngwIDs == nil {
		return nil
	}

	svc := ec2.New(cfg.AWSSession)
	fmtStr := "Deleted EC2 NatGateway"
	if cfg.DryRun {
		for _, id := range *ngwIDs {
			fmt.Println(drStr, fmtStr, id)
		}
		return nil
	}
	var params *ec2.DeleteNatGatewayInput
	for _, id := range *ngwIDs {
		params = &ec2.DeleteNatGatewayInput{
			NatGatewayId: aws.String(id),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteNatGatewayWithContext(ctx, params)
		if err != nil {
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s (%s)\n", id, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
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
				isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
				if isAWSError {
					aerrCode := herr.(awserr.Error).Code()
					if aerrCode == invpCode {
						fmt.Printf("NetworkAcl %s Entry %d skipped (%s)\n", *acl.NetworkAclId, *a.RuleNumber, aerrCode)
						continue
					} else {
						fmt.Printf("Could not delete NetworkAcl %s Entry %d (%s)\n", *acl.NetworkAclId, *a.RuleNumber, aerrCode)
						continue
					}
				}
				return err
			}
			fmt.Printf("%s %s Entry %d\n", fmtStr, *acl.NetworkAclId, *a.RuleNumber)
		}
	}
	return nil
}

func deleteEC2NetworkACLsByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	acls, aerr := describe.GetEC2NetworkACLs(ids)
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerr := herr.(awserr.Error)
				aerrCode, aerrMsg := aerr.Code(), aerr.Message()
				if aerrCode == invpCode && strings.Contains(aerrMsg, "default") {
					fmt.Printf("Default Network ACL %s skipped\n", *acl.NetworkAclId)
					continue
				}
				if aerrCode == drCode {
					fmt.Println(fmtStr, *acl.NetworkAclId)
					continue
				}
				if aerrCode == depvCode {
					fmt.Printf("Could not delete %s (%s)\n", *acl.NetworkAclId, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, *acl.NetworkAclId)
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerr := herr.(awserr.Error)
				// aerrCode, aerrMsg := aerr.Code(), aerr.Message()
				aerrCode := aerr.Code()
				if aerrCode == nopCode {
					fmt.Printf("Could not detach NetworkInterface %s (%s)\n", id, aerrCode)
					continue
				}
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s (%s)\n", id, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
	}
	return nil
}

// NOTE: no ARN
func deleteEC2NetworkInterfacesByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	// To delete a network interface, all attachments must be deleted first.
	// DescribeNetworkInterfaces will find attachment ID's.
	enis, nierr := describe.GetEC2NetworkInterfaces(ids)
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerr := herr.(awserr.Error)
				aerrCode, aerrMsg := aerr.Code(), aerr.Message()
				// fmt.Println(aerrMsg, aerrCode)
				if strings.Contains(aerrMsg, "in use") {
					fmt.Printf("Network interface %s in use\n", id)
					continue
				}
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
				if aerrCode == invpCode || aerrCode == depvCode {
					fmt.Printf("Could not delete %s (%s)\n", id, aerrCode)
					continue
				}
				if aerrCode == "InvalidNetworkInterfaceID.NotFound" {
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
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
		if *r.GatewayId == "local" {
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerr := herr.(awserr.Error)
				aerrCode, aerrMsg := aerr.Code(), aerr.Message()
				if strings.Contains(aerrMsg, ".NotFound") {
					fmt.Printf("%s CIDR %s not found\n", *rtID, *r.DestinationCidrBlock)
					continue
				}
				if aerrCode == drCode {
					fmt.Printf("%s: Dst CIDR Block %s\n", fmtStr, *r.DestinationCidrBlock)
					continue
				}
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s CIDR %s (%s)\n", *rtID, *r.DestinationCidrBlock, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Printf("%s: Dst CIDR Block %s\n", fmtStr, *r.DestinationCidrBlock)
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerr := err.(awserr.Error)
				aerrCode := aerr.Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
				if aerrCode == depvCode || aerrCode == invpCode {
					if strings.Contains(aerr.Message(), "main route table") {
						fmt.Printf("Main RouteTableAssociation %s skipped\n", id)
						continue
					} else {
						fmt.Printf("Could not delete RouteTableAssociation %s\n", id)
						continue
					}
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
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
	rts, err := describe.GetEC2RouteTables(ids)
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

	// Now delete route table
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
		_, derr := svc.DeleteRouteTableWithContext(ctx, params)
		if err != nil {
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, derr)
			if isAWSError {
				aerrCode := err.(awserr.Error).Code()
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete RouteTable %s (%s)\n", *rt.RouteTableId, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, *rt.RouteTableId)
	}
	return nil
}

var defaultGroupName = "default"

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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerr := herr.(awserr.Error)
				aerrCode, aerrMsg := aerr.Code(), aerr.Message()
				if strings.Contains(aerrMsg, ".NotFound") {
					fmt.Printf("SecurityGroup Ingress Rule for %s not found\n", *sg.GroupId)
					continue
				}
				if aerrCode == drCode {
					fmt.Printf("%s for %s\n", fmtStr, *sg.GroupId)
					continue
				}
				if aerrCode == mprmCode {
					fmt.Printf("Ingress IpPermissions for %s missing\n", *sg.GroupId)
					continue
				}
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("SecurityGroup Ingress Rule for %s skipped (%s)\n", *sg.GroupId, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Printf("%s for %s\n", fmtStr, *sg.GroupId)
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerr := herr.(awserr.Error)
				aerrCode, aerrMsg := aerr.Code(), aerr.Message()
				if strings.Contains(aerrMsg, ".NotFound") {
					fmt.Printf("SecurityGroup Egress Rule for %s not found\n", *sg.GroupId)
					continue
				}
				if aerrCode == drCode {
					fmt.Printf("%s for %s\n", fmtStr, *sg.GroupId)
					continue
				}
				if aerrCode == mprmCode {
					fmt.Printf("Egress IpPermissions for %s missing\n", *sg.GroupId)
					continue
				}
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("SecurityGroup Egress Rule for %s skipped (%s)\n", *sg.GroupId, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Printf("%s for %s\n", fmtStr, *sg.GroupId)
	}
	return nil
}

// NOTE: all security group references must be removed before deleting before
// deleting a security group
// If autoscaling groups or launch configurations aren't tagged and require
// deletion, they must be deleted before security groups are deleted
func deleteEC2SecurityGroupsByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	sgs, rerr := describe.GetEC2SecurityGroups(ids)
	if rerr != nil {
		return rerr
	}
	if sgs == nil {
		return nil
	}

	// First delete ingress/egress rules (security group references)
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, *sg.GroupId)
					continue
				}
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("SecurityGroup %s skipped (%s)\n", *sg.GroupId, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, *sg.GroupId)
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s (%s)\n", id, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Subnet %s skipped (%s)\n", id, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s (%s)\n", id, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
	}
	return nil
}

func deleteEC2VPCCIDRBlocks(cfg *DeleteConfig, vpcIDs *[]string) error {
	if vpcIDs == nil {
		return nil
	}

	vpcs, verr := describe.GetEC2VPCs(vpcIDs)
	if verr != nil {
		return verr
	}
	if vpcs == nil {
		return nil
	}

	svc := ec2.New(cfg.AWSSession)
	if cfg.DryRun {
		for _, id := range *vpcIDs {
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
				isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
				if isAWSError {
					aerrCode := herr.(awserr.Error).Code()
					if aerrCode == depvCode || aerrCode == invpCode {
						continue
					}
				}
				return herr
			}
			fmt.Printf("%s Deleted EC2 VPC %s CIDRBlockAssociation %s\n", drStr, *vpc.VpcId, *b.AssociationId)
		}
	}
	return nil
}

func deleteEC2VPCsByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	// Ensure that all associated internet gateways are deleted
	// igws, ierr := describe.GetEC2InternetGatewaysByVPC(ids)
	// if ierr != nil {
	// 	return ierr
	// }
	// if dierr := deleteEC2InternetGatewaysFromIGWs(cfg, igws); dierr != nil {
	// 	return dierr
	// }

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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == drCode {
					fmt.Println(fmtStr, id)
					continue
				}
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s (%s)\n", id, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
	}
	return nil
}

// NOTE: this only supports deleting classic (non-application) ELB's
func deleteElasticLoadBalancersByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}

	svc := elb.New(cfg.AWSSession)
	fmtStr := "Deleted ELB"
	if cfg.DryRun {
		for _, id := range *ids {
			fmt.Println(drStr, fmtStr, id)
		}
		return nil
	}
	var params *elb.DeleteLoadBalancerInput
	for _, id := range *ids {
		params = &elb.DeleteLoadBalancerInput{
			LoadBalancerName: aws.String(id),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteLoadBalancerWithContext(ctx, params)
		if err != nil {
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s (%s)\n", id, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
	}
	// Wait for ELB's to delete
	time.Sleep(time.Duration(30) * time.Second)
	return nil
}

func removeIAMRolesFromInstanceProfilesByIPrs(cfg *DeleteConfig, iprs *[]*iam.InstanceProfile) error {
	if iprs == nil {
		return nil
	}

	svc := iam.New(cfg.AWSSession)
	if cfg.DryRun {
		for _, ipr := range *iprs {
			fmt.Println("Removed Role from IAM InstanceProfile", ipr.InstanceProfileId)
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
				isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
				if isAWSError {
					aerrCode := herr.(awserr.Error).Code()
					if aerrCode == depvCode || aerrCode == invpCode || aerrCode == dcfCode {
						fmt.Printf("Could not delete %s from InstanceProfile %s (%s)\n", *rl.RoleName, *ipr.InstanceProfileName, aerrCode)
						continue
					}
				}
				return herr
			}
			fmt.Printf("Removed Role %s from IAM InstanceProfile %s\n", *ipr.InstanceProfileName, *rl.RoleName)
		}
	}
	return nil
}

// NOTE: must delete roles from instance profile before deleting roles. Must
// be done in this step because of one-way knowledge relationship.
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == nseCode {
					fmt.Printf("Instance Profile %s cannot be found\n", *ipr.InstanceProfileName)
					continue
				}
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s (%s)\n", *ipr.InstanceProfileName, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, *ipr.InstanceProfileName)
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
				isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
				if isAWSError {
					aerrCode := herr.(awserr.Error).Code()
					if aerrCode == depvCode || aerrCode == invpCode || aerrCode == dcfCode {
						fmt.Printf("Could not delete %s RolePolicy %s (%s)\n", rn, rp, aerrCode)
						continue
					}
				}
				return herr
			}
			fmt.Println(fmtStr, rp)
		}
	}
	return nil
}

func deleteIAMRolesByNames(cfg *DeleteConfig, ns *[]string) error {
	if ns == nil {
		return nil
	}

	// First delete RolePolicies
	rps, lerr := describe.GetIAMRolePoliciesByRoles(ns)
	if lerr != nil {
		return lerr
	}
	if perr := deleteIAMRolePoliciesByRoles(cfg, rps); perr != nil {
		return perr
	}

	// Now delete roles
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == depvCode || aerrCode == invpCode || aerrCode == dcfCode {
					fmt.Printf("Could not delete %s (%s)\n", n, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, n)
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s (%s)\n", n, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, n)
	}
	return nil
}

func deleteS3Objects(cfg *DeleteConfig, bktIDs *[]string) error {
	if bktIDs == nil {
		return nil
	}

	fmtStr := "Deleted S3 Object"
	svc := s3.New(cfg.AWSSession)

	if cfg.DryRun {
		for _, bkt := range *bktIDs {
			fmt.Printf("%s %s from S3 Bucket %s\n", drStr, fmtStr, bkt)
		}
		return nil
	}

	var params *s3.DeleteObjectsInput
	for _, bkt := range *bktIDs {
		objs, oerr := describe.GetS3BucketObjects(bkt)
		if oerr != nil || objs == nil {
			continue
		}

		objIDs := make([]*s3.ObjectIdentifier, 0, len(*objs))
		for _, o := range *objs {
			objIDs = append(objIDs, &s3.ObjectIdentifier{Key: o.Key})
		}
		params = &s3.DeleteObjectsInput{
			Bucket: aws.String(bkt),
			Delete: &s3.Delete{Objects: objIDs},
		}
		ctx := aws.BackgroundContext()
		resp, err := svc.DeleteObjectsWithContext(ctx, params)
		if err != nil {
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s (%s)\n", bkt, aerrCode)
					continue
				}
			}
			return herr
		}
		for _, o := range resp.Deleted {
			fmt.Printf("%s %s from S3 Bucket %s\n", fmtStr, *o.Key, bkt)
		}
	}
	return nil
}

// NOTE: must delete objects in bucket first
func deleteS3BucketsByNames(cfg *DeleteConfig, ns *[]string) error {
	if ns == nil {
		return nil
	}
	// Delete all objects in buckets
	if berr := deleteS3Objects(cfg, ns); berr != nil {
		return berr
	}

	svc := s3.New(cfg.AWSSession)
	fmtStr := "Deleted S3 Bucket"
	if cfg.DryRun {
		for _, id := range *ns {
			fmt.Println(drStr, fmtStr, id)
		}
		return nil
	}
	var params *s3.DeleteBucketInput
	for _, id := range *ns {
		params = &s3.DeleteBucketInput{
			Bucket: aws.String(id),
		}

		ctx := aws.BackgroundContext()
		_, err := svc.DeleteBucketWithContext(ctx, params)
		if err != nil {
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s (%s)\n", id, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
	}
	return nil
}

func deleteRoute53ResourceRecordSets(cfg *DeleteConfig, hzResMap *map[string][]*route53.ResourceRecordSet) error {
	if hzResMap == nil {
		return nil
	}
	svc := route53.New(cfg.AWSSession)
	fmtStr := "Deleted Route53 ResourceRecordSet"
	var params *route53.ChangeResourceRecordSetsInput
	if cfg.DryRun {
		for hz, rrs := range *hzResMap {
			for _, rs := range rrs {
				fmt.Printf("%s %s %s (HZ %s)\n", drStr, fmtStr, *rs.Name, hz)
			}
		}
		return nil
	}
	for hz, rrs := range *hzResMap {
		changes := make([]*route53.Change, 0, len(rrs))
		for _, rs := range rrs {
			changes = append(changes, &route53.Change{
				Action:            aws.String(route53.ChangeActionDelete),
				ResourceRecordSet: rs,
			})
		}

		params = &route53.ChangeResourceRecordSetsInput{
			ChangeBatch:  &route53.ChangeBatch{Changes: changes},
			HostedZoneId: aws.String(hz),
		}

		ctx := aws.BackgroundContext()
		_, cerr := svc.ChangeResourceRecordSetsWithContext(ctx, params)
		if cerr != nil {
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, cerr)
			if isAWSError {
				aerr := herr.(awserr.Error)
				aerrCode, aerrMsg := aerr.Code(), aerr.Message()
				if strings.Contains(aerrMsg, ".NotFound") {
					for _, c := range changes {
						fmt.Printf("%s Record Set %s not found\n", hz, *c.ResourceRecordSet.Name)
					}
					continue
				}
				if aerrCode == depvCode || aerrCode == invpCode {
					for _, c := range changes {
						fmt.Printf("Could not delete %s Record Set %s (%s)\n", hz, *c.ResourceRecordSet.Name, aerrCode)
					}
					continue
				}
			}
			return herr
		}
	}
	return nil
}

// NOTE: must delete all non-default resource record sets before deleting a
// hosted zone. Will receive HostedZoneNotEmpty otherwise
func deleteRoute53HostedZonesByIDs(cfg *DeleteConfig, ids *[]string) error {
	if ids == nil {
		return nil
	}
	// First delete all resource record sets
	rrs, rerr := describe.GetRoute53ResourceRecordSets(ids)
	if rerr != nil {
		return rerr
	}
	if rrerr := deleteRoute53ResourceRecordSets(cfg, rrs); rrerr != nil {
		return rrerr
	}
	// Then delete hosted zones
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
			isAWSError, herr := handleAWSError(cfg.IgnoreErrors, err)
			if isAWSError {
				aerrCode := herr.(awserr.Error).Code()
				if aerrCode == depvCode || aerrCode == invpCode {
					fmt.Printf("Could not delete %s (%s)\n", id, aerrCode)
					continue
				}
			}
			return herr
		}
		fmt.Println(fmtStr, id)
	}
	return nil
}

// TraverseDependencyGraph traverses necesssary linkages of each resource
func TraverseDependencyGraph(depMap *map[string][]string) {
	if depMap == nil {
		return
	}
	for dt, ids := range *depMap {
		switch dt {
		case arn.EC2VPCRType:
			break
		case arn.EC2SubnetRType:
			// Get NAT Gateways
			ngws, serr := describe.GetEC2NatGatewaysBySubnetIDs(&ids)
			if serr != nil || ngws == nil {
				break
			}
			ngwt := arn.EC2NatGatewayRType
			for _, ngw := range *ngws {
				(*depMap)[ngwt] = append((*depMap)[ngwt], *ngw.NatGatewayId)
			}
			// Get Network ACL's
			nacls, nerr := describe.GetEC2NetworkACLsBySubnet(&ids)
			if nerr != nil || nacls == nil {
				break
			}
			naclt := arn.EC2NetworkACLRType
			for _, nacl := range *nacls {
				if !*nacl.IsDefault {
					(*depMap)[naclt] = append((*depMap)[naclt], *nacl.NetworkAclId)
				}
			}
			break
		case arn.EC2SecurityGroupRType:
			// Get SecurityGroup Rule
			// IP permissions Ingress/Egress will be deleted when deleting SecurityGroups
			break
		case arn.EC2InstanceRType:
			// Get EBS Volumes
			break
		case arn.EC2NetworkInterfaceRType:
			adrs, aerr := describe.GetEC2EIPAddressesByENIIDs(&ids)
			if aerr != nil || adrs == nil {
				break
			}
			// Get EC2EIPRType
			// Get EC2EIPAssociationRType
			eipt, eipat := arn.EC2EIPRType, arn.EC2EIPAssociationRType
			for _, adr := range *adrs {
				fmt.Println(*adr.AllocationId, *adr.AssociationId)
				if adr.AllocationId != nil {
					(*depMap)[eipt] = append((*depMap)[eipt], *adr.AllocationId)
				}
				if adr.AssociationId != nil {
					(*depMap)[eipat] = append((*depMap)[eipat], *adr.AssociationId)
				}
			}
			break
		case arn.ElasticLoadBalancingLoadBalancerRType:
			// Get ELB Policies
			break
		case arn.EC2RouteTableRType:
			// RouteTable Routes will be deleted when deleting a RouteTable
			rts, rerr := describe.GetEC2RouteTables(&ids)
			if rerr != nil || rts == nil {
				break
			}
			// Get Subnet-RouteTable Association
			rtat := arn.EC2RouteTableAssociationRType
			for _, rtb := range *rts {
				for _, a := range rtb.Associations {
					if !*a.Main {
						(*depMap)[rtat] = append((*depMap)[rtat], *a.RouteTableAssociationId)
					}
				}
			}
			break
		case arn.AutoScalingGroupRType:
			asgs, aerr := describe.GetAutoScalingGroups(&ids)
			if aerr != nil || asgs == nil {
				break
			}
			// Get AS LaunchConfigurations
			lcs := make([]string, 0, len(*asgs))
			lct := arn.AutoScalingLaunchConfigurationRType
			for _, asg := range *asgs {
				lcs = append(lcs, *asg.LaunchConfigurationName)
				(*depMap)[lct] = append((*depMap)[lct], *asg.LaunchConfigurationName)
			}
			// Get IAM instance profiles
			iprs, ierr := describe.GetInstanceProfilesByLaunchConfigs(&lcs)
			if ierr != nil || iprs == nil {
				break
			}
			iprt, rlt := arn.IAMInstanceProfileRType, arn.IAMRoleRType
			for _, ipr := range *iprs {
				(*depMap)[iprt] = append((*depMap)[iprt], *ipr.InstanceProfileId)
				// Get IAM roles
				for _, rl := range ipr.Roles {
					(*depMap)[rlt] = append((*depMap)[rlt], *rl.RoleName)
				}
			}
			// IAM RolePolicies will be deleted when deleting Roles
			break
		case arn.Route53HostedZoneRType:
			// Route53 RecordSets will be deleted when deleting HostedZones
			break
		case arn.S3BucketRType:
			// S3 Objects will be deleted when deleting a Bucket
			break
		}
	}
	return
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
		return deleteEC2AmisByIDs(cfg, ids)
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
		return deleteElasticLoadBalancersByIDs(cfg, ids)
	case arn.IAMInstanceProfileRType:
		return deleteIAMInstanceProfilesByIDs(cfg, ids)
	case arn.IAMRoleRType:
		return deleteIAMRolesByNames(cfg, ids)
	case arn.IAMUserRType:
		return deleteIAMUsersByNames(cfg, ids)
	case arn.S3BucketRType:
		return deleteS3BucketsByNames(cfg, ids)
		// case arn.Route53ChangeRType:
		// 	return deleteRoute53ChangesByIDs(cfg, ids)
	case arn.Route53HostedZoneRType:
		return deleteRoute53HostedZonesByIDs(cfg, ids)
	}
	fmt.Printf("%s is not a supported ResourceType\n", cfg.ResourceType)
	return nil
}

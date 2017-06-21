package deleter

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/coreos/grafiti/arn"
)

// EC2CustomerGatewayDeleter represents a collection of AWS EC2 customer gateways
type EC2CustomerGatewayDeleter struct {
	Client        ec2iface.EC2API
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2CustomerGatewayDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// AddResourceNames adds EC2 customer gateway names to ResourceNames
func (rd *EC2CustomerGatewayDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes customer gateways from AWS
func (rd *EC2CustomerGatewayDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	fmtStr := "Deleted EC2 CustomerGateway"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	var params *ec2.DeleteCustomerGatewayInput
	for _, n := range rd.ResourceNames {
		params = &ec2.DeleteCustomerGatewayInput{
			CustomerGatewayId: n.AWSString(),
			DryRun:            aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.Client.DeleteCustomerGatewayWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2CustomerGatewayRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, n)
	}

	return nil
}

// EC2ElasticIPAllocationDeleter represents a collection of AWS EC2 elastic IP allocations
type EC2ElasticIPAllocationDeleter struct {
	Client        ec2iface.EC2API
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2ElasticIPAllocationDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "ResourceNames": %v}`, rd.ResourceType, rd.ResourceNames)
}

// AddResourceNames adds EC2 elastic IP allocation names to ResourceNames
func (rd *EC2ElasticIPAllocationDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes elastic IP allocations from AWS
func (rd *EC2ElasticIPAllocationDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	fmtStr := "Released EC2 ElasticIPAllocation"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	var params *ec2.ReleaseAddressInput
	for _, n := range rd.ResourceNames {
		params = &ec2.ReleaseAddressInput{
			AllocationId: n.AWSString(),
			DryRun:       aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.Client.ReleaseAddressWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2EIPRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, n)
	}

	return nil
}

// EC2ElasticIPAssocationDeleter represents a collection of AWS EC2 elastic IP associations
type EC2ElasticIPAssocationDeleter struct {
	Client        ec2iface.EC2API
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2ElasticIPAssocationDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "ResourceNames": %v}`, rd.ResourceType, rd.ResourceNames)
}

// AddResourceNames adds EC2 elastic IP association names to ResourceNames
func (rd *EC2ElasticIPAssocationDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes elastic IP associations from AWS
func (rd *EC2ElasticIPAssocationDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	fmtStr := "Disassociated EC2 ElasticIP"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	var params *ec2.DisassociateAddressInput
	for _, n := range rd.ResourceNames {
		params = &ec2.DisassociateAddressInput{
			AssociationId: n.AWSString(),
			DryRun:        aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.Client.DisassociateAddressWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2EIPAssociationRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, n)
	}

	return nil
}

// EC2NetworkInterfaceDeleter represents a collection of AWS EC2 network interfaces
type EC2NetworkInterfaceDeleter struct {
	Client        ec2iface.EC2API
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2NetworkInterfaceDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "ResourceNames": %v}`, rd.ResourceType, rd.ResourceNames)
}

// AddResourceNames adds EC2 network interface names to ResourceNames
func (rd *EC2NetworkInterfaceDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes EC2 network interfaces from AWS
func (rd *EC2NetworkInterfaceDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	// To delete a network interface, all attachments must be deleted first
	enis, nerr := rd.RequestEC2NetworkInterfaces()
	if nerr != nil && !cfg.IgnoreErrors {
		return nerr
	}

	eniaNames := make(arn.ResourceNames, 0)
	for _, eni := range enis {
		if eni.Attachment != nil && eni.Attachment.AttachmentId != nil {
			// eth0 is the primary network interface and cannot be detached
			if eni.Attachment.DeviceIndex != nil && *eni.Attachment.DeviceIndex == 0 {
				continue
			}
			eniaNames = append(eniaNames, arn.ResourceName(*eni.Attachment.AttachmentId))
		}
	}

	// Detach network interfaces
	eniaDel := &EC2NetworkInterfaceAttachmentDeleter{ResourceNames: eniaNames}
	if err := eniaDel.DeleteResources(cfg); err != nil && !cfg.IgnoreErrors {
		return err
	}

	fmtStr := "Deleted EC2 NetworkInterface"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	var params *ec2.DeleteNetworkInterfaceInput
	for _, n := range rd.ResourceNames {
		params = &ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: n.AWSString(),
			DryRun:             aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.Client.DeleteNetworkInterfaceWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2NetworkInterfaceRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, n)
	}

	return nil
}

// RequestEC2NetworkInterfaces requests EC2 network interfaces by name from the
// AWS API
func (rd *EC2NetworkInterfaceDeleter) RequestEC2NetworkInterfaces() ([]*ec2.NetworkInterface, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	// If resource names are passed into the 'NetworkInterfaceId' field and an interface
	// with one of those names does not exist, DescribeNetworkInterfaces will error.
	// Using filters avoids this issue.
	params := &ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("network-interface-id"), Values: rd.ResourceNames.AWSStringSlice()},
		},
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.Client.DescribeNetworkInterfacesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return resp.NetworkInterfaces, nil
}

// RequestEC2EIPAddressessFromNetworkInterfaces requests EC2 elastic IP addresses by
// network interface names from the AWS API
func (rd *EC2NetworkInterfaceDeleter) RequestEC2EIPAddressessFromNetworkInterfaces() ([]*ec2.Address, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeAddressesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("network-interface-id"), Values: rd.ResourceNames.AWSStringSlice()},
		},
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.Client.DescribeAddressesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return resp.Addresses, nil
}

// EC2NetworkInterfaceAttachmentDeleter represents a collection of AWS EC2 network interface attachments
type EC2NetworkInterfaceAttachmentDeleter struct {
	Client        ec2iface.EC2API
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2NetworkInterfaceAttachmentDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "ResourceNames": %v}`, rd.ResourceType, rd.ResourceNames)
}

// AddResourceNames adds EC2 network interface attachment names to ResourceNames
func (rd *EC2NetworkInterfaceAttachmentDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes EC2 network interface attachments from AWS
func (rd *EC2NetworkInterfaceAttachmentDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	fmtStr := "Detached EC2 NetworkInterface"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	var params *ec2.DetachNetworkInterfaceInput
	for _, n := range rd.ResourceNames {
		params = &ec2.DetachNetworkInterfaceInput{
			AttachmentId: n.AWSString(),
			Force:        aws.Bool(true),
			DryRun:       aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.Client.DetachNetworkInterfaceWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2NetworkInterfaceAttachmentRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, n)
	}

	return nil
}

// EC2InstanceDeleter represents a collection of AWS EC2 instances
type EC2InstanceDeleter struct {
	Client        ec2iface.EC2API
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2InstanceDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "ResourceNames": %v}`, rd.ResourceType, rd.ResourceNames)
}

// AddResourceNames adds EC2 instance names to ResourceNames
func (rd *EC2InstanceDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes EC2 instances from AWS
func (rd *EC2InstanceDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	ress, rerr := rd.RequestEC2InstanceReservations()
	if rerr != nil && !cfg.IgnoreErrors {
		return rerr
	}

	iNames := make(arn.ResourceNames, 0)
	for _, res := range ress {
		for _, ins := range res.Instances {
			code := *ins.State.Code
			// If instance is shutting down (32) or terminated (48), skip
			if code == 32 || code == 48 {
				continue
			}
			iNames = append(iNames, arn.ResourceName(*ins.InstanceId))
		}
	}

	if len(iNames) == 0 {
		return nil
	}

	fmtStr := "Terminated EC2 Instance"

	params := &ec2.TerminateInstancesInput{
		InstanceIds: iNames.AWSStringSlice(),
		DryRun:      aws.Bool(cfg.DryRun),
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.Client.TerminateInstancesWithContext(ctx, params)
	if err != nil {
		for _, n := range iNames {
			cfg.logDeleteError(arn.EC2InstanceRType, n, err)
		}
		if cfg.IgnoreErrors {
			return nil
		}
		return err
	}

	for _, id := range iNames {
		fmt.Println(fmtStr, id)
	}

	// Instances take awhile to shut down, so block until they've terminated
	if len(resp.TerminatingInstances) > 0 {
		termInstances := make([]*string, 0, len(resp.TerminatingInstances))
		for _, r := range resp.TerminatingInstances {
			termInstances = append(termInstances, r.InstanceId)
		}
		rd.waitUntilTerminated(cfg, termInstances)
	}

	return nil
}

func (rd *EC2InstanceDeleter) waitUntilTerminated(cfg *DeleteConfig, tis []*string) {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	params := &ec2.DescribeInstancesInput{
		InstanceIds: tis,
	}

	ctx := aws.BackgroundContext()
	if err := rd.Client.WaitUntilInstanceTerminatedWithContext(ctx, params); err != nil {
		for _, ti := range tis {
			cfg.logDeleteError(arn.EC2InstanceRType, arn.ResourceName(*ti), err)
		}
	}
}

// RequestEC2InstanceReservations retrieves EC2 instances by instance names from the AWS API
func (rd *EC2InstanceDeleter) RequestEC2InstanceReservations() ([]*ec2.Reservation, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeInstancesInput{
		InstanceIds: rd.ResourceNames.AWSStringSlice(),
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.Client.DescribeInstancesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return resp.Reservations, nil
}

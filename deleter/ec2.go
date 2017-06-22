package deleter

import (
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
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

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2CustomerGatewayDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
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

	cgws, rerr := rd.RequestEC2CustomerGateways()
	if rerr != nil && !cfg.IgnoreErrors {
		return rerr
	}

	fmtStr := "Deleted EC2 CustomerGateway"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	var params *ec2.DeleteCustomerGatewayInput
	for _, cgw := range cgws {
		params = &ec2.DeleteCustomerGatewayInput{
			CustomerGatewayId: cgw.CustomerGatewayId,
			DryRun:            aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteCustomerGatewayWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2CustomerGatewayRType, arn.ResourceName(*cgw.CustomerGatewayId), err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, *cgw.CustomerGatewayId)
	}

	return nil
}

// RequestEC2CustomerGateways requests EC2 customer gateways by names from the AWS API
func (rd *EC2CustomerGatewayDeleter) RequestEC2CustomerGateways() ([]*ec2.CustomerGateway, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeCustomerGatewaysInput{
		CustomerGatewayIds: rd.ResourceNames.AWSStringSlice(),
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeCustomerGatewaysWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return resp.CustomerGateways, nil
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

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2ElasticIPAllocationDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
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

	var params *ec2.ReleaseAddressInput
	for _, n := range rd.ResourceNames {
		params = &ec2.ReleaseAddressInput{
			AllocationId: n.AWSString(),
			DryRun:       aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().ReleaseAddressWithContext(ctx, params)
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

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2ElasticIPAssocationDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
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

	var params *ec2.DisassociateAddressInput
	for _, n := range rd.ResourceNames {
		params = &ec2.DisassociateAddressInput{
			AssociationId: n.AWSString(),
			DryRun:        aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DisassociateAddressWithContext(ctx, params)
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

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2NetworkInterfaceDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
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
	if err := eniaDel.DeleteResources(cfg); err != nil {
		return err
	}

	fmtStr := "Deleted EC2 NetworkInterface"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	var params *ec2.DeleteNetworkInterfaceInput
	for _, eni := range enis {
		params = &ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: eni.NetworkInterfaceId,
			DryRun:             aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteNetworkInterfaceWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2NetworkInterfaceRType, arn.ResourceName(*eni.NetworkInterfaceId), err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, *eni.NetworkInterfaceId)
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
	params := &ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("network-interface-id"), Values: rd.ResourceNames.AWSStringSlice()},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeNetworkInterfacesWithContext(ctx, params)
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

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeAddressesWithContext(ctx, params)
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

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2NetworkInterfaceAttachmentDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
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
		_, err := rd.GetClient().DetachNetworkInterfaceWithContext(ctx, params)
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

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2InstanceDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
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

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().TerminateInstancesWithContext(ctx, params)
	if err != nil {
		for _, n := range iNames {
			cfg.logDeleteError(arn.EC2InstanceRType, n, err)
		}
		if cfg.IgnoreErrors {
			return nil
		}
		return err
	}

	// Instances take awhile to shut down, so block until they've terminated
	if len(resp.TerminatingInstances) > 0 {
		fmt.Println("Waiting for EC2 Instances to terminate...")
		termInstances := make([]*string, 0, len(resp.TerminatingInstances))
		for _, r := range resp.TerminatingInstances {
			termInstances = append(termInstances, r.InstanceId)
		}
		rd.waitUntilTerminated(cfg, termInstances)
	}

	for _, id := range iNames {
		fmt.Println(fmtStr, id)
	}

	return nil
}

func (rd *EC2InstanceDeleter) waitUntilTerminated(cfg *DeleteConfig, tis []*string) {
	params := &ec2.DescribeInstancesInput{
		InstanceIds: tis,
	}

	ctx := aws.BackgroundContext()
	if err := rd.GetClient().WaitUntilInstanceTerminatedWithContext(ctx, params); err != nil {
		for _, ti := range tis {
			cfg.logDeleteError(arn.EC2InstanceRType, arn.ResourceName(*ti), err)
		}
	}
}

// RequestEC2InstanceReservations requests EC2 instances by instance names from the AWS API
func (rd *EC2InstanceDeleter) RequestEC2InstanceReservations() ([]*ec2.Reservation, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeInstancesInput{
		InstanceIds: rd.ResourceNames.AWSStringSlice(),
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeInstancesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return resp.Reservations, nil
}

// EC2InternetGatewayAttachmentDeleter represents a collection of AWS EC2 internet gateway attachments
type EC2InternetGatewayAttachmentDeleter struct {
	Client              ec2iface.EC2API
	ResourceType        arn.ResourceType
	InternetGatewayName arn.ResourceName
	AttachmentNames     arn.ResourceNames
}

func (rd *EC2InternetGatewayAttachmentDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "InternetGatewayName": "%s", "AttachmentNames": %v}`, rd.ResourceType, rd.InternetGatewayName, rd.AttachmentNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2InternetGatewayAttachmentDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
}

// AddResourceNames adds EC2 internet gateway attachment names to AttachmentNames
func (rd *EC2InternetGatewayAttachmentDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.AttachmentNames = append(rd.AttachmentNames, ns...)
}

// DeleteResources deletes EC2 internet gateway attachments from AWS
func (rd *EC2InternetGatewayAttachmentDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.AttachmentNames) == 0 || rd.InternetGatewayName == "" {
		return nil
	}

	fmtStr := "Detached EC2 InternetGateway"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	var params *ec2.DetachInternetGatewayInput
	for _, an := range rd.AttachmentNames {
		params = &ec2.DetachInternetGatewayInput{
			InternetGatewayId: rd.InternetGatewayName.AWSString(),
			DryRun:            aws.Bool(cfg.DryRun),
			VpcId:             an.AWSString(),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DetachInternetGatewayWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2InternetGatewayAttachmentRType, an, err, logrus.Fields{
				"parent_resource_type": arn.EC2InternetGatewayRType,
				"parent_resource_name": rd.InternetGatewayName,
			})
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Printf("%s %s from VPC %s\n", fmtStr, rd.InternetGatewayName, an)
	}

	return nil
}

// EC2InternetGatewayDeleter represents a collection of AWS EC2 internet gateways
type EC2InternetGatewayDeleter struct {
	Client        ec2iface.EC2API
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2InternetGatewayDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2InternetGatewayDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
}

// AddResourceNames adds EC2 internet gateway names to ResourceNames
func (rd *EC2InternetGatewayDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes EC2 internet gateways from AWS
// NOTE: must detach all internet gateways from vpc's before deletion
func (rd *EC2InternetGatewayDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	igws, ierr := rd.RequestEC2InternetGateways()
	if ierr != nil && !cfg.IgnoreErrors {
		return ierr
	}

	// Detach internet gateways from all vpc's
	for _, igw := range igws {
		igwaDel := &EC2InternetGatewayAttachmentDeleter{
			InternetGatewayName: arn.ResourceName(*igw.InternetGatewayId),
		}
		for _, an := range igw.Attachments {
			igwaDel.AddResourceNames(arn.ResourceName(*an.VpcId))
		}
		if err := igwaDel.DeleteResources(cfg); err != nil {
			return err
		}
	}

	fmtStr := "Deleted EC2 InternetGateway"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	var params *ec2.DeleteInternetGatewayInput
	for _, igw := range igws {
		params = &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: igw.InternetGatewayId,
			DryRun:            aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteInternetGatewayWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2InternetGatewayRType, arn.ResourceName(*igw.InternetGatewayId), err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, *igw.InternetGatewayId)
	}

	return nil
}

// RequestEC2InternetGateways requests EC2 internet gateways by name from the AWS API
func (rd *EC2InternetGatewayDeleter) RequestEC2InternetGateways() ([]*ec2.InternetGateway, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeInternetGatewaysInput{
		InternetGatewayIds: rd.ResourceNames.AWSStringSlice(),
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeInternetGatewaysWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return resp.InternetGateways, nil
}

// EC2NatGatewayDeleter represents a collection of AWS EC2 NAT gateways
type EC2NatGatewayDeleter struct {
	Client        ec2iface.EC2API
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2NatGatewayDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2NatGatewayDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
}

// AddResourceNames adds EC2 NAT gateway names to ResourceNames
func (rd *EC2NatGatewayDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes EC2 NAT gateways from AWS
func (rd *EC2NatGatewayDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	ngws, rerr := rd.RequestEC2NatGateways()
	if rerr != nil && !cfg.IgnoreErrors {
		return rerr
	}

	fmtStr := "Deleted EC2 NatGateway"
	if cfg.DryRun {
		for _, n := range rd.ResourceNames {
			fmt.Println(drStr, fmtStr, n)
		}
		return nil
	}

	var params *ec2.DeleteNatGatewayInput
	for _, ngw := range ngws {
		params = &ec2.DeleteNatGatewayInput{
			NatGatewayId: ngw.NatGatewayId,
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteNatGatewayWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2NatGatewayRType, arn.ResourceName(*ngw.NatGatewayId), err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, *ngw.NatGatewayId)
	}

	// Wait for NAT Gateways to delete
	time.Sleep(time.Duration(1) * time.Minute)
	return nil
}

// RequestEC2NatGateways requests EC2 nat gateways by name from the AWS API
func (rd *EC2NatGatewayDeleter) RequestEC2NatGateways() ([]*ec2.NatGateway, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeNatGatewaysInput{
		NatGatewayIds: rd.ResourceNames.AWSStringSlice(),
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeNatGatewaysWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return resp.NatGateways, nil
}

// EC2RouteTableRouteDeleter represents a collection of AWS EC2 route table routes
type EC2RouteTableRouteDeleter struct {
	Client       ec2iface.EC2API
	ResourceType arn.ResourceType
	RouteTable   *ec2.RouteTable
}

func (rd *EC2RouteTableRouteDeleter) String() string {
	routes := make([]string, 0, len(rd.RouteTable.Routes))
	for _, r := range rd.RouteTable.Routes {
		routes = append(routes, *r.DestinationCidrBlock)
	}
	return fmt.Sprintf(`{"Type": "%s", "RouteTable": "%s", "Routes": %v}`, rd.ResourceType, *rd.RouteTable.RouteTableId, routes)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2RouteTableRouteDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
}

const localGatewayID = "local"

// DeleteResources deletes EC2 route table routes from AWS
func (rd *EC2RouteTableRouteDeleter) DeleteResources(cfg *DeleteConfig) error {
	if rd.RouteTable == nil {
		return nil
	}

	fmtStr := "Deleted RouteTable Route"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	var params *ec2.DeleteRouteInput
	for _, r := range rd.RouteTable.Routes {
		if r.GatewayId != nil && *r.GatewayId == localGatewayID {
			continue
		}

		params = &ec2.DeleteRouteInput{
			DestinationCidrBlock: r.DestinationCidrBlock,
			RouteTableId:         rd.RouteTable.RouteTableId,
			DryRun:               aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteRouteWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2RouteTableRouteRType, arn.ResourceName(*r.DestinationCidrBlock), err, logrus.Fields{
				"parent_resource_type": arn.EC2RouteTableRType,
				"parent_resource_name": *rd.RouteTable.RouteTableId,
			})
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Printf("%s: Dst CIDR Block %s\n", fmtStr, *r.DestinationCidrBlock)
	}

	return nil
}

// EC2RouteTableAssociationDeleter represents a collection of AWS EC2 route table associations
type EC2RouteTableAssociationDeleter struct {
	Client        ec2iface.EC2API
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2RouteTableAssociationDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2RouteTableAssociationDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
}

// AddResourceNames adds EC2 route table association names to ResourceNames
func (rd *EC2RouteTableAssociationDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes EC2 route table associations from AWS
func (rd *EC2RouteTableAssociationDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	fmtStr := "Deleted RouteTable Association"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	var params *ec2.DisassociateRouteTableInput
	for _, n := range rd.ResourceNames {
		params = &ec2.DisassociateRouteTableInput{
			AssociationId: n.AWSString(),
			DryRun:        aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DisassociateRouteTableWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2RouteTableAssociationRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, n)
	}

	return nil
}

// EC2RouteTableDeleter represents a collection of AWS EC2 route tables
type EC2RouteTableDeleter struct {
	Client        ec2iface.EC2API
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2RouteTableDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2RouteTableDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
}

// AddResourceNames adds EC2 route table names to ResourceNames
func (rd *EC2RouteTableDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes EC2 route tables from AWS
// NOTE: can only delete a route table once all subnets have been disassociated,
// and and all routes have been deleted. Cannot delete the main (default) route
// table
func (rd *EC2RouteTableDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	// Ensure all routes are deleted
	rts, rerr := rd.RequestEC2RouteTables()
	if rerr != nil && !cfg.IgnoreErrors {
		return rerr
	}

	var rtrDel *EC2RouteTableRouteDeleter
	for _, rt := range rts {
		rtrDel = &EC2RouteTableRouteDeleter{RouteTable: rt}
		if err := rtrDel.DeleteResources(cfg); err != nil {
			return err
		}
	}

	// Delete route table
	fmtStr := "Deleted EC2 RouteTable"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	var params *ec2.DeleteRouteTableInput
	for _, rt := range rts {
		params = &ec2.DeleteRouteTableInput{
			RouteTableId: rt.RouteTableId,
			DryRun:       aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteRouteTableWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2RouteTableRType, arn.ResourceName(*rt.RouteTableId), err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, *rt.RouteTableId)
	}

	return nil
}

// RequestEC2RouteTables requests route tables by name from the AWS API
func (rd *EC2RouteTableDeleter) RequestEC2RouteTables() ([]*ec2.RouteTable, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeRouteTablesInput{
		RouteTableIds: rd.ResourceNames.AWSStringSlice(),
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeRouteTablesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	rts := make([]*ec2.RouteTable, 0)
	for _, rt := range resp.RouteTables {
		for _, a := range rt.Associations {
			if a.Main != nil && !*a.Main {
				rts = append(rts, rt)
				break
			}
		}
	}

	return rts, nil
}

// EC2SecurityGroupIngressRuleDeleter represents a collection of AWS EC2 security group ingress rules
type EC2SecurityGroupIngressRuleDeleter struct {
	Client         ec2iface.EC2API
	ResourceType   arn.ResourceType
	SecurityGroups []*ec2.SecurityGroup
}

func (rd *EC2SecurityGroupIngressRuleDeleter) String() string {
	rules := make([]string, 0)
	for _, sg := range rd.SecurityGroups {
		rules = append(rules, fmt.Sprintf(`{"SecurityGroupName": "%s", "IpPermissions": %v}`, *sg.GroupName, sg.IpPermissions))
	}
	return fmt.Sprintf(`{"Type": "%s", "IngressRules": %v}`, rd.ResourceType, rules)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2SecurityGroupIngressRuleDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
}

// DeleteResources deletes EC2 security group ingress rules from AWS
// NOTE: all security group ingress references must be removed before deleting before
// deleting a security group ingress
func (rd *EC2SecurityGroupIngressRuleDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.SecurityGroups) == 0 {
		return nil
	}

	fmtStr := "Deleted EC2 SecurityGroup Ingress Rule"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	var params *ec2.RevokeSecurityGroupIngressInput
	for _, sg := range rd.SecurityGroups {
		if len(sg.IpPermissions) == 0 {
			continue
		}

		params = &ec2.RevokeSecurityGroupIngressInput{
			GroupId:       sg.GroupId,
			IpPermissions: sg.IpPermissions,
			DryRun:        aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().RevokeSecurityGroupIngressWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2SecurityGroupIngressRType, arn.ResourceName(*sg.GroupId), err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Printf("%s for %s\n", fmtStr, *sg.GroupId)
	}
	return nil
}

// EC2SecurityGroupEgressRuleDeleter represents a collection of AWS EC2 security group egress rules
type EC2SecurityGroupEgressRuleDeleter struct {
	Client         ec2iface.EC2API
	ResourceType   arn.ResourceType
	SecurityGroups []*ec2.SecurityGroup
}

func (rd *EC2SecurityGroupEgressRuleDeleter) String() string {
	rules := make([]string, 0)
	for _, sg := range rd.SecurityGroups {
		rules = append(rules, fmt.Sprintf(`{"SecurityGroupName": "%s", "IpPermissions": %v}`, *sg.GroupName, sg.IpPermissionsEgress))
	}
	return fmt.Sprintf(`{"Type": "%s", "EgressRules": %v}`, rd.ResourceType, rules)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2SecurityGroupEgressRuleDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
}

// DeleteResources deletes EC2 security group egress rules from AWS
func (rd *EC2SecurityGroupEgressRuleDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.SecurityGroups) == 0 {
		return nil
	}

	fmtStr := "Deleted EC2 SecurityGroup Egress Rule"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	var params *ec2.RevokeSecurityGroupEgressInput
	for _, sg := range rd.SecurityGroups {
		if len(sg.IpPermissionsEgress) == 0 {
			continue
		}

		params = &ec2.RevokeSecurityGroupEgressInput{
			GroupId:       sg.GroupId,
			IpPermissions: sg.IpPermissionsEgress,
			DryRun:        aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().RevokeSecurityGroupEgressWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2SecurityGroupEgressRType, arn.ResourceName(*sg.GroupId), err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Printf("%s for %s\n", fmtStr, *sg.GroupId)
	}

	return nil
}

// EC2SecurityGroupDeleter represents a collection of AWS EC2 security groups
type EC2SecurityGroupDeleter struct {
	Client        ec2iface.EC2API
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2SecurityGroupDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2SecurityGroupDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
}

// AddResourceNames adds EC2 security group names to ResourceNames
func (rd *EC2SecurityGroupDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes EC2 security groups from AWS
// NOTE: all security group references must be removed before deleting before
// deleting a security group
func (rd *EC2SecurityGroupDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	sgs, rerr := rd.RequestEC2SecurityGroups()
	if rerr != nil && !cfg.IgnoreErrors {
		return rerr
	}
	if len(sgs) == 0 {
		return nil
	}

	// Delete ingress/egress rules (security group references)
	sgIngressDel := &EC2SecurityGroupIngressRuleDeleter{SecurityGroups: sgs}
	if err := sgIngressDel.DeleteResources(cfg); err != nil {
		return err
	}
	sgEgressDel := &EC2SecurityGroupIngressRuleDeleter{SecurityGroups: sgs}
	if err := sgEgressDel.DeleteResources(cfg); err != nil {
		return err
	}

	fmtStr := "Deleted EC2 SecurityGroup"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	var params *ec2.DeleteSecurityGroupInput
	for _, sg := range sgs {
		params = &ec2.DeleteSecurityGroupInput{
			GroupId: sg.GroupId,
			DryRun:  aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteSecurityGroupWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2SecurityGroupRType, arn.ResourceName(*sg.GroupId), err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, *sg.GroupId)
	}

	return nil
}

const defaultGroupName = "default"

// RequestEC2SecurityGroups requests EC2 security groups by name from the AWS API
func (rd *EC2SecurityGroupDeleter) RequestEC2SecurityGroups() ([]*ec2.SecurityGroup, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeSecurityGroupsInput{
		GroupIds: rd.ResourceNames.AWSStringSlice(),
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeSecurityGroupsWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	// Default security groups cannot be deleted, so remove them from response elements
	sgs := make([]*ec2.SecurityGroup, 0)
	for _, sg := range resp.SecurityGroups {
		if *sg.GroupName != defaultGroupName {
			sgs = append(sgs, sg)
		}
	}

	return sgs, nil
}

// EC2SubnetDeleter represents a collection of AWS EC2 subnets
type EC2SubnetDeleter struct {
	Client        ec2iface.EC2API
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2SubnetDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2SubnetDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
}

// AddResourceNames adds EC2 subnet names to ResourceNames
func (rd *EC2SubnetDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes EC2 subnets from AWS
// NOTE: ensure all network interfaces and network acl's are disassociated
func (rd *EC2SubnetDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	fmtStr := "Deleted EC2 Subnet"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	var params *ec2.DeleteSubnetInput
	for _, n := range rd.ResourceNames {
		params = &ec2.DeleteSubnetInput{
			SubnetId: n.AWSString(),
			DryRun:   aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteSubnetWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2SubnetRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, n)
	}

	return nil
}

// RequestEC2Subnets requests EC2 subnets by subnet name from the AWS API
func (rd *EC2SubnetDeleter) RequestEC2Subnets() ([]*ec2.Subnet, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeSubnetsInput{
		SubnetIds: rd.ResourceNames.AWSStringSlice(),
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeSubnetsWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	sns := make([]*ec2.Subnet, 0)
	for _, sn := range resp.Subnets {
		if sn.DefaultForAz != nil && !*sn.DefaultForAz {
			sns = append(sns, sn)
		}
	}

	return sns, nil
}

// EC2VPCCIDRBlockAssociationDeleter represents a collection of AWS EC2 VPC CIDR block associations
type EC2VPCCIDRBlockAssociationDeleter struct {
	Client              ec2iface.EC2API
	ResourceType        arn.ResourceType
	VPCName             arn.ResourceName
	VPCAssociationNames arn.ResourceNames
}

func (rd *EC2VPCCIDRBlockAssociationDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.VPCAssociationNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2VPCCIDRBlockAssociationDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
}

// AddResourceNames adds EC2 VPC CIDR block association names to VPCAssociationNames
func (rd *EC2VPCCIDRBlockAssociationDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.VPCAssociationNames = append(rd.VPCAssociationNames, ns...)
}

// DeleteResources deletes EC2 VPC CIDR block associations from AWS
func (rd *EC2VPCCIDRBlockAssociationDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.VPCAssociationNames) == 0 {
		return nil
	}

	if cfg.DryRun {
		for _, n := range rd.VPCAssociationNames {
			fmt.Printf("%s Deleted EC2 VPC %s CIDRBlockAssociation\n", drStr, n)
		}
		return nil
	}

	var params *ec2.DisassociateVpcCidrBlockInput
	for _, n := range rd.VPCAssociationNames {
		params = &ec2.DisassociateVpcCidrBlockInput{
			AssociationId: n.AWSString(),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DisassociateVpcCidrBlockWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2VPCCIDRAssociationRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Printf("%s Deleted EC2 VPC %s CIDRBlockAssociation %s\n", drStr, rd.VPCName, n)
	}

	return nil
}

// EC2VPCDeleter represents a collection of AWS EC2 VPC's
type EC2VPCDeleter struct {
	Client        ec2iface.EC2API
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2VPCDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2VPCDeleter) GetClient() ec2iface.EC2API {
	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}
	return rd.Client
}

// AddResourceNames adds EC2 VPC names to ResourceNames
func (rd *EC2VPCDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes EC2 VPC's from AWS
func (rd *EC2VPCDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	// Disassociate vpc cidr blocks
	vpcs, verr := rd.RequestEC2VPCs()
	if verr != nil && !cfg.IgnoreErrors {
		return verr
	}

	for _, vpc := range vpcs {
		vpcaDel := &EC2VPCCIDRBlockAssociationDeleter{VPCName: arn.ResourceName(*vpc.VpcId)}
		for _, a := range vpc.Ipv6CidrBlockAssociationSet {
			vpcaDel.AddResourceNames(arn.ResourceName(*a.AssociationId))
		}
		if err := vpcaDel.DeleteResources(cfg); err != nil {
			return err
		}
	}

	// Now delete VPC itself
	fmtStr := "Deleted EC2 VPC"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	var params *ec2.DeleteVpcInput
	for _, n := range rd.ResourceNames {
		params = &ec2.DeleteVpcInput{
			VpcId:  n.AWSString(),
			DryRun: aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteVpcWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2VPCRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, n)
	}

	return nil
}

// RequestEC2VPCs requests EC2 vpc's by vpc names from the AWS API
func (rd *EC2VPCDeleter) RequestEC2VPCs() ([]*ec2.Vpc, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeVpcsInput{
		VpcIds: rd.ResourceNames.AWSStringSlice(),
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeVpcsWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	vpcs := make([]*ec2.Vpc, 0)
	for _, vpc := range resp.Vpcs {
		if vpc.IsDefault != nil && !*vpc.IsDefault {
			vpcs = append(vpcs, vpc)
		}
	}

	return vpcs, nil
}

// RequestEC2InstanceReservationsFromVPCs requests EC2 instance reservations from vpc names from the AWS API
func (rd *EC2VPCDeleter) RequestEC2InstanceReservationsFromVPCs() ([]*ec2.Reservation, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: rd.ResourceNames.AWSStringSlice()},
		},
	}
	irs := make([]*ec2.Reservation, 0)

	for {
		ctx := aws.BackgroundContext()
		resp, err := rd.GetClient().DescribeInstancesWithContext(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, r := range resp.Reservations {
			irs = append(irs, r)
		}

		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}

		params.NextToken = resp.NextToken
	}

	return irs, nil
}

// RequestEC2InternetGatewaysFromVPCs requests EC2 internet gateways by vpc names from the AWS API
func (rd *EC2VPCDeleter) RequestEC2InternetGatewaysFromVPCs() ([]*ec2.InternetGateway, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeInternetGatewaysInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("attachment.vpc-id"), Values: rd.ResourceNames.AWSStringSlice()},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeInternetGatewaysWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return resp.InternetGateways, nil
}

// RequestEC2NatGatewaysFromVPCs requests EC2 nat gateways by vpc names from the AWS API
func (rd *EC2VPCDeleter) RequestEC2NatGatewaysFromVPCs() ([]*ec2.NatGateway, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeNatGatewaysInput{
		Filter: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: rd.ResourceNames.AWSStringSlice()},
		},
	}
	ngws := make([]*ec2.NatGateway, 0)

	for {
		ctx := aws.BackgroundContext()
		resp, err := rd.GetClient().DescribeNatGatewaysWithContext(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, ngw := range resp.NatGateways {
			if ngw.State != nil && *ngw.State != "deleting" && *ngw.State != "deleted" {
				ngws = append(ngws, ngw)
			}
		}

		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}

		params.NextToken = resp.NextToken
	}

	return ngws, nil
}

// RequestEC2NetworkInterfacesFromVPCs requests EC2 network interfaces by vpc names from the AWS API
func (rd *EC2VPCDeleter) RequestEC2NetworkInterfacesFromVPCs() ([]*ec2.NetworkInterface, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: rd.ResourceNames.AWSStringSlice()},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeNetworkInterfacesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return resp.NetworkInterfaces, nil
}

// RequestEC2RouteTablesFromVPCs requests EC2 subnet-routetable associations by vpc names from the AWS API
func (rd *EC2VPCDeleter) RequestEC2RouteTablesFromVPCs() ([]*ec2.RouteTable, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: rd.ResourceNames.AWSStringSlice()},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeRouteTablesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	rts := make([]*ec2.RouteTable, 0)
	for _, rt := range resp.RouteTables {
		for _, a := range rt.Associations {
			if a.Main != nil && !*a.Main {
				rts = append(rts, rt)
				break
			}
		}
	}

	return rts, nil
}

// RequestEC2SecurityGroupsFromVPCs requests EC2 security groups by vpc names from the AWS API
func (rd *EC2VPCDeleter) RequestEC2SecurityGroupsFromVPCs() ([]*ec2.SecurityGroup, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: rd.ResourceNames.AWSStringSlice()},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeSecurityGroupsWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	sgs := make([]*ec2.SecurityGroup, 0)
	for _, sg := range resp.SecurityGroups {
		if sg.GroupName != nil && *sg.GroupName != defaultGroupName {
			sgs = append(sgs, sg)
		}
	}

	return sgs, nil
}

// RequestEC2SubnetsFromVPCs requests EC2 subnets by vpc names from the AWS API
func (rd *EC2VPCDeleter) RequestEC2SubnetsFromVPCs() ([]*ec2.Subnet, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: rd.ResourceNames.AWSStringSlice()},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeSubnetsWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	sns := make([]*ec2.Subnet, 0)
	for _, sn := range resp.Subnets {
		if sn.DefaultForAz != nil && !*sn.DefaultForAz {
			sns = append(sns, sn)
		}
	}

	return sns, nil
}

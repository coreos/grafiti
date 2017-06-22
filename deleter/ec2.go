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

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
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
		_, err := rd.Client.DetachInternetGatewayWithContext(ctx, params)
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
	if ierr != nil {
		if !cfg.IgnoreErrors {
			return ierr
		}
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
			if !cfg.IgnoreErrors {
				return err
			}
		}
	}

	fmtStr := "Deleted EC2 InternetGateway"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	var params *ec2.DeleteInternetGatewayInput
	for _, n := range rd.ResourceNames {
		params = &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: n.AWSString(),
			DryRun:            aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.Client.DeleteInternetGatewayWithContext(ctx, params)
		if err != nil {
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, n)
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

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.Client.DescribeInternetGatewaysWithContext(ctx, params)
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

// AddResourceNames adds EC2 NAT gateway names to ResourceNames
func (rd *EC2NatGatewayDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes EC2 NAT gateways from AWS
func (rd *EC2NatGatewayDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	fmtStr := "Deleted EC2 NatGateway"
	if cfg.DryRun {
		for _, n := range rd.ResourceNames {
			fmt.Println(drStr, fmtStr, n)
		}
		return nil
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	var params *ec2.DeleteNatGatewayInput
	for _, n := range rd.ResourceNames {
		params = &ec2.DeleteNatGatewayInput{
			NatGatewayId: n.AWSString(),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.Client.DeleteNatGatewayWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2NatGatewayRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, n)
	}

	// Wait for NAT Gateways to delete
	time.Sleep(time.Duration(1) * time.Minute)
	return nil
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

// DeleteResources deletes EC2 route table routes from AWS
func (rd *EC2RouteTableRouteDeleter) DeleteResources(cfg *DeleteConfig) error {
	if rd.RouteTable == nil {
		return nil
	}

	fmtStr := "Deleted RouteTable Route"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	var params *ec2.DeleteRouteInput
	for _, r := range rd.RouteTable.Routes {
		if r.GatewayId != nil && *r.GatewayId == "local" {
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
		_, err := rd.Client.DeleteRouteWithContext(ctx, params)
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

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
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
		_, err := rd.Client.DisassociateRouteTableWithContext(ctx, params)
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
		if err := rtrDel.DeleteResources(cfg); err != nil && !cfg.IgnoreErrors {
			return err
		}
	}

	// Delete route table
	fmtStr := "Deleted EC2 RouteTable"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	var params *ec2.DeleteRouteTableInput
	for _, n := range rd.ResourceNames {
		params = &ec2.DeleteRouteTableInput{
			RouteTableId: n.AWSString(),
			DryRun:       aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.Client.DeleteRouteTableWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2RouteTableRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, n)
	}

	return nil
}

// RequestEC2RouteTables requests EC2 route tables by route table name from the AWS API
func (rd *EC2RouteTableDeleter) RequestEC2RouteTables() ([]*ec2.RouteTable, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	params := &ec2.DescribeRouteTablesInput{
		RouteTableIds: rd.ResourceNames.AWSStringSlice(),
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.Client.DescribeRouteTablesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return resp.RouteTables, nil
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

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
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
		_, err := rd.Client.RevokeSecurityGroupIngressWithContext(ctx, params)
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

// DeleteResources deletes EC2 security group egress rules from AWS
func (rd *EC2SecurityGroupEgressRuleDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.SecurityGroups) == 0 {
		return nil
	}

	fmtStr := "Deleted EC2 SecurityGroup Egress Rule"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
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
		_, err := rd.Client.RevokeSecurityGroupEgressWithContext(ctx, params)
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
	if err := sgIngressDel.DeleteResources(cfg); err != nil && !cfg.IgnoreErrors {
		return err
	}
	sgEgressDel := &EC2SecurityGroupIngressRuleDeleter{SecurityGroups: sgs}
	if err := sgEgressDel.DeleteResources(cfg); err != nil && !cfg.IgnoreErrors {
		return err
	}

	fmtStr := "Deleted EC2 SecurityGroup"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
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
		_, err := rd.Client.DeleteSecurityGroupWithContext(ctx, params)
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

// RequestEC2SecurityGroups requests EC2 security groups by security group name from the AWS API
func (rd *EC2SecurityGroupDeleter) RequestEC2SecurityGroups() ([]*ec2.SecurityGroup, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &ec2.DescribeSecurityGroupsInput{
		GroupIds: rd.ResourceNames.AWSStringSlice(),
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.Client.DescribeSecurityGroupsWithContext(ctx, params)
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

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
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
		_, err := rd.Client.DeleteSubnetWithContext(ctx, params)
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

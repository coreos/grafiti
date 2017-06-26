package deleter

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/coreos/grafiti/arn"
)

const (
	deletingState            = "deleting"
	deletedState             = "deleted"
	detachingState           = "detaching"
	detachedState            = "detached"
	localInternetGatewayID   = "local"
	defaultSecurityGroupName = "default"
)

func isDeleting(state string) bool {
	return state == deletingState || state == deletedState
}

func isDetaching(state string) bool {
	return state == detachingState || state == detachedState
}

// Filter keys
const (
	cgwFilterKey           = "customer-gateway-id"
	eniFilterKey           = "network-interface-id"
	instanceFilterKey      = "instance-id"
	igwFilterKey           = "internet-gateway-id"
	ngwFilterKey           = "nat-gateway-id"
	naclFilterKey          = "network-acl-id"
	rtbFilterKey           = "route-table-id"
	sgFilterKey            = "group-id"
	subnetFilterKey        = "subnet-id"
	vpcFilterKey           = "vpc-id"
	vpcAttachmentFilterKey = "attachment.vpc-id"
	vconnFilterKey         = "vpn-connection-id"
	vgwFilterKey           = "vpn-gateway-id"
)

// EC2Client aliases an EC2API so requestEC2* functions can be shared between
// RequestEC2* functions
type EC2Client struct {
	ec2iface.EC2API
}

// EC2CustomerGatewayDeleter represents a collection of AWS EC2 customer gateways
type EC2CustomerGatewayDeleter struct {
	Client        EC2Client
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2CustomerGatewayDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2CustomerGatewayDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
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

	var params *ec2.DeleteCustomerGatewayInput
	for _, cgw := range cgws {
		params = &ec2.DeleteCustomerGatewayInput{
			CustomerGatewayId: cgw.CustomerGatewayId,
			DryRun:            aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteCustomerGatewayWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Println(drStr, fmtStr, *cgw.CustomerGatewayId)
				continue
			}
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

	size, chunk := len(rd.ResourceNames), 200
	cgws := make([]*ec2.CustomerGateway, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		cgws, err = rd.GetClient().requestEC2CustomerGateways(cgwFilterKey, rd.ResourceNames[i:stop], cgws)
		if err != nil {
			return cgws, err
		}
	}
	return cgws, nil
}

// Requesting customer gateways using filters prevents API errors caused by
// requesting non-existent customer gateways
func (c *EC2Client) requestEC2CustomerGateways(filterKey string, chunk arn.ResourceNames, cgws []*ec2.CustomerGateway) ([]*ec2.CustomerGateway, error) {
	params := &ec2.DescribeCustomerGatewaysInput{
		Filters: []*ec2.Filter{
			{Name: aws.String(filterKey), Values: chunk.AWSStringSlice()},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := c.DescribeCustomerGatewaysWithContext(ctx, params)
	if err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err)
		return cgws, err
	}

	for _, cgw := range resp.CustomerGateways {
		if !isDeleting(aws.StringValue(cgw.State)) {
			cgws = append(cgws, cgw)
		}
	}

	return cgws, nil
}

// EC2ElasticIPAllocationDeleter represents a collection of AWS EC2 elastic IP allocations
type EC2ElasticIPAllocationDeleter struct {
	Client        EC2Client
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2ElasticIPAllocationDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "ResourceNames": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2ElasticIPAllocationDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
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

	var params *ec2.ReleaseAddressInput
	for _, n := range rd.ResourceNames {
		params = &ec2.ReleaseAddressInput{
			AllocationId: n.AWSString(),
			DryRun:       aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().ReleaseAddressWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Println(drStr, fmtStr, n)
				continue
			}
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
	Client        EC2Client
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2ElasticIPAssocationDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "ResourceNames": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2ElasticIPAssocationDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
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

	var params *ec2.DisassociateAddressInput
	for _, n := range rd.ResourceNames {
		params = &ec2.DisassociateAddressInput{
			AssociationId: n.AWSString(),
			DryRun:        aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DisassociateAddressWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Println(drStr, fmtStr, n)
				continue
			}
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
	Client        EC2Client
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2NetworkInterfaceDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "ResourceNames": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2NetworkInterfaceDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
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

	var params *ec2.DeleteNetworkInterfaceInput
	for _, eni := range enis {
		params = &ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: eni.NetworkInterfaceId,
			DryRun:             aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteNetworkInterfaceWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Println(drStr, fmtStr, *eni.NetworkInterfaceId)
				continue
			}
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

	size, chunk := len(rd.ResourceNames), 200
	enis := make([]*ec2.NetworkInterface, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		enis, err = rd.GetClient().requestEC2NetworkInterfaces(eniFilterKey, rd.ResourceNames[i:stop], enis)
		if err != nil {
			return enis, err
		}
	}

	return enis, nil
}

// Requesting network interfaces using filters prevents API errors caused by
// requesting non-existent network interfaces
func (c *EC2Client) requestEC2NetworkInterfaces(filterKey string, chunk arn.ResourceNames, enis []*ec2.NetworkInterface) ([]*ec2.NetworkInterface, error) {
	params := &ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String(filterKey), Values: chunk.AWSStringSlice()},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := c.DescribeNetworkInterfacesWithContext(ctx, params)
	if err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err)
		return enis, err
	}

	enis = append(enis, resp.NetworkInterfaces...)

	return enis, nil
}

// RequestEC2EIPAddressessFromNetworkInterfaces requests EC2 elastic IP addresses by
// network interface names from the AWS API
func (rd *EC2NetworkInterfaceDeleter) RequestEC2EIPAddressessFromNetworkInterfaces() ([]*ec2.Address, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	addresses := make([]*ec2.Address, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		addresses, err = rd.GetClient().requestEC2EIPAddresses(eniFilterKey, rd.ResourceNames[i:stop], addresses)
		if err != nil {
			return addresses, err
		}
	}

	return addresses, nil
}

func (c *EC2Client) requestEC2EIPAddresses(filterKey string, chunk arn.ResourceNames, addresses []*ec2.Address) ([]*ec2.Address, error) {
	params := &ec2.DescribeAddressesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String(filterKey), Values: chunk.AWSStringSlice()},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := c.DescribeAddressesWithContext(ctx, params)
	if err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err)
		return addresses, err
	}

	addresses = append(addresses, resp.Addresses...)

	return addresses, nil
}

// EC2NetworkInterfaceAttachmentDeleter represents a collection of AWS EC2 network interface attachments
type EC2NetworkInterfaceAttachmentDeleter struct {
	Client        EC2Client
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2NetworkInterfaceAttachmentDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "ResourceNames": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2NetworkInterfaceAttachmentDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
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

	var params *ec2.DetachNetworkInterfaceInput
	for _, n := range rd.ResourceNames {
		params = &ec2.DetachNetworkInterfaceInput{
			AttachmentId: n.AWSString(),
			Force:        aws.Bool(true),
			DryRun:       aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DetachNetworkInterfaceWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Println(drStr, fmtStr, n)
				continue
			}
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

// EC2NetworkACLEntryDeleter represents a collection of AWS EC2 network acl entries
type EC2NetworkACLEntryDeleter struct {
	Client            EC2Client
	ResourceType      arn.ResourceType
	NetworkACLName    arn.ResourceName
	NetworkACLEntries []*ec2.NetworkAclEntry
}

func (rd *EC2NetworkACLEntryDeleter) String() string {
	aclEntries := []struct {
		RuleNumber int64
		Egress     bool
	}{}
	for _, entry := range rd.NetworkACLEntries {
		aclEntries = append(aclEntries, struct {
			RuleNumber int64
			Egress     bool
		}{*entry.RuleNumber, *entry.Egress})
	}
	return fmt.Sprintf(`{"Type": "%s", "NetworkACLName": %v, "NetworkACLEntries": %v}`, rd.ResourceType, rd.NetworkACLName, aclEntries)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2NetworkACLEntryDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
}

// DeleteResources deletes EC2 network acl entries from AWS
func (rd *EC2NetworkACLEntryDeleter) DeleteResources(cfg *DeleteConfig) error {
	if rd.NetworkACLName == "" || len(rd.NetworkACLEntries) == 0 {
		return nil
	}

	fmtStr := "Deleted EC2 NetworkAcl"

	var params *ec2.DeleteNetworkAclEntryInput
	for _, entry := range rd.NetworkACLEntries {
		params = &ec2.DeleteNetworkAclEntryInput{
			NetworkAclId: rd.NetworkACLName.AWSString(),
			RuleNumber:   entry.RuleNumber,
			Egress:       entry.Egress,
			DryRun:       aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteNetworkAclEntryWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Printf("%s %s %s Entry %d\n", drStr, fmtStr, rd.NetworkACLName, *entry.RuleNumber)
				continue
			}
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Printf("%s %s Entry %d\n", fmtStr, rd.NetworkACLName, *entry.RuleNumber)
	}

	return nil
}

// EC2NetworkACLDeleter represents a collection of AWS EC2 network acl's
type EC2NetworkACLDeleter struct {
	Client        EC2Client
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2NetworkACLDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "ResourceNames": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2NetworkACLDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
}

// AddResourceNames adds EC2 network acl names to ResourceNames
func (rd *EC2NetworkACLDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes EC2 network acl's from AWS
func (rd *EC2NetworkACLDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	acls, rerr := rd.RequestEC2NetworkACLs()
	if rerr != nil && !cfg.IgnoreErrors {
		return rerr
	}

	fmtStr := "Deleted EC2 NetworkAcl"

	var (
		params       *ec2.DeleteNetworkAclInput
		naclEntryDel *EC2NetworkACLEntryDeleter
	)
	for _, acl := range acls {
		// First delete network acl entries
		naclEntryDel = &EC2NetworkACLEntryDeleter{NetworkACLName: arn.ResourceName(*acl.NetworkAclId)}
		if err := naclEntryDel.DeleteResources(cfg); err != nil {
			return err
		}

		params = &ec2.DeleteNetworkAclInput{
			NetworkAclId: acl.NetworkAclId,
			DryRun:       aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteNetworkAclWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Println(drStr, fmtStr, *acl.NetworkAclId)
				continue
			}
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, *acl.NetworkAclId)
	}

	return nil
}

// RequestEC2NetworkACLs retrieves network acl's by network acl ID
func (rd *EC2NetworkACLDeleter) RequestEC2NetworkACLs() ([]*ec2.NetworkAcl, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	acls := make([]*ec2.NetworkAcl, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		acls, err = rd.GetClient().requestEC2NetworkACLs(naclFilterKey, rd.ResourceNames[i:stop], acls)
		if err != nil {
			return acls, err
		}
	}

	return acls, nil
}

// Requesting network acl's using filters prevents API errors caused by
// requesting non-existent network acl's
func (c *EC2Client) requestEC2NetworkACLs(filterKey string, chunk arn.ResourceNames, acls []*ec2.NetworkAcl) ([]*ec2.NetworkAcl, error) {
	params := &ec2.DescribeNetworkAclsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String(filterKey), Values: chunk.AWSStringSlice()},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := c.DescribeNetworkAclsWithContext(ctx, params)
	if err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err)
		return acls, err
	}

	for _, acl := range resp.NetworkAcls {
		if !isDefaultNACL(acl) {
			acls = append(acls, acl)
		}
	}

	return acls, nil
}

func isDefaultNACL(acl *ec2.NetworkAcl) bool {
	return aws.BoolValue(acl.IsDefault)
}

// EC2InstanceDeleter represents a collection of AWS EC2 instances
type EC2InstanceDeleter struct {
	Client        EC2Client
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2InstanceDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "ResourceNames": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2InstanceDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
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

	instances, rerr := rd.RequestEC2Instances()
	if rerr != nil && !cfg.IgnoreErrors {
		return rerr
	}

	instanceNames := make(arn.ResourceNames, 0, len(instances))
	for _, instance := range instances {
		instanceNames = append(instanceNames, arn.ResourceName(*instance.InstanceId))
	}

	if len(instanceNames) == 0 {
		return nil
	}

	fmtStr := "Terminated EC2 Instance"

	params := &ec2.TerminateInstancesInput{
		InstanceIds: instanceNames.AWSStringSlice(),
		DryRun:      aws.Bool(cfg.DryRun),
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().TerminateInstancesWithContext(ctx, params)
	if err != nil {
		if isDryRun(err) {
			for _, n := range instanceNames {
				fmt.Println(drStr, fmtStr, n)
			}
			return nil
		}
		for _, n := range instanceNames {
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

	for _, id := range instanceNames {
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

// RequestEC2Instances requests EC2 instances by instance names from the AWS API
func (rd *EC2InstanceDeleter) RequestEC2Instances() ([]*ec2.Instance, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	instances := make([]*ec2.Instance, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		instances, err = rd.GetClient().requestEC2Instances(instanceFilterKey, rd.ResourceNames[i:stop], instances)
		if err != nil {
			return instances, err
		}
	}

	return instances, nil
}

// Requesting nat gateways using filters prevents API errors caused by
// requesting non-existent nat gateways
func (c *EC2Client) requestEC2Instances(filterKey string, chunk arn.ResourceNames, instances []*ec2.Instance) ([]*ec2.Instance, error) {
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String(filterKey), Values: chunk.AWSStringSlice()},
		},
	}

	for {
		ctx := aws.BackgroundContext()
		resp, err := c.DescribeInstancesWithContext(ctx, params)
		if err != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", err)
			return nil, err
		}

		for _, reservation := range resp.Reservations {
			for _, instance := range reservation.Instances {
				if !isInstanceTerminating(instance) {
					instances = append(instances, instance)
				}
			}
		}

		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}

		params.NextToken = resp.NextToken
	}

	return instances, nil
}

func isInstanceTerminating(instance *ec2.Instance) bool {
	// Instance is shutting down (32) or terminated (48)
	return instance.State.Code != nil && (*instance.State.Code == 32 || *instance.State.Code == 48)
}

// RequestIAMInstanceProfilesFromInstances retrieves instance profiles from instance ID's
func (rd *EC2InstanceDeleter) RequestIAMInstanceProfilesFromInstances() ([]*iam.InstanceProfile, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	instances, ierr := rd.RequestEC2Instances()
	if ierr != nil || len(instances) == 0 {
		return nil, ierr
	}

	// We cannot request instance profiles by their ID's so we must search
	// iteratively with a map
	want := map[string]struct{}{}
	var iprName string
	for _, instance := range instances {
		if instance.IamInstanceProfile == nil || instance.IamInstanceProfile.Arn == nil {
			continue
		}

		iprSplit := strings.Split(*instance.IamInstanceProfile.Arn, "instance-profile/")
		if len(iprSplit) != 2 || iprSplit[1] == "" {
			continue
		}
		iprName = iprSplit[1]

		if _, ok := want[iprName]; !ok {
			want[iprName] = struct{}{}
		}
	}

	svc := iam.New(setUpAWSSession())
	iprs := make([]*iam.InstanceProfile, 0)
	params := new(iam.ListInstanceProfilesInput)
	for {
		ctx := aws.BackgroundContext()
		resp, err := svc.ListInstanceProfilesWithContext(ctx, params)
		if err != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", err)
			return nil, err
		}

		for _, ipr := range resp.InstanceProfiles {
			if _, ok := want[*ipr.InstanceProfileName]; ok {
				iprs = append(iprs, ipr)
			}
		}

		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}

		params.Marker = resp.Marker
	}

	return iprs, nil
}

// EC2InternetGatewayAttachmentDeleter represents a collection of AWS EC2 internet gateway attachments
type EC2InternetGatewayAttachmentDeleter struct {
	Client              EC2Client
	ResourceType        arn.ResourceType
	InternetGatewayName arn.ResourceName
	AttachmentNames     arn.ResourceNames
}

func (rd *EC2InternetGatewayAttachmentDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "InternetGatewayName": "%s", "AttachmentNames": %v}`, rd.ResourceType, rd.InternetGatewayName, rd.AttachmentNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2InternetGatewayAttachmentDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
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

	var params *ec2.DetachInternetGatewayInput
	for _, an := range rd.AttachmentNames {
		params = &ec2.DetachInternetGatewayInput{
			InternetGatewayId: rd.InternetGatewayName.AWSString(),
			DryRun:            aws.Bool(cfg.DryRun),
			VpcId:             an.AWSString(),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DetachInternetGatewayWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Printf("%s %s %s from VPC %s\n", drStr, fmtStr, rd.InternetGatewayName, an)
				continue
			}
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
	Client        EC2Client
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2InternetGatewayDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2InternetGatewayDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
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

	var params *ec2.DeleteInternetGatewayInput
	for _, igw := range igws {
		params = &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: igw.InternetGatewayId,
			DryRun:            aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteInternetGatewayWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Println(drStr, fmtStr, *igw.InternetGatewayId)
				continue
			}
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

	size, chunk := len(rd.ResourceNames), 200
	igws := make([]*ec2.InternetGateway, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		igws, err = rd.GetClient().requestEC2InternetGateways(igwFilterKey, rd.ResourceNames[i:stop], igws)
		if err != nil {
			return igws, err
		}
	}

	return igws, nil
}

// Requesting internet gateways using filters prevents API errors caused by
// requesting non-existent internet gateways
func (c *EC2Client) requestEC2InternetGateways(filterKey string, chunk arn.ResourceNames, igws []*ec2.InternetGateway) ([]*ec2.InternetGateway, error) {
	params := &ec2.DescribeInternetGatewaysInput{
		Filters: []*ec2.Filter{
			{Name: aws.String(filterKey), Values: chunk.AWSStringSlice()},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := c.DescribeInternetGatewaysWithContext(ctx, params)
	if err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err)
		return igws, err
	}

	igws = append(igws, resp.InternetGateways...)

	return igws, nil
}

// EC2NatGatewayDeleter represents a collection of AWS EC2 NAT gateways
type EC2NatGatewayDeleter struct {
	Client        EC2Client
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2NatGatewayDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2NatGatewayDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
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

	var params *ec2.DeleteNatGatewayInput
	for _, ngw := range ngws {
		if cfg.DryRun {
			fmt.Println(drStr, fmtStr, *ngw.NatGatewayId)
			continue
		}

		params = &ec2.DeleteNatGatewayInput{
			NatGatewayId: ngw.NatGatewayId,
		}

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

	return nil
}

// RequestEC2NatGateways requests EC2 nat gateways by name from the AWS API
func (rd *EC2NatGatewayDeleter) RequestEC2NatGateways() ([]*ec2.NatGateway, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	ngws := make([]*ec2.NatGateway, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		ngws, err = rd.GetClient().requestEC2NatGateways(ngwFilterKey, rd.ResourceNames[i:stop], ngws)
		if err != nil {
			return ngws, err
		}
	}

	return ngws, nil
}

// Requesting nat gateways using filters prevents API errors caused by
// requesting non-existent nat gateways
func (c *EC2Client) requestEC2NatGateways(filterKey string, chunk arn.ResourceNames, ngws []*ec2.NatGateway) ([]*ec2.NatGateway, error) {
	params := &ec2.DescribeNatGatewaysInput{
		Filter: []*ec2.Filter{
			{Name: aws.String(filterKey), Values: chunk.AWSStringSlice()},
		},
	}

	for {
		ctx := aws.BackgroundContext()
		resp, err := c.DescribeNatGatewaysWithContext(ctx, params)
		if err != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", err)
			return nil, err
		}

		for _, ngw := range resp.NatGateways {
			if !isDeleting(aws.StringValue(ngw.State)) {
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

// EC2RouteTableRouteDeleter represents a collection of AWS EC2 route table routes
type EC2RouteTableRouteDeleter struct {
	Client       EC2Client
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
func (rd *EC2RouteTableRouteDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
}

// DeleteResources deletes EC2 route table routes from AWS
func (rd *EC2RouteTableRouteDeleter) DeleteResources(cfg *DeleteConfig) error {
	if rd.RouteTable == nil {
		return nil
	}

	fmtStr := "Deleted RouteTable Route"

	var params *ec2.DeleteRouteInput
	for _, r := range rd.RouteTable.Routes {
		if isLocalGateway(r.GatewayId) {
			continue
		}

		params = &ec2.DeleteRouteInput{
			DestinationCidrBlock: r.DestinationCidrBlock,
			RouteTableId:         rd.RouteTable.RouteTableId,
			DryRun:               aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteRouteWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Printf("%s %s: Dst CIDR Block %s\n", drStr, fmtStr, *r.DestinationCidrBlock)
				continue
			}
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

func isLocalGateway(gwID *string) bool {
	return aws.StringValue(gwID) == localInternetGatewayID
}

// EC2RouteTableAssociationDeleter represents a collection of AWS EC2 route table associations
type EC2RouteTableAssociationDeleter struct {
	Client        EC2Client
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2RouteTableAssociationDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2RouteTableAssociationDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
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

	var params *ec2.DisassociateRouteTableInput
	for _, n := range rd.ResourceNames {
		params = &ec2.DisassociateRouteTableInput{
			AssociationId: n.AWSString(),
			DryRun:        aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DisassociateRouteTableWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Println(drStr, fmtStr, n)
				continue
			}
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
	Client        EC2Client
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2RouteTableDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2RouteTableDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
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

	var params *ec2.DeleteRouteTableInput
	for _, rt := range rts {
		params = &ec2.DeleteRouteTableInput{
			RouteTableId: rt.RouteTableId,
			DryRun:       aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteRouteTableWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Println(drStr, fmtStr, *rt.RouteTableId)
				continue
			}
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

	size, chunk := len(rd.ResourceNames), 200
	rtbs := make([]*ec2.RouteTable, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		rtbs, err = rd.GetClient().requestEC2RouteTables(rtbFilterKey, rd.ResourceNames[i:stop], rtbs)
		if err != nil {
			return rtbs, err
		}
	}

	return rtbs, nil
}

// Requesting route tables using filters prevents API errors caused by
// requesting non-existent route tables
func (c *EC2Client) requestEC2RouteTables(filterKey string, chunk arn.ResourceNames, rtbs []*ec2.RouteTable) ([]*ec2.RouteTable, error) {
	params := &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String(filterKey), Values: chunk.AWSStringSlice()},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := c.DescribeRouteTablesWithContext(ctx, params)
	if err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err)
		return rtbs, err
	}

	for _, rtb := range resp.RouteTables {
		if !isMainRouteTable(rtb) {
			rtbs = append(rtbs, rtb)
		}
	}

	return rtbs, nil
}

// If a route tables' association is a main association, the route table
// cannot be deleted explicitly
func isMainRouteTable(rtb *ec2.RouteTable) bool {
	for _, a := range rtb.Associations {
		if a.Main != nil && *a.Main {
			return true
		}
	}
	return false
}

// EC2SecurityGroupIngressRuleDeleter represents a collection of AWS EC2 security group ingress rules
type EC2SecurityGroupIngressRuleDeleter struct {
	Client         EC2Client
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
func (rd *EC2SecurityGroupIngressRuleDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
}

// DeleteResources deletes EC2 security group ingress rules from AWS
// NOTE: all security group ingress references must be removed before deleting before
// deleting a security group ingress
func (rd *EC2SecurityGroupIngressRuleDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.SecurityGroups) == 0 {
		return nil
	}

	fmtStr := "Deleted EC2 SecurityGroup Ingress Rule"

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

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().RevokeSecurityGroupIngressWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Printf("%s %s for %s\n", drStr, fmtStr, *sg.GroupId)
				continue
			}
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
	Client         EC2Client
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
func (rd *EC2SecurityGroupEgressRuleDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
}

// DeleteResources deletes EC2 security group egress rules from AWS
func (rd *EC2SecurityGroupEgressRuleDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.SecurityGroups) == 0 {
		return nil
	}

	fmtStr := "Deleted EC2 SecurityGroup Egress Rule"

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

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().RevokeSecurityGroupEgressWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Printf("%s %s for %s\n", drStr, fmtStr, *sg.GroupId)
				continue
			}
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
	Client        EC2Client
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2SecurityGroupDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2SecurityGroupDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
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
	sgEgressDel := &EC2SecurityGroupEgressRuleDeleter{SecurityGroups: sgs}
	if err := sgEgressDel.DeleteResources(cfg); err != nil {
		return err
	}

	fmtStr := "Deleted EC2 SecurityGroup"

	var params *ec2.DeleteSecurityGroupInput
	for _, sg := range sgs {
		params = &ec2.DeleteSecurityGroupInput{
			GroupId: sg.GroupId,
			DryRun:  aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteSecurityGroupWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Println(drStr, fmtStr, *sg.GroupId)
				continue
			}
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

// RequestEC2SecurityGroups requests EC2 security groups by name from the AWS API
func (rd *EC2SecurityGroupDeleter) RequestEC2SecurityGroups() ([]*ec2.SecurityGroup, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	sgs := make([]*ec2.SecurityGroup, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		sgs, err = rd.GetClient().requestEC2SecurityGroups(sgFilterKey, rd.ResourceNames[i:stop], sgs)
		if err != nil {
			return sgs, err
		}
	}

	return sgs, nil
}

// Requesting security groups using filters prevents API errors caused by
// requesting non-existent security groups
func (c *EC2Client) requestEC2SecurityGroups(filterKey string, chunk arn.ResourceNames, sgs []*ec2.SecurityGroup) ([]*ec2.SecurityGroup, error) {
	params := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String(filterKey), Values: chunk.AWSStringSlice()},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := c.DescribeSecurityGroupsWithContext(ctx, params)
	if err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err)
		return sgs, err
	}

	for _, sg := range resp.SecurityGroups {
		if !isDefaultSecurityGroup(sg) {
			sgs = append(sgs, sg)
		}
	}

	return sgs, nil
}

// Default security groups cannot be deleted, so remove them from response
// elements
func isDefaultSecurityGroup(sg *ec2.SecurityGroup) bool {
	return aws.StringValue(sg.GroupName) == defaultSecurityGroupName
}

// EC2SubnetDeleter represents a collection of AWS EC2 subnets
type EC2SubnetDeleter struct {
	Client        EC2Client
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2SubnetDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2SubnetDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
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

	sns, rerr := rd.RequestEC2Subnets()
	if rerr != nil && !cfg.IgnoreErrors {
		return rerr
	}

	fmtStr := "Deleted EC2 Subnet"

	var params *ec2.DeleteSubnetInput
	for _, sn := range sns {
		params = &ec2.DeleteSubnetInput{
			SubnetId: sn.SubnetId,
			DryRun:   aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteSubnetWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Println(drStr, fmtStr, *sn.SubnetId)
				continue
			}
			cfg.logDeleteError(arn.EC2SubnetRType, arn.ResourceName(*sn.SubnetId), err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, *sn.SubnetId)
	}

	return nil
}

// RequestEC2Subnets requests EC2 subnets by subnet name from the AWS API
func (rd *EC2SubnetDeleter) RequestEC2Subnets() ([]*ec2.Subnet, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	subnets := make([]*ec2.Subnet, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		subnets, err = rd.GetClient().requestEC2Subnets(subnetFilterKey, rd.ResourceNames[i:stop], subnets)
		if err != nil {
			return subnets, err
		}
	}

	return subnets, nil
}

// Requesting subnets using filters prevents API errors caused by requesting
// non-existent subnets
func (c *EC2Client) requestEC2Subnets(filterKey string, chunk arn.ResourceNames, subnets []*ec2.Subnet) ([]*ec2.Subnet, error) {
	params := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String(filterKey), Values: chunk.AWSStringSlice()},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := c.DescribeSubnetsWithContext(ctx, params)
	if err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err)
		return subnets, err
	}

	for _, subnet := range resp.Subnets {
		if !isDefaultSubnet(subnet) {
			subnets = append(subnets, subnet)
		}
	}

	return subnets, nil
}

func isDefaultSubnet(sn *ec2.Subnet) bool {
	return aws.BoolValue(sn.DefaultForAz)
}

// EC2VPCCIDRBlockAssociationDeleter represents a collection of AWS EC2 VPC CIDR block associations
type EC2VPCCIDRBlockAssociationDeleter struct {
	Client              EC2Client
	ResourceType        arn.ResourceType
	VPCName             arn.ResourceName
	VPCAssociationNames arn.ResourceNames
}

func (rd *EC2VPCCIDRBlockAssociationDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.VPCAssociationNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2VPCCIDRBlockAssociationDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
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

	var params *ec2.DisassociateVpcCidrBlockInput
	for _, n := range rd.VPCAssociationNames {
		if cfg.DryRun {
			fmt.Printf("%s Deleted EC2 VPC %s CIDRBlockAssociation\n", drStr, n)
			continue
		}

		params = &ec2.DisassociateVpcCidrBlockInput{
			AssociationId: n.AWSString(),
		}

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
	Client        EC2Client
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2VPCDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2VPCDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
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

	var params *ec2.DeleteVpcInput
	for _, vpc := range vpcs {
		params = &ec2.DeleteVpcInput{
			VpcId:  vpc.VpcId,
			DryRun: aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteVpcWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Println(drStr, fmtStr, *vpc.VpcId)
				continue
			}
			cfg.logDeleteError(arn.EC2VPCRType, arn.ResourceName(*vpc.VpcId), err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, *vpc.VpcId)
	}

	return nil
}

// RequestEC2VPCs requests EC2 vpc's by vpc names from the AWS API
func (rd *EC2VPCDeleter) RequestEC2VPCs() ([]*ec2.Vpc, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	vpcs := make([]*ec2.Vpc, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		vpcs, err = rd.GetClient().requestEC2VPCs(vpcFilterKey, rd.ResourceNames[i:stop], vpcs)
		if err != nil {
			return vpcs, err
		}
	}

	return vpcs, nil
}

// Requesting vpc's using filters prevents API errors caused by requesting
// non-existent vpc's
func (c *EC2Client) requestEC2VPCs(filterKey string, chunk arn.ResourceNames, vpcs []*ec2.Vpc) ([]*ec2.Vpc, error) {
	params := &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String(filterKey), Values: chunk.AWSStringSlice()},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := c.DescribeVpcsWithContext(ctx, params)
	if err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err)
		return vpcs, err
	}

	for _, vpc := range resp.Vpcs {
		if !isDefaultVPC(vpc) {
			vpcs = append(vpcs, vpc)
		}
	}

	return vpcs, nil
}

func isDefaultVPC(vpc *ec2.Vpc) bool {
	return aws.BoolValue(vpc.IsDefault)
}

// RequestEC2InstancesFromVPCs requests EC2 instance reservations from vpc names from the AWS API
func (rd *EC2VPCDeleter) RequestEC2InstancesFromVPCs() ([]*ec2.Instance, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	instances := make([]*ec2.Instance, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		instances, err = rd.GetClient().requestEC2Instances(vpcFilterKey, rd.ResourceNames[i:stop], instances)
		if err != nil {
			return instances, err
		}
	}

	return instances, nil
}

// RequestEC2InternetGatewaysFromVPCs requests EC2 internet gateways by vpc names from the AWS API
func (rd *EC2VPCDeleter) RequestEC2InternetGatewaysFromVPCs() ([]*ec2.InternetGateway, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	igws := make([]*ec2.InternetGateway, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		igws, err = rd.GetClient().requestEC2InternetGateways(vpcAttachmentFilterKey, rd.ResourceNames[i:stop], igws)
		if err != nil {
			return igws, err
		}
	}

	return igws, nil
}

// RequestEC2NatGatewaysFromVPCs requests EC2 nat gateways by vpc names from the AWS API
func (rd *EC2VPCDeleter) RequestEC2NatGatewaysFromVPCs() ([]*ec2.NatGateway, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	ngws := make([]*ec2.NatGateway, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		ngws, err = rd.GetClient().requestEC2NatGateways(vpcFilterKey, rd.ResourceNames[i:stop], ngws)
		if err != nil {
			return ngws, err
		}
	}

	return ngws, nil
}

// RequestEC2NetworkInterfacesFromVPCs requests EC2 network interfaces by vpc names from the AWS API
func (rd *EC2VPCDeleter) RequestEC2NetworkInterfacesFromVPCs() ([]*ec2.NetworkInterface, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	enis := make([]*ec2.NetworkInterface, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		enis, err = rd.GetClient().requestEC2NetworkInterfaces(vpcFilterKey, rd.ResourceNames[i:stop], enis)
		if err != nil {
			return enis, err
		}
	}

	return enis, nil
}

// RequestEC2RouteTablesFromVPCs requests EC2 route tables by vpc names from the AWS API
func (rd *EC2VPCDeleter) RequestEC2RouteTablesFromVPCs() ([]*ec2.RouteTable, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	rtbs := make([]*ec2.RouteTable, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		rtbs, err = rd.GetClient().requestEC2RouteTables(vpcFilterKey, rd.ResourceNames[i:stop], rtbs)
		if err != nil {
			return rtbs, err
		}
	}

	return rtbs, nil
}

// RequestEC2SecurityGroupsFromVPCs requests EC2 security groups by vpc names
// from the AWS API
func (rd *EC2VPCDeleter) RequestEC2SecurityGroupsFromVPCs() ([]*ec2.SecurityGroup, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	sgs := make([]*ec2.SecurityGroup, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		sgs, err = rd.GetClient().requestEC2SecurityGroups(vpcFilterKey, rd.ResourceNames[i:stop], sgs)
		if err != nil {
			return sgs, err
		}
	}

	return sgs, nil
}

// RequestEC2SubnetsFromVPCs requests EC2 subnets by vpc names from the AWS API
func (rd *EC2VPCDeleter) RequestEC2SubnetsFromVPCs() ([]*ec2.Subnet, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	subnets := make([]*ec2.Subnet, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		subnets, err = rd.GetClient().requestEC2Subnets(vpcFilterKey, rd.ResourceNames[i:stop], subnets)
		if err != nil {
			return subnets, err
		}
	}

	return subnets, nil
}

// RequestEC2VPNGatewaysFromVPCs requests EC2 vpn gateways by vpc names from the AWS API
func (rd *EC2VPCDeleter) RequestEC2VPNGatewaysFromVPCs() ([]*ec2.VpnGateway, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	vgws := make([]*ec2.VpnGateway, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		vgws, err = rd.GetClient().requestEC2VPNGateways(vpcAttachmentFilterKey, rd.ResourceNames[i:stop], vgws)
		if err != nil {
			return vgws, err
		}
	}

	return vgws, nil
}

// EC2VPNConnectionRouteDeleter represents a collection of AWS EC2 vpn
// connection routes
type EC2VPNConnectionRouteDeleter struct {
	Client        EC2Client
	ResourceType  arn.ResourceType
	VPNConnection *ec2.VpnConnection
}

func (rd *EC2VPNConnectionRouteDeleter) String() string {
	routes := make([]string, 0)
	for _, route := range rd.VPNConnection.Routes {
		routes = append(routes, fmt.Sprintf(`{"VpnConnectionId": "%s", "DestinationCidrBlock": "%s"}`, *rd.VPNConnection.VpnConnectionId, *route.DestinationCidrBlock))
	}
	return fmt.Sprintf(`{"Type": "%s", "VPNRoutes": %v}`, rd.ResourceType, routes)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2VPNConnectionRouteDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
}

// DeleteResources deletes EC2 vpn connection routes from AWS
func (rd *EC2VPNConnectionRouteDeleter) DeleteResources(cfg *DeleteConfig) error {
	if rd.VPNConnection == nil {
		return nil
	}

	fmtStr := "Deleted EC2 VPN Connection Route"

	var params *ec2.DeleteVpnConnectionRouteInput
	for _, route := range rd.VPNConnection.Routes {
		if isDeleting(aws.StringValue(route.State)) {
			continue
		}

		params = &ec2.DeleteVpnConnectionRouteInput{
			DestinationCidrBlock: route.DestinationCidrBlock,
			VpnConnectionId:      rd.VPNConnection.VpnConnectionId,
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteVpnConnectionRouteWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Printf("%s %s %s from %s\n", drStr, fmtStr, *route.DestinationCidrBlock, *rd.VPNConnection.VpnConnectionId)
				continue
			}
			cfg.logDeleteError(arn.EC2VPNConnectionRouteRType, arn.ResourceName(*route.DestinationCidrBlock), err, logrus.Fields{
				"parent_resource_type": arn.EC2VPNConnectionRType,
				"parent_resource_name": *rd.VPNConnection.VpnConnectionId,
			})
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Printf("%s %s from %s\n", fmtStr, *route.DestinationCidrBlock, *rd.VPNConnection.VpnConnectionId)
	}

	return nil
}

// EC2VPNConnectionDeleter represents a collection of AWS EC2 vpn connections
type EC2VPNConnectionDeleter struct {
	Client        EC2Client
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2VPNConnectionDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2VPNConnectionDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
}

// AddResourceNames adds EC2 vpn connection names to ResourceNames
func (rd *EC2VPNConnectionDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes EC2 vpn connections from AWS
func (rd *EC2VPNConnectionDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	vcs, rerr := rd.RequestEC2VPNConnections()
	if rerr != nil && !cfg.IgnoreErrors {
		return rerr
	}

	fmtStr := "Deleted EC2 VPN Connection"

	var (
		params *ec2.DeleteVpnConnectionInput
		vcrDel *EC2VPNConnectionRouteDeleter
	)
	for _, vc := range vcs {
		if isDeleting(aws.StringValue(vc.State)) {
			continue
		}

		vcrDel = &EC2VPNConnectionRouteDeleter{VPNConnection: vc}
		if err := vcrDel.DeleteResources(cfg); err != nil {
			return err
		}

		params = &ec2.DeleteVpnConnectionInput{
			VpnConnectionId: vc.VpnConnectionId,
			DryRun:          aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteVpnConnectionWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Println(drStr, fmtStr, *vc.VpnConnectionId)
				continue
			}
			cfg.logDeleteError(arn.EC2VPNConnectionRType, arn.ResourceName(*vc.VpnConnectionId), err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, *vc.VpnConnectionId)
	}

	return nil
}

// RequestEC2VPNConnections requests EC2 vpn connections by names from the AWS API
func (rd *EC2VPNConnectionDeleter) RequestEC2VPNConnections() ([]*ec2.VpnConnection, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	vconns := make([]*ec2.VpnConnection, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		vconns, err = rd.GetClient().requestEC2VPNConnections(vconnFilterKey, rd.ResourceNames[i:stop], vconns)
		if err != nil {
			return vconns, err
		}
	}

	return vconns, nil
}

// Requesting vpn connections using filters prevents API errors caused by
// requesting non-existent vpn connections and requesting too many vpn
// connections in one request
func (c *EC2Client) requestEC2VPNConnections(filterKey string, chunk arn.ResourceNames, vconns []*ec2.VpnConnection) ([]*ec2.VpnConnection, error) {
	params := &ec2.DescribeVpnConnectionsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String(filterKey), Values: chunk.AWSStringSlice()},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := c.DescribeVpnConnectionsWithContext(ctx, params)
	if err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err)
		return vconns, err
	}

	for _, vconn := range resp.VpnConnections {
		if !isDeleting(aws.StringValue(vconn.State)) {
			vconns = append(vconns, vconn)
		}
	}

	return vconns, nil
}

// EC2VPNGatewayAttachmentDeleter represents a collection of AWS EC2 vpn gateway
// attachments
type EC2VPNGatewayAttachmentDeleter struct {
	Client       EC2Client
	ResourceType arn.ResourceType
	VPNGateway   *ec2.VpnGateway
}

func (rd *EC2VPNGatewayAttachmentDeleter) String() string {
	attachments := make([]string, 0)
	for _, attachment := range rd.VPNGateway.VpcAttachments {
		attachments = append(attachments, fmt.Sprintf(`{"VpnGatewayId": "%s", "VpcId": "%s"}`, *rd.VPNGateway.VpnGatewayId, *attachment.VpcId))
	}
	return fmt.Sprintf(`{"Type": "%s", "VPNAttachments": %v}`, rd.ResourceType, attachments)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2VPNGatewayAttachmentDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
}

// DeleteResources deletes EC2 vpn gateway attachments from AWS
func (rd *EC2VPNGatewayAttachmentDeleter) DeleteResources(cfg *DeleteConfig) error {
	if rd.VPNGateway == nil {
		return nil
	}

	fmtStr := "Detached EC2 VPN Gateway"

	var params *ec2.DetachVpnGatewayInput
	for _, attachment := range rd.VPNGateway.VpcAttachments {
		// Skip attachments that are in the process of detaching
		if isDetaching(aws.StringValue(attachment.State)) {
			continue
		}

		params = &ec2.DetachVpnGatewayInput{
			VpcId:        attachment.VpcId,
			VpnGatewayId: rd.VPNGateway.VpnGatewayId,
			DryRun:       aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DetachVpnGatewayWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Printf("%s %s %s from %s\n", drStr, fmtStr, *rd.VPNGateway.VpnGatewayId, *attachment.VpcId)
				continue
			}
			cfg.logDeleteError(arn.EC2VPNGatewayAttachmentRType, arn.ResourceName(*attachment.VpcId), err, logrus.Fields{
				"parent_resource_type": arn.EC2VPNGatewayRType,
				"parent_resource_name": *rd.VPNGateway.VpnGatewayId,
			})
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Printf("%s %s from %s\n", fmtStr, *rd.VPNGateway.VpnGatewayId, *attachment.VpcId)
	}

	return nil
}

// EC2VPNGatewayDeleter represents a collection of AWS EC2 vpn gateways
type EC2VPNGatewayDeleter struct {
	Client        EC2Client
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2VPNGatewayDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *EC2VPNGatewayDeleter) GetClient() *EC2Client {
	if rd.Client == (EC2Client{}) {
		rd.Client = EC2Client{ec2.New(setUpAWSSession())}
	}
	return &rd.Client
}

// AddResourceNames adds EC2 vpn gateway names to ResourceNames
func (rd *EC2VPNGatewayDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes EC2 vpn gateways from AWS
func (rd *EC2VPNGatewayDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	vgws, rerr := rd.RequestEC2VPNGateways()
	if rerr != nil && !cfg.IgnoreErrors {
		return rerr
	}

	// Now delete VPN gateway itself
	fmtStr := "Deleted EC2 VPN Gateway"

	var (
		params  *ec2.DeleteVpnGatewayInput
		vpnaDel *EC2VPNGatewayAttachmentDeleter
	)
	for _, vgw := range vgws {
		if isDeleting(aws.StringValue(vgw.State)) {
			continue
		}

		vpnaDel = &EC2VPNGatewayAttachmentDeleter{VPNGateway: vgw}
		if err := vpnaDel.DeleteResources(cfg); err != nil {
			return err
		}

		params = &ec2.DeleteVpnGatewayInput{
			VpnGatewayId: vgw.VpnGatewayId,
			DryRun:       aws.Bool(cfg.DryRun),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteVpnGatewayWithContext(ctx, params)
		if err != nil {
			if isDryRun(err) {
				fmt.Println(drStr, fmtStr, *vgw.VpnGatewayId)
				continue
			}
			cfg.logDeleteError(arn.EC2VPNGatewayRType, arn.ResourceName(*vgw.VpnGatewayId), err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, *vgw.VpnGatewayId)
	}

	return nil
}

// RequestEC2VPNGateways requests EC2 vpn gateways by names from the AWS API
func (rd *EC2VPNGatewayDeleter) RequestEC2VPNGateways() ([]*ec2.VpnGateway, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	vgws := make([]*ec2.VpnGateway, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		vgws, err = rd.GetClient().requestEC2VPNGateways(vgwFilterKey, rd.ResourceNames[i:stop], vgws)
		if err != nil {
			return vgws, err
		}
	}

	return vgws, nil
}

// Requesting vpn gateways using filters prevents API errors caused by
// requesting non-existent vpn gateways and requesting too many vpn gateways
// in one request
func (c *EC2Client) requestEC2VPNGateways(filterKey string, chunk arn.ResourceNames, vgws []*ec2.VpnGateway) ([]*ec2.VpnGateway, error) {
	params := &ec2.DescribeVpnGatewaysInput{
		Filters: []*ec2.Filter{
			{Name: aws.String(filterKey), Values: chunk.AWSStringSlice()},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := c.DescribeVpnGatewaysWithContext(ctx, params)
	if err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err)
		return vgws, err
	}

	for _, vgw := range resp.VpnGateways {
		if !isDeleting(aws.StringValue(vgw.State)) {
			vgws = append(vgws, vgw)
		}
	}

	return vgws, nil
}

// RequestEC2VPNConnectionsFromVPNGateways requests EC2 vpn connections by vpn
// gateway names from the AWS API
func (rd *EC2VPNGatewayDeleter) RequestEC2VPNConnectionsFromVPNGateways() ([]*ec2.VpnConnection, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 200
	vconns := make([]*ec2.VpnConnection, 0)
	var err error
	// Can only filter in batches of 200
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		vconns, err = rd.GetClient().requestEC2VPNConnections(vgwFilterKey, rd.ResourceNames[i:stop], vconns)
		if err != nil {
			return vconns, err
		}
	}

	return vconns, nil
}

package describe

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/viper"
)

const (
	drCode = "DryRunOperation"
)

func setUpAWSSession() *session.Session {
	return session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(viper.GetString("grafiti.az")),
		},
	))
}

// GetAutoScalingGroupsByNames returns AutoScalingGroups by AutoScalingGroupNames
func GetAutoScalingGroupsByNames(asgNames *[]string) (*[]*autoscaling.Group, error) {
	if asgNames == nil {
		return nil, nil
	}
	svc := autoscaling.New(setUpAWSSession())
	params := &autoscaling.DescribeAutoScalingGroupsInput{}
	if asgNames != nil {
		params.SetAutoScalingGroupNames(aws.StringSlice(*asgNames))
	}
	asgs := make([]*autoscaling.Group, 0)
	for {
		req, resp := svc.DescribeAutoScalingGroupsRequest(params)
		if err := req.Send(); err != nil {
			return nil, err
		}

		for _, asg := range resp.AutoScalingGroups {
			asgs = append(asgs, asg)
		}

		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}
		params.SetNextToken(*resp.NextToken)
	}

	return &asgs, nil
}

// GetAutoScalingLaunchConfigurationsByNames returns launch configurations by
// configuration name
func GetAutoScalingLaunchConfigurationsByNames(lcNames *[]string) (*[]*autoscaling.LaunchConfiguration, error) {
	if lcNames == nil {
		return nil, nil
	}
	svc := autoscaling.New(setUpAWSSession())
	params := &autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: aws.StringSlice(*lcNames),
	}
	asgs := make([]*autoscaling.LaunchConfiguration, 0)
	for {
		req, resp := svc.DescribeLaunchConfigurationsRequest(params)
		if err := req.Send(); err != nil {
			return nil, err
		}

		for _, asg := range resp.LaunchConfigurations {
			asgs = append(asgs, asg)
		}

		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}
		params.SetNextToken(*resp.NextToken)
	}

	return &asgs, nil
}

// GetEC2EIPAddressesByENIIDs retrieves elastic IP addresses from network interfaces
func GetEC2EIPAddressesByENIIDs(eniIDs *[]string) (*[]*ec2.Address, error) {
	if eniIDs == nil {
		return nil, nil
	}

	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeAddressesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("network-interface-id"), Values: aws.StringSlice(*eniIDs)},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeAddressesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return &resp.Addresses, nil
}

// GetIAMInstanceProfilesByLaunchConfigs retrieves instance profiles from launch configuration ID's
func GetIAMInstanceProfilesByLaunchConfigs(lcIDs *[]string) (*[]*iam.InstanceProfile, error) {
	if lcIDs == nil {
		return nil, nil
	}
	svc := iam.New(setUpAWSSession())
	iprs := make([]*iam.InstanceProfile, 0)

	params := &iam.ListInstanceProfilesInput{}
	for {
		ctx := aws.BackgroundContext()
		resp, err := svc.ListInstanceProfilesWithContext(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, ipr := range resp.InstanceProfiles {
			iprs = append(iprs, ipr)
		}

		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}
		params.SetMarker(*resp.Marker)
	}

	return &iprs, nil
}

// GetEC2InstanceReservationsByVPCIDs retrieves instance reservations from vpc ID's
func GetEC2InstanceReservationsByVPCIDs(vpcIDs *[]string) (*[]*ec2.Reservation, error) {
	if vpcIDs == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	irs := make([]*ec2.Reservation, 0)

	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: aws.StringSlice(*vpcIDs)},
		},
	}
	for {
		ctx := aws.BackgroundContext()
		resp, err := svc.DescribeInstancesWithContext(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, r := range resp.Reservations {
			irs = append(irs, r)
		}

		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}
		params.SetNextToken(*resp.NextToken)
	}

	return &irs, nil
}

// GetEC2InternetGatewaysByVPCIDs retrieves internet gateways by vpc ID
func GetEC2InternetGatewaysByVPCIDs(vpcIDs *[]string) (*[]*ec2.InternetGateway, error) {
	if vpcIDs == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeInternetGatewaysInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("attachment.vpc-id"), Values: aws.StringSlice(*vpcIDs)},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeInternetGatewaysWithContext(ctx, params)
	if err != nil {
		return nil, err
	}
	return &resp.InternetGateways, nil
}

// GetEC2InternetGateways retrieves internet gateways by gateway ID
func GetEC2InternetGateways(ids *[]string) (*[]*ec2.InternetGateway, error) {
	if ids == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeInternetGatewaysInput{
		InternetGatewayIds: aws.StringSlice(*ids),
	}

	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeInternetGatewaysWithContext(ctx, params)
	if err != nil {
		return nil, err
	}
	return &resp.InternetGateways, nil
}

// GetEC2InstanceReservations retrieves the states of instances by instance ID's
func GetEC2InstanceReservations(ids *[]string) (*[]*ec2.Reservation, error) {
	if ids == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice(*ids),
	}

	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeInstancesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return &resp.Reservations, nil
}

// GetEC2NetworkInterfacesBySubnetIDs retrieves network interfaces by subnet ID
func GetEC2NetworkInterfacesBySubnetIDs(snIDs *[]string) (*[]*ec2.NetworkInterface, error) {
	if snIDs == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("subnet-id"), Values: aws.StringSlice(*snIDs)},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeNetworkInterfacesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}
	return &resp.NetworkInterfaces, nil
}

// GetEC2NatGatewaysBySubnetIDs retrieves network interfaces by network interface ID
func GetEC2NatGatewaysBySubnetIDs(snIDs *[]string) (*[]*ec2.NatGateway, error) {
	if snIDs == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeNatGatewaysInput{
		Filter: []*ec2.Filter{
			{Name: aws.String("subnet-id"), Values: aws.StringSlice(*snIDs)},
		},
	}

	ngws := make([]*ec2.NatGateway, 0)
	for {
		ctx := aws.BackgroundContext()
		resp, err := svc.DescribeNatGatewaysWithContext(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, ngw := range resp.NatGateways {
			ngws = append(ngws, ngw)
		}

		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}
		params.SetNextToken(*resp.NextToken)
	}

	return &ngws, nil
}

// GetEC2NatGatewaysByVPCIDs retrieves nat gateways by vpc ID
func GetEC2NatGatewaysByVPCIDs(vpcIDs *[]string) (*[]*ec2.NatGateway, error) {
	if vpcIDs == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeNatGatewaysInput{
		Filter: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: aws.StringSlice(*vpcIDs)},
		},
	}

	ngws := make([]*ec2.NatGateway, 0)
	for {
		ctx := aws.BackgroundContext()
		resp, err := svc.DescribeNatGatewaysWithContext(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, ngw := range resp.NatGateways {
			ngws = append(ngws, ngw)
		}

		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}
		params.SetNextToken(*resp.NextToken)
	}

	return &ngws, nil
}

// GetEC2NetworkInterfaces retrieves network interfaces by network interface ID
// NOTE: if eniIDs is passed into the 'NetworkInterfaceId' field and an interface
// with one of those ID's does not exist, DescribeNetworkInterfaces will error.
// The 'Filters' field is used to avoid this issue.
func GetEC2NetworkInterfaces(eniIDs *[]string) (*[]*ec2.NetworkInterface, error) {
	if eniIDs == nil {
		return nil, nil
	}

	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("network-interface-id"), Values: aws.StringSlice(*eniIDs)},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeNetworkInterfacesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}
	return &resp.NetworkInterfaces, nil
}

// GetEC2NetworkACLs retrieves network acl's by network acl ID
func GetEC2NetworkACLs(aclIDs *[]string) (*[]*ec2.NetworkAcl, error) {
	if aclIDs == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeNetworkAclsInput{
		NetworkAclIds: aws.StringSlice(*aclIDs),
	}

	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeNetworkAclsWithContext(ctx, params)
	if err != nil {
		return nil, err
	}
	return &resp.NetworkAcls, nil
}

// GetEC2NetworkACLsBySubnetIDs retrieves network acl's by subnet ID
func GetEC2NetworkACLsBySubnetIDs(snIDs *[]string) (*[]*ec2.NetworkAcl, error) {
	if snIDs == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeNetworkAclsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("association.subnet-id"), Values: aws.StringSlice(*snIDs)},
		},
	}

	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeNetworkAclsWithContext(ctx, params)
	if err != nil {
		return nil, err
	}
	return &resp.NetworkAcls, nil
}

// GetEC2NetworkInterfacesByVPCIDs gathers resources directly tied to a
// NetworkInterface that cannot be found by tags but must be deleted before
// have a wealth of resource information. Describing their associations helps
// construct the dependency graph for deletion.
func GetEC2NetworkInterfacesByVPCIDs(vpcIDs *[]string) (*[]*ec2.NetworkInterface, error) {
	if vpcIDs == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: aws.StringSlice(*vpcIDs)},
		},
	}
	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeNetworkInterfacesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}
	return &resp.NetworkInterfaces, nil
}

// GetEC2RouteTablesByVPCIDs retrieves subnet-routetable associations by vpc ID's
func GetEC2RouteTablesByVPCIDs(vpcIDs *[]string) (*[]*ec2.RouteTable, error) {
	if vpcIDs == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: aws.StringSlice(*vpcIDs)},
		},
	}
	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeRouteTablesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return &resp.RouteTables, nil
}

// GetEC2RouteTables retrieves route tables by routetable ID
func GetEC2RouteTables(rtIDs *[]string) (*[]*ec2.RouteTable, error) {
	if rtIDs == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeRouteTablesInput{
		RouteTableIds: aws.StringSlice(*rtIDs),
	}
	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeRouteTablesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return &resp.RouteTables, nil
}

// GetEC2SecurityGroupsByVPCIDs retrieves securitygroup info by vpc ID
func GetEC2SecurityGroupsByVPCIDs(vpcIDs *[]string) (*[]*ec2.SecurityGroup, error) {
	if vpcIDs == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: aws.StringSlice(*vpcIDs)},
		},
	}
	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeSecurityGroupsWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return &resp.SecurityGroups, nil
}

// GetEC2SecurityGroupReferences retrieves egress rules by securitygroup ID
func GetEC2SecurityGroupReferences(sgIDs *[]string) (*[]*ec2.SecurityGroupReference, error) {
	if sgIDs == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeSecurityGroupReferencesInput{
		GroupId: aws.StringSlice(*sgIDs),
	}
	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeSecurityGroupReferencesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return &resp.SecurityGroupReferenceSet, nil
}

// GetEC2SecurityGroups retrieves securitygroup info by securitygroup ID
func GetEC2SecurityGroups(sgIDs *[]string) (*[]*ec2.SecurityGroup, error) {
	if sgIDs == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeSecurityGroupsInput{
		GroupIds: aws.StringSlice(*sgIDs),
	}
	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeSecurityGroupsWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	filteredSGs := make([]*ec2.SecurityGroup, 0)
	for _, sg := range resp.SecurityGroups {
		if *sg.GroupId != "default" {
			filteredSGs = append(filteredSGs, sg)
		}
	}

	return &filteredSGs, nil
}

// GetEC2SubnetsByVPCIDs retrieves subnets by vpc ID's
func GetEC2SubnetsByVPCIDs(vpcIDs *[]string) (*[]*ec2.Subnet, error) {
	if vpcIDs == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: aws.StringSlice(*vpcIDs)},
		},
	}
	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeSubnetsWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return &resp.Subnets, nil
}

// GetEC2Subnets retrieves subnets by subnet ID
func GetEC2Subnets(snIDs *[]string) (*[]*ec2.Subnet, error) {
	if snIDs == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeSubnetsInput{
		SubnetIds: aws.StringSlice(*snIDs),
	}
	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeSubnetsWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return &resp.Subnets, nil
}

// GetEC2VPCs retrieves vpc info by vpc ID
func GetEC2VPCs(vpcIDs *[]string) (*[]*ec2.Vpc, error) {
	if vpcIDs == nil {
		return nil, nil
	}
	svc := ec2.New(setUpAWSSession())
	params := &ec2.DescribeVpcsInput{
		VpcIds: aws.StringSlice(*vpcIDs),
	}
	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeVpcsWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return &resp.Vpcs, nil
}

// GetIAMInstanceProfilesByIDs retrieves instance profiles by profile ID's
func GetIAMInstanceProfilesByIDs(iprIDs *[]string) (*[]*iam.InstanceProfile, error) {
	if iprIDs == nil {
		return nil, nil
	}
	// We cannot request a filtered list of instance profiles, so we must
	// iterate through all returned profiles and select the ones we want.
	want := make(map[string]bool)
	for _, n := range *iprIDs {
		if _, ok := want[n]; !ok {
			want[n] = true
		}
	}

	svc := iam.New(setUpAWSSession())
	params := &iam.ListInstanceProfilesInput{
		MaxItems: aws.Int64(100),
	}
	iprs := make([]*iam.InstanceProfile, 0)
	for {
		ctx := aws.BackgroundContext()
		resp, lerr := svc.ListInstanceProfilesWithContext(ctx, params)
		if lerr != nil {
			return nil, lerr
		}

		for _, rp := range resp.InstanceProfiles {
			if _, ok := want[*rp.InstanceProfileId]; ok {
				iprs = append(iprs, rp)
			}
		}

		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}
		if resp.Marker != nil && *resp.Marker != "" {
			params.SetMarker(*resp.Marker)
		}

	}
	return &iprs, nil
}

// GetIAMRolePoliciesByRoles finds all RolePolicies defined for a slice of Roles and
// returns a map of RoleName -> RolePolicies
func GetIAMRolePoliciesByRoles(rns *[]string) (*map[string][]string, error) {
	if rns == nil {
		return nil, nil
	}
	svc := iam.New(setUpAWSSession())
	params := &iam.ListRolePoliciesInput{
		MaxItems: aws.Int64(100),
	}
	rpsMap := make(map[string][]string)
	for _, rn := range *rns {
		params.SetRoleName(rn)
		ctx := aws.BackgroundContext()
		resp, lerr := svc.ListRolePoliciesWithContext(ctx, params)
		if lerr != nil {
			return nil, lerr
		}

		for _, rp := range resp.PolicyNames {
			rpsMap[rn] = append(rpsMap[rn], *rp)
		}

		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}
		if resp.Marker != nil && *resp.Marker != "" {
			params.SetMarker(*resp.Marker)
		}

	}
	return &rpsMap, nil
}

// GetRoute53HostedZones retrieves a list of all hosted zones
func GetRoute53HostedZones() (*[]*route53.HostedZone, error) {
	hzs := make([]*route53.HostedZone, 0)
	svc := route53.New(setUpAWSSession())
	params := &route53.ListHostedZonesInput{
		MaxItems: aws.String("100"),
	}
	for {
		ctx := aws.BackgroundContext()
		resp, err := svc.ListHostedZonesWithContext(ctx, params)
		if err != nil {
			return nil, err
		}
		for _, hz := range resp.HostedZones {
			hzs = append(hzs, hz)
		}

		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}

		if resp.NextMarker != nil && *resp.NextMarker != "" {
			params.SetMarker(*resp.NextMarker)
		}
	}
	return &hzs, nil
}

// GetRoute53HostedZonesByIDs retrieves a list of hosted zones by their ID's
func GetRoute53HostedZonesByIDs(hzIDs *[]string) (*[]*route53.HostedZone, error) {
	if hzIDs == nil {
		return nil, nil
	}
	// Only way to filter is iteratively (no query params)
	want := make(map[string]bool)
	for _, id := range *hzIDs {
		if _, ok := want["/hostedzone/"+id]; !ok {
			want["/hostedzone/"+id] = true
		}
	}

	wantedHZs := make([]*route53.HostedZone, 0)
	hzs, _ := GetRoute53HostedZones()

	for _, hz := range *hzs {
		if _, ok := want[*hz.Id]; ok {
			wantedHZs = append(wantedHZs, hz)
		}
	}

	return &wantedHZs, nil
}

// GetRoute53ResourceRecordSetsByHZID retrieves a list of ResourceRecordSets by hz id
func GetRoute53ResourceRecordSetsByHZID(hzID string) (*[]*route53.ResourceRecordSet, error) {
	if hzID == "" {
		return nil, nil
	}
	rrs := make([]*route53.ResourceRecordSet, 0)
	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(hzID),
		MaxItems:     aws.String("100"),
	}
	svc := route53.New(setUpAWSSession())
	for {
		ctx := aws.BackgroundContext()
		resp, err := svc.ListResourceRecordSetsWithContext(ctx, params)
		if err != nil {
			return nil, err
		}
		for _, r := range resp.ResourceRecordSets {
			rrs = append(rrs, r)
		}

		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}
		if resp.NextRecordIdentifier != nil && *resp.NextRecordIdentifier != "" {
			params.SetStartRecordIdentifier(*resp.NextRecordIdentifier)
		}
		if resp.NextRecordType != nil && *resp.NextRecordType != "" {
			params.SetStartRecordType(*resp.NextRecordType)
		}
		if resp.NextRecordName != nil && *resp.NextRecordName != "" {
			params.SetStartRecordName(*resp.NextRecordName)
		}
	}
	return &rrs, nil
}

// GetS3BucketObjects gets objects in an S3 bucket
func GetS3BucketObjects(bktID string) (*[]*s3.Object, error) {
	svc := s3.New(setUpAWSSession())
	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(bktID),
	}
	objs := make([]*s3.Object, 0)
	for {
		req, resp := svc.ListObjectsV2Request(params)
		if err := req.Send(); err != nil {
			return nil, err
		}

		for _, o := range resp.Contents {
			objs = append(objs, o)
		}

		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}
		if resp.ContinuationToken != nil && *resp.ContinuationToken != "" {
			params.SetContinuationToken(*resp.ContinuationToken)
		}
	}

	return &objs, nil
}

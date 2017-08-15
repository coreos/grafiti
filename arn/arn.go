package arn

import (
	"fmt"
	re "regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/tidwall/gjson"
)

// ResourceARN aliases a string type for ARNs
type ResourceARN string

// String casts an ResourceARN to a string
func (a ResourceARN) String() string {
	return string(a)
}

// AWSString converts an ResourceARN to a string pointer
func (a ResourceARN) AWSString() *string {
	return aws.String(a.String())
}

// ToResourceARN converts a string pointer to a ResourceARN
func ToResourceARN(sp *string) ResourceARN {
	return ResourceARN(aws.StringValue(sp))
}

// ResourceARNs aliases string slice for ARNs
type ResourceARNs []ResourceARN

// AWSStringSlice converts a slice of ResourceARNs to a slice of string pointers
func (as ResourceARNs) AWSStringSlice() []*string {
	res := make([]*string, 0, len(as))
	for _, a := range as {
		res = append(res, a.AWSString())
	}
	return res
}

// ResourceType aliases a string type for AWS resource types
type ResourceType string

// String casts a ResourceType to a string
func (r ResourceType) String() string {
	return string(r)
}

// AWSString converts a ResourceType to a string pointer
func (r ResourceType) AWSString() *string {
	return aws.String(r.String())
}

// ToResourceType converts a string pointer to a ResourceType
func ToResourceType(sp *string) ResourceType {
	return ResourceType(aws.StringValue(sp))
}

// ResourceTypes aliases a string slice for AWS resource types
type ResourceTypes []ResourceType

// AWSStringSlice converts a slice of ResourceTypes to a slice of string pointers
func (rs ResourceTypes) AWSStringSlice() []*string {
	res := make([]*string, 0, len(rs))
	for _, r := range rs {
		res = append(res, r.AWSString())
	}
	return res
}

// ResourceName aliases a string type for AWS resource names
type ResourceName string

// String casts a ResourceName to a string
func (r ResourceName) String() string {
	return string(r)
}

// AWSString converts a ResourceName to a string pointer
func (r ResourceName) AWSString() *string {
	return aws.String(r.String())
}

// ToResourceName converts a string pointer to a ResourceName
func ToResourceName(sp *string) ResourceName {
	return ResourceName(aws.StringValue(sp))
}

// ResourceNames aliases a string slice for AWS resource names
type ResourceNames []ResourceName

// AWSStringSlice converts a slice of ResourceNames to a slice of string pointers
func (rs ResourceNames) AWSStringSlice() []*string {
	res := make([]*string, 0, len(rs))
	for _, r := range rs {
		res = append(res, r.AWSString())
	}
	return res
}

const (
	// AutoScalingGroupRType is an AWS ResourceType enum value
	AutoScalingGroupRType = "AWS::AutoScaling::AutoScalingGroup"
	// AutoScalingLaunchConfigurationRType is an AWS ResourceType enum value
	AutoScalingLaunchConfigurationRType = "AWS::AutoScaling::LaunchConfiguration"
	// AutoScalingPolicyRType is an AWS ResourceType enum value
	AutoScalingPolicyRType = "AWS::AutoScaling::ScalingPolicy"
	// AutoScalingScheduledActionRType is an AWS ResourceType enum value
	AutoScalingScheduledActionRType = "AWS::AutoScaling::ScheduledAction"
	// ACMCertificateRType is an AWS ResourceType enum value
	ACMCertificateRType = "AWS::ACM::Certificate"
	// CloudTrailTrailRType is an AWS ResourceType enum value
	CloudTrailTrailRType = "AWS::CloudTrail::Trail"
	// CodePipelinePipelineRType is an AWS ResourceType enum value
	CodePipelinePipelineRType = "AWS::CodePipeline::Pipeline"
	// EC2AMIRType is an AWS ResourceType enum value
	EC2AMIRType = "AWS::EC2::Ami"
	// EC2BundleTaskRType is an AWS ResourceType enum value
	EC2BundleTaskRType = "AWS::EC2::BundleTask"
	// EC2ConversionTaskRType is an AWS ResourceType enum value
	EC2ConversionTaskRType = "AWS::EC2::ConversionTask"
	// EC2CustomerGatewayRType is an AWS ResourceType enum value
	EC2CustomerGatewayRType = "AWS::EC2::CustomerGateway"
	// EC2DHCPOptionsRType is an AWS ResourceType enum value
	EC2DHCPOptionsRType = "AWS::EC2::DHCPOptions"
	// EC2EIPRType is an AWS ResourceType enum value
	EC2EIPRType = "AWS::EC2::EIP"
	// EC2EIPAssociationRType is an AWS ResourceType enum value
	EC2EIPAssociationRType = "AWS::EC2::EIPAssociation"
	// EC2ExportTaskRType is an AWS ResourceType enum value
	EC2ExportTaskRType = "AWS::EC2::ExportTask"
	// EC2FlowLogRType is an AWS ResourceType enum value
	EC2FlowLogRType = "AWS::EC2::FlowLog"
	// EC2HostRType is an AWS ResourceType enum value
	EC2HostRType = "AWS::EC2::Host"
	// EC2ImportTaskRType is an AWS ResourceType enum value
	EC2ImportTaskRType = "AWS::EC2::ImportTask"
	// EC2InstanceRType is an AWS ResourceType enum value
	EC2InstanceRType = "AWS::EC2::Instance"
	// EC2InternetGatewayRType is an AWS ResourceType enum value
	EC2InternetGatewayRType = "AWS::EC2::InternetGateway"
	// EC2InternetGatewayAttachmentRType is an AWS ResourceType enum value
	EC2InternetGatewayAttachmentRType = "AWS::EC2::InternetGatewayAttachment"
	// EC2KeyPairRType is an AWS ResourceType enum value
	EC2KeyPairRType = "AWS::EC2::KeyPair"
	// EC2NatGatewayRType is an AWS ResourceType enum value
	EC2NatGatewayRType = "AWS::EC2::NatGateway"
	// EC2NetworkACLRType is an AWS ResourceType enum value
	EC2NetworkACLRType = "AWS::EC2::NetworkAcl"
	// EC2NetworkACLEntryRType is an AWS ResourceType enum value
	EC2NetworkACLEntryRType = "AWS::EC2::NetworkAclEntry"
	// EC2NetworkInterfaceRType is an AWS ResourceType enum value
	EC2NetworkInterfaceRType = "AWS::EC2::NetworkInterface"
	// EC2NetworkInterfaceAttachmentRType is an AWS ResourceType enum value
	EC2NetworkInterfaceAttachmentRType = "AWS::EC2::NetworkInterfaceAttachment"
	// EC2PlacementGroupRType is an AWS ResourceType enum value
	EC2PlacementGroupRType = "AWS::EC2::PlacementGroup"
	// EC2ReservedInstanceRType is an AWS ResourceType enum value
	EC2ReservedInstanceRType = "AWS::EC2::ReservedInstance"
	// EC2ReservedInstancesListingRType is an AWS ResourceType enum value
	EC2ReservedInstancesListingRType = "AWS::EC2::ReservedInstancesListing"
	// EC2ReservedInstancesModificationRType is an AWS ResourceType enum value
	EC2ReservedInstancesModificationRType = "AWS::EC2::ReservedInstancesModification"
	// EC2RouteTableRType is an AWS ResourceType enum value
	EC2RouteTableRType = "AWS::EC2::RouteTable"
	// EC2RouteTableAssociationRType is a grafiti-specificn AWS ResourceType enum value
	EC2RouteTableAssociationRType = "AWS::EC2::RouteTableAssociation"
	// EC2RouteTableRouteRType is a grafiti-specificn AWS ResourceType enum value
	EC2RouteTableRouteRType = "AWS::EC2::RouteTableRoute"
	// EC2ScheduledInstanceRType is an AWS ResourceType enum value
	EC2ScheduledInstanceRType = "AWS::EC2::ScheduledInstance"
	// EC2SecurityGroupRType is an AWS ResourceType enum value
	EC2SecurityGroupRType = "AWS::EC2::SecurityGroup"
	// EC2SecurityGroupEgressRType is an AWS ResourceType enum value
	EC2SecurityGroupEgressRType = "AWS::EC2::SecurityGroupEgressRule"
	// EC2SecurityGroupIngressRType is an AWS ResourceType enum value
	EC2SecurityGroupIngressRType = "AWS::EC2::SecurityGroupIngressRule"
	// EC2SnapshotRType is an AWS ResourceType enum value
	EC2SnapshotRType = "AWS::EC2::Snapshot"
	// EC2SpotFleetRequestRType is an AWS ResourceType enum value
	EC2SpotFleetRequestRType = "AWS::EC2::SpotFleetRequest"
	// EC2SpotInstanceRequestRType is an AWS ResourceType enum value
	EC2SpotInstanceRequestRType = "AWS::EC2::SpotInstanceRequest"
	// EC2SubnetRType is an AWS ResourceType enum value
	EC2SubnetRType = "AWS::EC2::Subnet"
	// EC2SubnetNetworkACLAssociationRType is an AWS ResourceType enum value
	EC2SubnetNetworkACLAssociationRType = "AWS::EC2::SubnetNetworkAclAssociation"
	// EC2SubnetRouteTableAssociationRType is an AWS ResourceType enum value
	EC2SubnetRouteTableAssociationRType = "AWS::EC2::SubnetRouteTableAssociation"
	// EC2VolumeRType is an AWS ResourceType enum value
	EC2VolumeRType = "AWS::EC2::Volume"
	// EC2VPCRType is an AWS ResourceType enum value
	EC2VPCRType = "AWS::EC2::VPC"
	// EC2VPCCIDRAssociationRType is an AWS ResourceType enum value
	EC2VPCCIDRAssociationRType = "AWS::EC2::VPCCIDRAssociation"
	// EC2VPCEndpointRType is an AWS ResourceType enum value
	EC2VPCEndpointRType = "AWS::EC2::VPCEndpoint"
	// EC2VPCPeeringConnectionRType is an AWS ResourceType enum value
	EC2VPCPeeringConnectionRType = "AWS::EC2::VPCPeeringConnection"
	// EC2VPNConnectionRType is an AWS ResourceType enum value
	EC2VPNConnectionRType = "AWS::EC2::VPNConnection"
	// EC2VPNConnectionRouteRType is an AWS ResourceType enum value
	EC2VPNConnectionRouteRType = "AWS::EC2::VPNConnectionRoute"
	// EC2VPNGatewayRType is an AWS ResourceType enum value
	EC2VPNGatewayRType = "AWS::EC2::VPNGateway"
	// EC2VPNGatewayAttachmentRType is an AWS ResourceType enum value
	EC2VPNGatewayAttachmentRType = "AWS::EC2::VPNGatewayAttachment"
	// ElasticLoadBalancingLoadBalancerRType is an AWS ResourceType enum value
	ElasticLoadBalancingLoadBalancerRType = "AWS::ElasticLoadBalancing::LoadBalancer"
	// IAMAccessKeyRType is an AWS ResourceType enum value
	IAMAccessKeyRType = "AWS::IAM::AccessKey"
	// IAMAccountAliasRType is an AWS ResourceType enum value
	IAMAccountAliasRType = "AWS::IAM::AccountAlias"
	// IAMGroupRType is an AWS ResourceType enum value
	IAMGroupRType = "AWS::IAM::Group"
	// IAMInstanceProfileRType is an AWS ResourceType enum value
	IAMInstanceProfileRType = "AWS::IAM::InstanceProfile"
	// IAMMfaDeviceRType is an AWS ResourceType enum value
	IAMMfaDeviceRType = "AWS::IAM::MfaDevice"
	// IAMOpenIDConnectProviderRType is an AWS ResourceType enum value
	IAMOpenIDConnectProviderRType = "AWS::IAM::OpenIDConnectProvider"
	// IAMPolicyRType is an AWS ResourceType enum value
	IAMPolicyRType = "AWS::IAM::Policy"
	// IAMRoleRType is an AWS ResourceType enum value
	IAMRoleRType = "AWS::IAM::Role"
	// IAMSamlProviderRType is an AWS ResourceType enum value
	IAMSamlProviderRType = "AWS::IAM::SamlProvider"
	// IAMServerCertificateRType is an AWS ResourceType enum value
	IAMServerCertificateRType = "AWS::IAM::ServerCertificate"
	// IAMSigningCertificateRType is an AWS ResourceType enum value
	IAMSigningCertificateRType = "AWS::IAM::SigningCertificate"
	// IAMSSHPublicKeyRType is an AWS ResourceType enum value
	IAMSSHPublicKeyRType = "AWS::IAM::SshPublicKey"
	// IAMUserRType is an AWS ResourceType enum value
	IAMUserRType = "AWS::IAM::User"
	// RedshiftClusterRType is an AWS ResourceType enum value
	RedshiftClusterRType = "AWS::Redshift::Cluster"
	// RedshiftClusterParameterGroupRType is an AWS ResourceType enum value
	RedshiftClusterParameterGroupRType = "AWS::Redshift::ClusterParameterGroup"
	// RedshiftClusterSecurityGroupRType is an AWS ResourceType enum value
	RedshiftClusterSecurityGroupRType = "AWS::Redshift::ClusterSecurityGroup"
	// RedshiftClusterSnapshotRType is an AWS ResourceType enum value
	RedshiftClusterSnapshotRType = "AWS::Redshift::ClusterSnapshot"
	// RedshiftClusterSubnetGroupRType is an AWS ResourceType enum value
	RedshiftClusterSubnetGroupRType = "AWS::Redshift::ClusterSubnetGroup"
	// RedshiftEventSubscriptionRType is an AWS ResourceType enum value
	RedshiftEventSubscriptionRType = "AWS::Redshift::EventSubscription"
	// RedshiftHsmClientCertificateRType is an AWS ResourceType enum value
	RedshiftHsmClientCertificateRType = "AWS::Redshift::HsmClientCertificate"
	// RedshiftHsmConfigurationRType is an AWS ResourceType enum value
	RedshiftHsmConfigurationRType = "AWS::Redshift::HsmConfiguration"
	// RDSDBClusterRType is an AWS ResourceType enum value
	RDSDBClusterRType = "AWS::RDS::DBCluster"
	// RDSDBClusterOptionGroupRType is an AWS ResourceType enum value
	RDSDBClusterOptionGroupRType = "AWS::RDS::DBClusterOptionGroup"
	// RDSDBClusterParameterGroupRType is an AWS ResourceType enum value
	RDSDBClusterParameterGroupRType = "AWS::RDS::DBClusterParameterGroup"
	// RDSDBClusterSnapshotRType is an AWS ResourceType enum value
	RDSDBClusterSnapshotRType = "AWS::RDS::DBClusterSnapshot"
	// RDSDBInstanceRType is an AWS ResourceType enum value
	RDSDBInstanceRType = "AWS::RDS::DBInstance"
	// RDSDBOptionGroupRType is an AWS ResourceType enum value
	RDSDBOptionGroupRType = "AWS::RDS::DBOptionGroup"
	// RDSDBParameterGroupRType is an AWS ResourceType enum value
	RDSDBParameterGroupRType = "AWS::RDS::DBParameterGroup"
	// RDSDBSecurityGroupRType is an AWS ResourceType enum value
	RDSDBSecurityGroupRType = "AWS::RDS::DBSecurityGroup"
	// RDSDBSnapshotRType is an AWS ResourceType enum value
	RDSDBSnapshotRType = "AWS::RDS::DBSnapshot"
	// RDSDBSubnetGroupRType is an AWS ResourceType enum value
	RDSDBSubnetGroupRType = "AWS::RDS::DBSubnetGroup"
	// RDSEventSubscriptionRType is an AWS ResourceType enum value
	RDSEventSubscriptionRType = "AWS::RDS::EventSubscription"
	// RDSReservedDBInstanceRType is an AWS ResourceType enum value
	RDSReservedDBInstanceRType = "AWS::RDS::ReservedDBInstance"
	// Route53ChangeRType is an AWS ResourceType enum value
	Route53ChangeRType = "AWS::Route53::Change"
	// Route53HostedZoneRType is an AWS ResourceType enum value
	Route53HostedZoneRType = "AWS::Route53::HostedZone"
	// Route53ResourceRecordSetRType is an AWS ResourceType enum value
	Route53ResourceRecordSetRType = "AWS::Route53::ResourceRecordSet"
	// S3BucketRType is an AWS ResourceType enum value
	S3BucketRType = "AWS::S3::Bucket"
	// S3ObjectRType is an AWS ResourceType enum value
	S3ObjectRType = "AWS::S3::Object"
)

const (
	// AutoScalingNamespace is an AWS Service Namespace enum value
	AutoScalingNamespace = "autoscaling"
	// ACMNamespace is an AWS Service Namespace enum value
	ACMNamespace = "acm"
	// CloudTrailNamespace is an AWS Service Namespace enum value
	CloudTrailNamespace = "cloudtrail"
	// CodePipelineNamespace is an AWS Service Namespace enum value
	CodePipelineNamespace = "codepipeline"
	// EC2Namespace is an AWS Service Namespace enum value
	EC2Namespace = "ec2"
	// ElasticLoadBalancingNamespace is an AWS Service Namespace enum value
	ElasticLoadBalancingNamespace = "elasticloadbalancing"
	// IAMNamespace is an AWS Service Namespace enum value
	IAMNamespace = "iam"
	// RedshiftNamespace is an AWS Service Namespace enum value
	RedshiftNamespace = "redshift"
	// RDSNamespace is an AWS Service Namespace enum value
	RDSNamespace = "rds"
	// Route53Namespace is an AWS Service Namespace enum value
	Route53Namespace = "route53"
	// S3Namespace is an AWS Service Namespace enum value
	S3Namespace = "s3"
)

// CTUnsupportedResourceTypes holds ResourceTypes of resources from which the
// CloudTrail API does not collect logs
var CTUnsupportedResourceTypes = map[ResourceType]struct{}{
	Route53HostedZoneRType: struct{}{},
}

// RGTAUnsupportedResourceTypes holds ResourceTypes of resources that the
// Resource Group Tagging API does not support
var RGTAUnsupportedResourceTypes = map[ResourceType]struct{}{
	Route53HostedZoneRType: struct{}{},
	AutoScalingGroupRType:  struct{}{},
}

// UntaggableResourceTypes holds ResourceTypes of resources that cannot be tagged
var UntaggableResourceTypes = map[ResourceType]struct{}{
	AutoScalingLaunchConfigurationRType: struct{}{},
	AutoScalingPolicyRType:              struct{}{},
	AutoScalingScheduledActionRType:     struct{}{},
	IAMInstanceProfileRType:             struct{}{},
	IAMUserRType:                        struct{}{},
	IAMRoleRType:                        struct{}{},
	IAMPolicyRType:                      struct{}{},
	IAMGroupRType:                       struct{}{},
	IAMMfaDeviceRType:                   struct{}{},
	IAMAccountAliasRType:                struct{}{},
}

// NamespaceForResource maps ResourceType to an ARN namespace
func NamespaceForResource(t ResourceType) string {
	rt := t.String()
	switch {
	case strings.HasPrefix(rt, "AWS::EC2::"):
		return EC2Namespace
	case strings.HasPrefix(rt, "AWS::AutoScaling::"):
		return AutoScalingNamespace
	case strings.HasPrefix(rt, "AWS::ACM::"):
		return ACMNamespace
	case strings.HasPrefix(rt, "AWS::CloudTrail::"):
		return CloudTrailNamespace
	case strings.HasPrefix(rt, "AWS::CodePipeline::"):
		return CodePipelineNamespace
	case strings.HasPrefix(rt, "AWS::ElasticLoadBalancing::"):
		return ElasticLoadBalancingNamespace
	case strings.HasPrefix(rt, "AWS::IAM::"):
		return IAMNamespace
	case strings.HasPrefix(rt, "AWS::Redshift::"):
		return RedshiftNamespace
	case strings.HasPrefix(rt, "AWS::RDS::"):
		return RDSNamespace
	case strings.HasPrefix(rt, "AWS::Route53::"):
		return Route53Namespace
	case strings.HasPrefix(rt, "AWS::S3::"):
		return S3Namespace
	}
	return ""
}

// HostedZonePrefix is the string prefixing hostedzone IDs when retrieved from
// the AWS API
const HostedZonePrefix = "/hostedzone/"

// SplitHostedZoneID splits a hosted zones' AWS ID, which might be prefixed with
// "/hostedzone/", into the actual ID (the suffix)
func SplitHostedZoneID(hzID string) ResourceName {
	if hzSplit := strings.Split(hzID, HostedZonePrefix); len(hzSplit) == 2 && hzSplit[1] != "" {
		hzID = hzSplit[1]
	}
	return ResourceName(hzID)
}

func getAutoScalingGroupARN(rn ResourceName) (string, error) {
	if rn == "" {
		return "", nil
	}

	svc := autoscaling.New(session.Must(session.NewSession(
		&aws.Config{},
	)))
	params := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: aws.StringSlice([]string{rn.String()}),
	}

	ctx := aws.BackgroundContext()
	resp, err := svc.DescribeAutoScalingGroupsWithContext(ctx, params)
	if err != nil || len(resp.AutoScalingGroups) == 0 {
		return "", err
	}

	return aws.StringValue(resp.AutoScalingGroups[0].AutoScalingGroupARN), nil
}

// MapResourceTypeToARN maps ResourceType to ARN
func MapResourceTypeToARN(rt ResourceType, rn ResourceName, parsedEvents ...gjson.Result) ResourceARN {
	var (
		parsedEvent       gjson.Result
		region, accountID string
	)
	ARNPrefix := fmt.Sprintf("arn:aws:%s", NamespaceForResource(rt))
	if len(parsedEvents) > 0 {
		parsedEvent = parsedEvents[0]
		region = parsedEvent.Get("awsRegion").Str
		accountID = parsedEvent.Get("userIdentity.accountId").Str
		// ARN prefixes lack a region for IAM resources, and lack both region and
		// account number for S3 and Route53 resources.
		switch NamespaceForResource(rt) {
		case IAMNamespace:
			ARNPrefix = fmt.Sprintf("%s::%s", ARNPrefix, accountID)
		case Route53Namespace, S3Namespace:
		default:
			ARNPrefix = fmt.Sprintf("%s:%s:%s", ARNPrefix, region, accountID)
		}
	}

	var arn string

	switch rt {
	case AutoScalingGroupRType:
		// arn:aws:autoscaling:region:account-id:autoScalingGroup:groupid:autoScalingGroupName/groupfriendlyname
		asgARN, err := getAutoScalingGroupARN(rn)
		if err != nil || asgARN == "" {
			return ""
		}
		arn = asgARN
	case AutoScalingLaunchConfigurationRType:
		// arn:aws:autoscaling:region:account-id:launchConfiguration:launchconfigid:launchConfigurationName/launchconfigfriendlyname
		// NOTE: type does not support tagging
	case AutoScalingPolicyRType:
		// arn:aws:autoscaling:region:account-id:scalingPolicy:policyid:autoScalingGroupName/groupfriendlyname:policyname/policyfriendlyname
		// NOTE: type does not support tagging
	case AutoScalingScheduledActionRType:
		// arn:aws:autoscaling:region:account-id:scheduledUpdateGroupAction:scheduleactionid:autoScalingGroupName/autoscalinggroupfriendlyname:scheduledActionName/scheduledactionfriendlyname
		// NOTE: type does not support tagging
	case ACMCertificateRType:
		// arn:aws:acm:region:account-id:certificate/certificate-id
		arn = fmt.Sprintf("%s:certificate/%s", ARNPrefix, rn)
	case CloudTrailTrailRType:
		// arn:aws:cloudtrail:region:account-id:trail/trailname
		arn = parsedEvent.Get("responseElements.trailARN").Str
	case CodePipelinePipelineRType:
		// arn:aws:codepipeline:region:account-id:resource-specifier
		arn = fmt.Sprintf("%s:%s", ARNPrefix, rn)
	case EC2AMIRType:
		// arn:aws:ec2:region::image/image-id
		arn = fmt.Sprintf("arn:aws:ec2:%s::image/%s", region, rn)
	case EC2BundleTaskRType:
	case EC2ConversionTaskRType:
	case EC2CustomerGatewayRType:
		// arn:aws:ec2:region:account-id:customer-gateway/cgw-id
		arn = fmt.Sprintf("%s:customer-gateway/%s", ARNPrefix, rn)
	case EC2DHCPOptionsRType:
		// arn:aws:ec2:region:account-id:dhcp-options/dhcp-options-id
		arn = fmt.Sprintf("%s:dhcp-options/%s", ARNPrefix, rn)
	case EC2EIPRType:
	case EC2EIPAssociationRType:
	case EC2ExportTaskRType:
	case EC2FlowLogRType:
	case EC2HostRType:
		// arn:aws:ec2:region:account-id:dedicated-host/host-id
		arn = fmt.Sprintf("%s:dedicated-host/%s", ARNPrefix, rn)
	case EC2ImportTaskRType:
	case EC2InstanceRType:
		// arn:aws:ec2:region:account-id:instance/instance-id
		arn = fmt.Sprintf("%s:instance/%s", ARNPrefix, rn)
	case EC2InternetGatewayRType:
		// arn:aws:ec2:region:account-id:internet-gateway/igw-id
		arn = fmt.Sprintf("%s:internet-gateway/%s", ARNPrefix, rn)
	case EC2KeyPairRType:
		// arn:aws:ec2:region:account-id:key-pair/key-pair-name
		arn = fmt.Sprintf("%s:key-pair/%s", ARNPrefix, rn)
	case EC2NatGatewayRType:
	case EC2NetworkACLRType:
		// arn:aws:ec2:region:account-id:network-acl/nacl-id
		arn = fmt.Sprintf("%s:network-acl/%s", ARNPrefix, rn)
	case EC2NetworkInterfaceRType:
		// arn:aws:ec2:region:account-id:network-interface/eni-id
		arn = fmt.Sprintf("%s:network-interface/%s", ARNPrefix, rn)
	case EC2NetworkInterfaceAttachmentRType:
	case EC2PlacementGroupRType:
		// arn:aws:ec2:region:account-id:placement-group/placement-group-name
		arn = fmt.Sprintf("%s:placement-group/%s", ARNPrefix, rn)
	case EC2ReservedInstanceRType:
	case EC2ReservedInstancesListingRType:
	case EC2ReservedInstancesModificationRType:
	case EC2RouteTableRType:
		// arn:aws:ec2:region:account-id:route-table/route-table-id
		arn = fmt.Sprintf("%s:route-table/%s", ARNPrefix, rn)
	case EC2ScheduledInstanceRType:
	case EC2SecurityGroupRType:
		// arn:aws:ec2:region:account-id:security-group/security-group-id
		arn = fmt.Sprintf("%s:security-group/%s", ARNPrefix, rn)
	case EC2SnapshotRType:
		// arn:aws:ec2:region:account-id:snapshot/snapshot-id
		arn = fmt.Sprintf("%s:snapshot/%s", ARNPrefix, rn)
	case EC2SpotFleetRequestRType:
	case EC2SpotInstanceRequestRType:
	case EC2SubnetRType:
		// arn:aws:ec2:region:account-id:subnet/subnet-id
		arn = fmt.Sprintf("%s:subnet/%s", ARNPrefix, rn)
	case EC2SubnetNetworkACLAssociationRType:
	case EC2SubnetRouteTableAssociationRType:
	case EC2VolumeRType:
		// arn:aws:ec2:region:account-id:volume/volume-id
		arn = fmt.Sprintf("%s:volume/%s", ARNPrefix, rn)
	case EC2VPCRType:
		// arn:aws:ec2:region:account-id:vpc/vpc-id
		arn = fmt.Sprintf("%s:vpc/%s", ARNPrefix, rn)
	case EC2VPCEndpointRType:
	case EC2VPCPeeringConnectionRType:
		// arn:aws:ec2:region:account-id:vpc-peering-connection/vpc-peering-connection-id
		arn = fmt.Sprintf("%s:vpc-peering-connection/%s", ARNPrefix, rn)
	case EC2VPNConnectionRType:
		// arn:aws:ec2:region:account-id:vpn-connection/vpn-id
		arn = fmt.Sprintf("%s:vpn-connection/%s", ARNPrefix, rn)
	case EC2VPNGatewayRType:
		// arn:aws:ec2:region:account-id:vpn-gateway/vgw-id
		arn = fmt.Sprintf("%s:vpn-gateway/%s", ARNPrefix, rn)
	case ElasticLoadBalancingLoadBalancerRType:
		// arn:aws:elasticloadbalancing:region:account-id:loadbalancer/name
		arn = fmt.Sprintf("%s:loadbalancer/%s", ARNPrefix, rn)
	case IAMAccessKeyRType:
	case IAMAccountAliasRType:
	case IAMGroupRType:
		// arn:aws:iam::account-id:group/group-name
		arn = fmt.Sprintf("%s:group/%s", ARNPrefix, rn)
	case IAMInstanceProfileRType:
		// arn:aws:iam::account-id:instance-profile/instance-profile-name
		// NOTE: type does not support tagging
	case IAMMfaDeviceRType:
		// arn:aws:iam::account-id:mfa/virtual-device-name
		arn = fmt.Sprintf("%s:mfa/%s", ARNPrefix, rn)
	case IAMOpenIDConnectProviderRType:
		// arn:aws:iam::account-id:oidc-provider/provider-name
		arn = fmt.Sprintf("%s:oidc-provider/%s", ARNPrefix, rn)
	case IAMPolicyRType:
		// arn:aws:iam::account-id:policy/policy-name
		// NOTE: type does not support tagging
	case IAMRoleRType:
		// arn:aws:iam::account-id:role/role-name
		// NOTE: type does not support tagging
	case IAMSamlProviderRType:
		// arn:aws:iam::account-id:saml-provider/provider-name
		arn = fmt.Sprintf("%s:saml-provider/%s", ARNPrefix, rn)
	case IAMServerCertificateRType:
		// arn:aws:iam::account-id:server-certificate/certificate-name
		arn = fmt.Sprintf("%s:server-certificate/%s", ARNPrefix, rn)
	case IAMSigningCertificateRType:
	case IAMSSHPublicKeyRType:
	case IAMUserRType:
		// arn:aws:iam::account-id:user/user-name
		// NOTE: type does not support tagging
	case RedshiftClusterRType:
		// arn:aws:redshift:region:account-id:cluster:clustername
		arn = fmt.Sprintf("%s:cluster:%s", ARNPrefix, rn)
	case RedshiftClusterParameterGroupRType:
		// arn:aws:redshift:region:account-id:parametergroup:parametergroupname
		arn = fmt.Sprintf("%s:parametergroup:%s", ARNPrefix, rn)
	case RedshiftClusterSecurityGroupRType:
		// arn:aws:redshift:region:account-id:securitygroup:securitygroupname
		arn = fmt.Sprintf("%s:securitygroup:%s", ARNPrefix, rn)
	case RedshiftClusterSnapshotRType:
		// arn:aws:redshift:region:account-id:snapshot:clustername/snapshotname
	case RedshiftClusterSubnetGroupRType:
		// arn:aws:redshift:region:account-id:subnetgroup:subnetgroupname
		arn = fmt.Sprintf("%s:subnetgroup:%s", ARNPrefix, rn)
	case RedshiftEventSubscriptionRType:
	case RedshiftHsmClientCertificateRType:
	case RedshiftHsmConfigurationRType:
	case RDSDBClusterRType:
		// arn:aws:rds:region:account-id:cluster:db-cluster-name
		arn = fmt.Sprintf("%s:cluster:%s", ARNPrefix, rn)
	case RDSDBClusterOptionGroupRType:
	case RDSDBClusterParameterGroupRType:
		// arn:aws:rds:region:account-id:cluster-pg:cluster-parameter-group-name
		arn = fmt.Sprintf("%s:cluster-pg:%s", ARNPrefix, rn)
	case RDSDBClusterSnapshotRType:
		// arn:aws:rds:region:account-id:cluster-snapshot:cluster-snapshot-name
		arn = fmt.Sprintf("%s:cluster-snapshot:%s", ARNPrefix, rn)
	case RDSDBInstanceRType:
		// arn:aws:rds:region:account-id:db:db-instance-name
		arn = fmt.Sprintf("%s:db:%s", ARNPrefix, rn)
	case RDSDBOptionGroupRType:
		// arn:aws:rds:region:account-id:og:option-group-name
		arn = fmt.Sprintf("%s:og:%s", ARNPrefix, rn)
	case RDSDBParameterGroupRType:
		// arn:aws:rds:region:account-id:pg:parameter-group-name
		arn = fmt.Sprintf("%s:pg:%s", ARNPrefix, rn)
	case RDSDBSecurityGroupRType:
		// arn:aws:rds:region:account-id:secgrp:security-group-name
		arn = fmt.Sprintf("%s:secgrp:%s", ARNPrefix, rn)
	case RDSDBSnapshotRType:
		// arn:aws:rds:region:account-id:snapshot:snapshot-name
		arn = fmt.Sprintf("%s:snapshot:%s", ARNPrefix, rn)
	case RDSDBSubnetGroupRType:
		// arn:aws:rds:region:account-id:subgrp:subnet-group-name
		arn = fmt.Sprintf("%s:subgrp:%s", ARNPrefix, rn)
	case RDSEventSubscriptionRType:
		// arn:aws:rds:region:account-id:es:subscription-name
		arn = fmt.Sprintf("%s:es:%s", ARNPrefix, rn)
	case RDSReservedDBInstanceRType:
	case Route53ChangeRType:
		// arn:aws:route53:::change/changeid
	case Route53HostedZoneRType:
		// arn:aws:route53:::hostedzone/zoneid
		arn = fmt.Sprintf("%s:::hostedzone/%s", ARNPrefix, SplitHostedZoneID(rn.String()))
	case S3BucketRType:
		// arn:aws:s3:::bucket-name
		arn = fmt.Sprintf("%s:::%s", ARNPrefix, rn)
	}
	return ResourceARN(arn)
}

func arnToID(pattern, sfx string) ResourceName {
	id := strings.Split(sfx, pattern)
	if len(id) == 2 {
		return ResourceName(id[1])
	}
	return ""
}

// MapARNToRTypeAndRName maps ARN to ResourceType and an identifying ResourceName
func MapARNToRTypeAndRName(arnStr ResourceARN) (ResourceType, ResourceName) {
	arn := arnStr.String()
	var sfx string
	switch {
	case strings.HasPrefix(arn, "arn:aws:autoscaling:"):
		erASG, err := re.Compile("arn:aws:autoscaling:[^:]+:[^:]+:(.+)")
		if err != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
			break
		}
		m := erASG.FindStringSubmatch(arn)
		if len(m) == 2 {
			sfx = m[1]
		} else {
			break
		}
		switch {
		case strings.HasPrefix(sfx, "autoScalingGroup:"):
			return AutoScalingGroupRType, arnToID("autoScalingGroupName/", sfx)
		case strings.HasPrefix(sfx, "launchConfiguration:"):
			return AutoScalingLaunchConfigurationRType, arnToID("launchConfigurationName/", sfx)
		}

	case strings.HasPrefix(arn, "arn:aws:ec2:"):
		erEC2, err := re.Compile("arn:aws:ec2:[^:]+:(?:[^:]+:)?(.+)")
		if err != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
			break
		}
		m := erEC2.FindStringSubmatch(arn)
		if len(m) == 2 {
			sfx = m[1]
		} else {
			break
		}
		switch {
		case strings.HasPrefix(sfx, "customer-gateway/"):
			return EC2CustomerGatewayRType, arnToID("customer-gateway/", sfx)
		case strings.HasPrefix(sfx, "instance/"):
			return EC2InstanceRType, arnToID("instance/", sfx)
		case strings.HasPrefix(sfx, "internet-gateway/"):
			return EC2InternetGatewayRType, arnToID("internet-gateway/", sfx)
		case strings.HasPrefix(sfx, "network-acl/"):
			return EC2NetworkACLRType, arnToID("network-acl/", sfx)
		case strings.HasPrefix(sfx, "network-interface/"):
			return EC2NetworkInterfaceRType, arnToID("network-interface/", sfx)
		case strings.HasPrefix(sfx, "route-table/"):
			return EC2RouteTableRType, arnToID("route-table/", sfx)
		case strings.HasPrefix(sfx, "security-group/"):
			return EC2SecurityGroupRType, arnToID("security-group/", sfx)
		case strings.HasPrefix(sfx, "subnet/"):
			return EC2SubnetRType, arnToID("subnet/", sfx)
		case strings.HasPrefix(sfx, "volume/"):
			return EC2VolumeRType, arnToID("volume/", sfx)
		case strings.HasPrefix(sfx, "vpc/"):
			return EC2VPCRType, arnToID("vpc/", sfx)
		case strings.HasPrefix(sfx, "vpn-connection/"):
			return EC2VPNConnectionRType, arnToID("vpn-connection/", sfx)
		case strings.HasPrefix(sfx, "vpn-gateway/"):
			return EC2VPNGatewayRType, arnToID("vpn-gateway/", sfx)
		}

	case strings.HasPrefix(arn, "arn:aws:elasticloadbalancing"):
		return ElasticLoadBalancingLoadBalancerRType, arnToID("loadbalancer/", arn)

	case strings.HasPrefix(arn, "arn:aws:iam::"):
		erIAM, err := re.Compile("arn:aws:iam::[^:]+:(.+)")
		if err != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", err.Error())
			break
		}
		m := erIAM.FindStringSubmatch(arn)
		if len(m) == 2 {
			sfx = m[1]
		} else {
			break
		}
		switch {
		case strings.HasPrefix(sfx, "instance-profile/"):
			return IAMInstanceProfileRType, arnToID("instance-profile/", sfx)
		case strings.HasPrefix(sfx, "policy/"):
			return IAMPolicyRType, arnToID("policy/", sfx)
		case strings.HasPrefix(sfx, "role/"):
			return IAMRoleRType, arnToID("role/", sfx)
		case strings.HasPrefix(sfx, "user/"):
			return IAMUserRType, arnToID("user/", sfx)
		}

	case strings.HasPrefix(arn, "arn:aws:route53:::"):
		m := strings.Split(arn, "arn:aws:route53:::")
		if len(m) == 2 {
			sfx = m[1]
		} else {
			break
		}
		switch {
		case strings.HasPrefix(sfx, "hostedzone/"):
			return Route53HostedZoneRType, arnToID("hostedzone/", sfx)
		}

	case strings.HasPrefix(arn, "arn:aws:s3:::"):
		return S3BucketRType, arnToID(":::", arn)
	}
	return "", ""
}

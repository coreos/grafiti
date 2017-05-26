package arn

import (
	"fmt"
	re "regexp"
	"strings"

	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/coreos/grafiti/describe"
	"github.com/tidwall/gjson"
)

const (
	// AutoScalingGroupRType is a ResourceType enum value
	AutoScalingGroupRType = "AWS::AutoScaling::AutoScalingGroup"
	// AutoScalingLaunchConfigurationRType is a ResourceType enum value
	AutoScalingLaunchConfigurationRType = "AWS::AutoScaling::LaunchConfiguration"
	// AutoScalingPolicyRType is a ResourceType enum value
	AutoScalingPolicyRType = "AWS::AutoScaling::ScalingPolicy"
	// AutoScalingScheduledActionRType is a ResourceType enum value
	AutoScalingScheduledActionRType = "AWS::AutoScaling::ScheduledAction"
	// ACMCertificateRType is a ResourceType enum value
	ACMCertificateRType = "AWS::ACM::Certificate"
	// CloudTrailTrailRType is a ResourceType enum value
	CloudTrailTrailRType = "AWS::CloudTrail::Trail"
	// CodePipelinePipelineRType is a ResourceType enum value
	CodePipelinePipelineRType = "AWS::CodePipeline::Pipeline"
	// EC2AmiRType is a ResourceType enum value
	EC2AmiRType = "AWS::EC2::Ami"
	// EC2BundleTaskRType is a ResourceType enum value
	EC2BundleTaskRType = "AWS::EC2::BundleTask"
	// EC2ConversionTaskRType is a ResourceType enum value
	EC2ConversionTaskRType = "AWS::EC2::ConversionTask"
	// EC2CustomerGatewayRType is a ResourceType enum value
	EC2CustomerGatewayRType = "AWS::EC2::CustomerGateway"
	// EC2DHCPOptionsRType is a ResourceType enum value
	EC2DHCPOptionsRType = "AWS::EC2::DHCPOptions"
	// EC2EIPRType is a ResourceType enum value
	EC2EIPRType = "AWS::EC2::EIP"
	// EC2EIPAssociationRType is a ResourceType enum value
	EC2EIPAssociationRType = "AWS::EC2::EIPAssociation"
	// EC2ExportTaskRType is a ResourceType enum value
	EC2ExportTaskRType = "AWS::EC2::ExportTask"
	// EC2FlowLogRType is a ResourceType enum value
	EC2FlowLogRType = "AWS::EC2::FlowLog"
	// EC2HostRType is a ResourceType enum value
	EC2HostRType = "AWS::EC2::Host"
	// EC2ImportTaskRType is a ResourceType enum value
	EC2ImportTaskRType = "AWS::EC2::ImportTask"
	// EC2InstanceRType is a ResourceType enum value
	EC2InstanceRType = "AWS::EC2::Instance"
	// EC2InternetGatewayRType is a ResourceType enum value
	EC2InternetGatewayRType = "AWS::EC2::InternetGateway"
	// EC2KeyPairRType is a ResourceType enum value
	EC2KeyPairRType = "AWS::EC2::KeyPair"
	// EC2NatGatewayRType is a ResourceType enum value
	EC2NatGatewayRType = "AWS::EC2::NatGateway"
	// EC2NetworkACLRType is a ResourceType enum value
	EC2NetworkACLRType = "AWS::EC2::NetworkAcl"
	// EC2NetworkInterfaceRType is a ResourceType enum value
	EC2NetworkInterfaceRType = "AWS::EC2::NetworkInterface"
	// EC2NetworkInterfaceAttachmentRType is a ResourceType enum value
	EC2NetworkInterfaceAttachmentRType = "AWS::EC2::NetworkInterfaceAttachment"
	// EC2PlacementGroupRType is a ResourceType enum value
	EC2PlacementGroupRType = "AWS::EC2::PlacementGroup"
	// EC2ReservedInstanceRType is a ResourceType enum value
	EC2ReservedInstanceRType = "AWS::EC2::ReservedInstance"
	// EC2ReservedInstancesListingRType is a ResourceType enum value
	EC2ReservedInstancesListingRType = "AWS::EC2::ReservedInstancesListing"
	// EC2ReservedInstancesModificationRType is a ResourceType enum value
	EC2ReservedInstancesModificationRType = "AWS::EC2::ReservedInstancesModification"
	// EC2RouteTableRType is a ResourceType enum value
	EC2RouteTableRType = "AWS::EC2::RouteTable"
	// EC2RouteTableAssociationRType is a grafiti-specific ResourceType enum value
	EC2RouteTableAssociationRType = "AWS::EC2::RouteTableAssociation"
	// EC2ScheduledInstanceRType is a ResourceType enum value
	EC2ScheduledInstanceRType = "AWS::EC2::ScheduledInstance"
	// EC2SecurityGroupRType is a ResourceType enum value
	EC2SecurityGroupRType = "AWS::EC2::SecurityGroup"
	// EC2SnapshotRType is a ResourceType enum value
	EC2SnapshotRType = "AWS::EC2::Snapshot"
	// EC2SpotFleetRequestRType is a ResourceType enum value
	EC2SpotFleetRequestRType = "AWS::EC2::SpotFleetRequest"
	// EC2SpotInstanceRequestRType is a ResourceType enum value
	EC2SpotInstanceRequestRType = "AWS::EC2::SpotInstanceRequest"
	// EC2SubnetRType is a ResourceType enum value
	EC2SubnetRType = "AWS::EC2::Subnet"
	// EC2SubnetNetworkACLAssociationRType is a ResourceType enum value
	EC2SubnetNetworkACLAssociationRType = "AWS::EC2::SubnetNetworkAclAssociation"
	// EC2SubnetRouteTableAssociationRType is a ResourceType enum value
	EC2SubnetRouteTableAssociationRType = "AWS::EC2::SubnetRouteTableAssociation"
	// EC2VolumeRType is a ResourceType enum value
	EC2VolumeRType = "AWS::EC2::Volume"
	// EC2VPCRType is a ResourceType enum value
	EC2VPCRType = "AWS::EC2::VPC"
	// EC2VPCEndpointRType is a ResourceType enum value
	EC2VPCEndpointRType = "AWS::EC2::VPCEndpoint"
	// EC2VPCPeeringConnectionRType is a ResourceType enum value
	EC2VPCPeeringConnectionRType = "AWS::EC2::VPCPeeringConnection"
	// EC2VPNConnectionRType is a ResourceType enum value
	EC2VPNConnectionRType = "AWS::EC2::VPNConnection"
	// EC2VPNGatewayRType is a ResourceType enum value
	EC2VPNGatewayRType = "AWS::EC2::VPNGateway"
	// ElasticLoadBalancingLoadBalancerRType is a ResourceType enum value
	ElasticLoadBalancingLoadBalancerRType = "AWS::ElasticLoadBalancing::LoadBalancer"
	// IAMAccessKeyRType is a ResourceType enum value
	IAMAccessKeyRType = "AWS::IAM::AccessKey"
	// IAMAccountAliasRType is a ResourceType enum value
	IAMAccountAliasRType = "AWS::IAM::AccountAlias"
	// IAMGroupRType is a ResourceType enum value
	IAMGroupRType = "AWS::IAM::Group"
	// IAMInstanceProfileRType is a ResourceType enum value
	IAMInstanceProfileRType = "AWS::IAM::InstanceProfile"
	// IAMMfaDeviceRType is a ResourceType enum value
	IAMMfaDeviceRType = "AWS::IAM::MfaDevice"
	// IAMOpenIDConnectProviderRType is a ResourceType enum value
	IAMOpenIDConnectProviderRType = "AWS::IAM::OpenIDConnectProvider"
	// IAMPolicyRType is a ResourceType enum value
	IAMPolicyRType = "AWS::IAM::Policy"
	// IAMRoleRType is a ResourceType enum value
	IAMRoleRType = "AWS::IAM::Role"
	// IAMSamlProviderRType is a ResourceType enum value
	IAMSamlProviderRType = "AWS::IAM::SamlProvider"
	// IAMServerCertificateRType is a ResourceType enum value
	IAMServerCertificateRType = "AWS::IAM::ServerCertificate"
	// IAMSigningCertificateRType is a ResourceType enum value
	IAMSigningCertificateRType = "AWS::IAM::SigningCertificate"
	// IAMSshPublicKeyRType is a ResourceType enum value
	IAMSshPublicKeyRType = "AWS::IAM::SshPublicKey"
	// IAMUserRType is a ResourceType enum value
	IAMUserRType = "AWS::IAM::User"
	// RedshiftClusterRType is a ResourceType enum value
	RedshiftClusterRType = "AWS::Redshift::Cluster"
	// RedshiftClusterParameterGroupRType is a ResourceType enum value
	RedshiftClusterParameterGroupRType = "AWS::Redshift::ClusterParameterGroup"
	// RedshiftClusterSecurityGroupRType is a ResourceType enum value
	RedshiftClusterSecurityGroupRType = "AWS::Redshift::ClusterSecurityGroup"
	// RedshiftClusterSnapshotRType is a ResourceType enum value
	RedshiftClusterSnapshotRType = "AWS::Redshift::ClusterSnapshot"
	// RedshiftClusterSubnetGroupRType is a ResourceType enum value
	RedshiftClusterSubnetGroupRType = "AWS::Redshift::ClusterSubnetGroup"
	// RedshiftEventSubscriptionRType is a ResourceType enum value
	RedshiftEventSubscriptionRType = "AWS::Redshift::EventSubscription"
	// RedshiftHsmClientCertificateRType is a ResourceType enum value
	RedshiftHsmClientCertificateRType = "AWS::Redshift::HsmClientCertificate"
	// RedshiftHsmConfigurationRType is a ResourceType enum value
	RedshiftHsmConfigurationRType = "AWS::Redshift::HsmConfiguration"
	// RDSDBClusterRType is a ResourceType enum value
	RDSDBClusterRType = "AWS::RDS::DBCluster"
	// RDSDBClusterOptionGroupRType is a ResourceType enum value
	RDSDBClusterOptionGroupRType = "AWS::RDS::DBClusterOptionGroup"
	// RDSDBClusterParameterGroupRType is a ResourceType enum value
	RDSDBClusterParameterGroupRType = "AWS::RDS::DBClusterParameterGroup"
	// RDSDBClusterSnapshotRType is a ResourceType enum value
	RDSDBClusterSnapshotRType = "AWS::RDS::DBClusterSnapshot"
	// RDSDBInstanceRType is a ResourceType enum value
	RDSDBInstanceRType = "AWS::RDS::DBInstance"
	// RDSDBOptionGroupRType is a ResourceType enum value
	RDSDBOptionGroupRType = "AWS::RDS::DBOptionGroup"
	// RDSDBParameterGroupRType is a ResourceType enum value
	RDSDBParameterGroupRType = "AWS::RDS::DBParameterGroup"
	// RDSDBSecurityGroupRType is a ResourceType enum value
	RDSDBSecurityGroupRType = "AWS::RDS::DBSecurityGroup"
	// RDSDBSnapshotRType is a ResourceType enum value
	RDSDBSnapshotRType = "AWS::RDS::DBSnapshot"
	// RDSDBSubnetGroupRType is a ResourceType enum value
	RDSDBSubnetGroupRType = "AWS::RDS::DBSubnetGroup"
	// RDSEventSubscriptionRType is a ResourceType enum value
	RDSEventSubscriptionRType = "AWS::RDS::EventSubscription"
	// RDSReservedDBInstanceRType is a ResourceType enum value
	RDSReservedDBInstanceRType = "AWS::RDS::ReservedDBInstance"
	// Route53ChangeRType is a ResourceType enum value
	Route53ChangeRType = "AWS::Route53::Change"
	// Route53HostedZoneRType is a ResourceType enum value
	Route53HostedZoneRType = "AWS::Route53::HostedZone"
	// S3BucketRType is a ResourceType enum value
	S3BucketRType = "AWS::S3::Bucket"
)

// CTUnsupportedResourceTypes holds values for which CloudTrail does not
// collect logs
var CTUnsupportedResourceTypes = map[string]bool{
	Route53HostedZoneRType: true,
}

// RGTAUnsupportedResourceTypes holds values the Resource Group Tagging
// API does not support
var RGTAUnsupportedResourceTypes = map[string]bool{
	Route53HostedZoneRType:              true,
	AutoScalingGroupRType:               true,
	AutoScalingLaunchConfigurationRType: true,
	AutoScalingPolicyRType:              true,
	AutoScalingScheduledActionRType:     true,
}

// NamespaceForResource maps ResourceType to an ARN namespace
func NamespaceForResource(resourceType string) string {
	switch {
	case strings.HasPrefix(resourceType, "AWS::EC2::"):
		return "ec2"
	case strings.HasPrefix(resourceType, "AWS::AutoScaling::"):
		return "autoscaling"
	case strings.HasPrefix(resourceType, "AWS::ACM::"):
		return "acm"
	case strings.HasPrefix(resourceType, "AWS::CloudTrail::"):
		return "cloudtrail"
	case strings.HasPrefix(resourceType, "AWS::CodePipeline::"):
		return "codepipeline"
	case strings.HasPrefix(resourceType, "AWS::ElasticLoadBalancing::"):
		return "elasticloadbalancing"
	case strings.HasPrefix(resourceType, "AWS::IAM::"):
		return "iam"
	case strings.HasPrefix(resourceType, "AWS::Redshift::"):
		return "redshift"
	case strings.HasPrefix(resourceType, "AWS::RDS::"):
		return "rds"
	case strings.HasPrefix(resourceType, "AWS::S3::"):
		return "s3"
	}
	return ""
}

// MapResourceTypeToARN maps ResourceType to ARN
func MapResourceTypeToARN(resource *cloudtrail.Resource, parsedEvent gjson.Result) string {
	region := parsedEvent.Get("awsRegion").Str
	accountID := parsedEvent.Get("userIdentity.accountId").Str
	ARNPrefix := fmt.Sprintf("arn:aws:%s:%s:%s", NamespaceForResource(*resource.ResourceType), region, accountID)
	// ARN prefixes lack a region for IAM resources
	if strings.HasPrefix(*resource.ResourceType, "AWS::IAM::") {
		ARNPrefix = fmt.Sprintf("arn:aws:%s::%s", NamespaceForResource(*resource.ResourceType), accountID)
	}
	switch *resource.ResourceType {
	case AutoScalingGroupRType:
		// arn:aws:autoscaling:region:account-id:autoScalingGroup:groupid:autoScalingGroupName/groupfriendlyname
		asgs, err := describe.GetAutoScalingGroupsByNames(&[]string{*resource.ResourceName})
		if asgs == nil || len(*asgs) == 0 || err != nil {
			return ""
		}
		return *(*asgs)[0].AutoScalingGroupARN
		// break
	case AutoScalingLaunchConfigurationRType:
		// NOTE: will not tag
		// arn:aws:autoscaling:region:account-id:launchConfiguration:launchconfigid:launchConfigurationName/launchconfigfriendlyname
		// lcs, err := describe.GetAutoScalingLaunchConfigurations(&[]string{*resource.ResourceName})
		// if lcs == nil || len(*lcs) == 0 || err != nil {
		// 	return ""
		// }
		// return *(*lcs)[0].LaunchConfigurationARN
		// break
	case AutoScalingPolicyRType:
		// arn:aws:autoscaling:region:account-id:scalingPolicy:policyid:autoScalingGroupName/groupfriendlyname:policyname/policyfriendlyname
		// gid := parsedEvent.Get("responseElements.groupid")
		// gfn := parsedEvent.Get("responseElements.groupfriendlyname")
		// return fmt.Sprintf("%s:autoScalingGroup:%s:autoScalingGroupName/%s", ARNPrefix, gid, gfn)
		break
	case AutoScalingScheduledActionRType:
		// arn:aws:autoscaling:region:account-id:scheduledUpdateGroupAction:scheduleactionid:autoScalingGroupName/autoscalinggroupfriendlyname:scheduledActionName/scheduledactionfriendlyname
		// said := parsedEvent.Get("responseElements.scheduleactionid")
		// asgfn := parsedEvent.Get("responseElements.autoscalinggroupfriendlyname")
		// san := parsedEvent.Get("responseElements.scheduledactionfriendlyname")
		// asString := fmt.Sprintf("%s:scheduledUpdateGroupAction:%s", ARNPrefix, said)
		// return fmt.Sprintf("%s:autoScalingGroupName/%s:scheduledActionName/%s", asString, asgfn, san)
		break
	case ACMCertificateRType:
		// arn:aws:acm:region:account-id:certificate/certificate-id
		return fmt.Sprintf("%s:certificate/%s", ARNPrefix, *resource.ResourceName)
	case CloudTrailTrailRType:
		// arn:aws:cloudtrail:region:account-id:trail/trailname
		return parsedEvent.Get("responseElements.trailARN").String()
	case CodePipelinePipelineRType:
		// arn:aws:codepipeline:region:account-id:resource-specifier
		return fmt.Sprintf("%s:%s", ARNPrefix, *resource.ResourceName)
	case EC2AmiRType:
		// arn:aws:ec2:region::image/image-id
		return fmt.Sprintf("arn:aws:ec2:%s::image/%s", region, *resource.ResourceName)
	case EC2BundleTaskRType:
		// arn:aws:ec2:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2ConversionTaskRType:
		// arn:aws:ec2:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2CustomerGatewayRType:
		// arn:aws:ec2:region:account-id:customer-gateway/cgw-id
		return fmt.Sprintf("%s:customer-gateway/%s", ARNPrefix, *resource.ResourceName)
	case EC2DHCPOptionsRType:
		// arn:aws:ec2:region:account-id:dhcp-options/dhcp-options-id
		return fmt.Sprintf("%s:dhcp-options/%s", ARNPrefix, *resource.ResourceName)
	case EC2EIPRType:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2EIPAssociationRType:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2ExportTaskRType:
		// arn:aws:ec2:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2FlowLogRType:
		// arn:aws:ec2:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2HostRType:
		// arn:aws:ec2:region:account-id:dedicated-host/host-id
		return fmt.Sprintf("%s:dedicated-host/%s", ARNPrefix, *resource.ResourceName)
	case EC2ImportTaskRType:
		// arn:aws:ec2:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2InstanceRType:
		// arn:aws:ec2:region:account-id:instance/instance-id
		return fmt.Sprintf("%s:instance/%s", ARNPrefix, *resource.ResourceName)
	case EC2InternetGatewayRType:
		// arn:aws:ec2:region:account-id:internet-gateway/igw-id
		return fmt.Sprintf("%s:internet-gateway/%s", ARNPrefix, *resource.ResourceName)
	case EC2KeyPairRType:
		// arn:aws:ec2:region:account-id:key-pair/key-pair-name
		return fmt.Sprintf("%s:key-pair/%s", ARNPrefix, *resource.ResourceName)
	case EC2NatGatewayRType:
		// arn:aws:ec2:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2NetworkACLRType:
		// arn:aws:ec2:region:account-id:network-acl/nacl-id
		return fmt.Sprintf("%s:network-acl/%s", ARNPrefix, *resource.ResourceName)
	case EC2NetworkInterfaceRType:
		// arn:aws:ec2:region:account-id:network-interface/eni-id
		return fmt.Sprintf("%s:network-interface/%s", ARNPrefix, *resource.ResourceName)
	case EC2NetworkInterfaceAttachmentRType:
		// arn:aws:ec2:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2PlacementGroupRType:
		// arn:aws:ec2:region:account-id:placement-group/placement-group-name
		return fmt.Sprintf("%s:placement-group/%s", ARNPrefix, *resource.ResourceName)
	case EC2ReservedInstanceRType:
		// arn:aws:ec2:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2ReservedInstancesListingRType:
		// arn:aws:ec2:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2ReservedInstancesModificationRType:
		// arn:aws:ec2:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2RouteTableRType:
		// arn:aws:ec2:region:account-id:route-table/route-table-id
		return fmt.Sprintf("%s:route-table/%s", ARNPrefix, *resource.ResourceName)
	case EC2ScheduledInstanceRType:
		// arn:aws:ec2:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2SecurityGroupRType:
		// arn:aws:ec2:region:account-id:security-group/security-group-id
		return fmt.Sprintf("%s:security-group/%s", ARNPrefix, *resource.ResourceName)
	case EC2SnapshotRType:
		// arn:aws:ec2:region:account-id:snapshot/snapshot-id
		return fmt.Sprintf("%s:snapshot/%s", ARNPrefix, *resource.ResourceName)
	case EC2SpotFleetRequestRType:
		// arn:aws:ec2:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2SpotInstanceRequestRType:
		// arn:aws:ec2:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2SubnetRType:
		// arn:aws:ec2:region:account-id:subnet/subnet-id
		return fmt.Sprintf("%s:subnet/%s", ARNPrefix, *resource.ResourceName)
	case EC2SubnetNetworkACLAssociationRType:
		// arn:aws:ec2:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2SubnetRouteTableAssociationRType:
		// arn:aws:ec2:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2VolumeRType:
		// arn:aws:ec2:region:account-id:volume/volume-id
		return fmt.Sprintf("%s:volume/%s", ARNPrefix, *resource.ResourceName)
	case EC2VPCRType:
		// arn:aws:ec2:region:account-id:vpc/vpc-id
		return fmt.Sprintf("%s:vpc/%s", ARNPrefix, *resource.ResourceName)
	case EC2VPCEndpointRType:
		// arn:aws:ec2:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case EC2VPCPeeringConnectionRType:
		// arn:aws:ec2:region:account-id:vpc-peering-connection/vpc-peering-connection-id
		return fmt.Sprintf("%s:vpc-peering-connection/%s", ARNPrefix, *resource.ResourceName)
	case EC2VPNConnectionRType:
		// arn:aws:ec2:region:account-id:vpn-connection/vpn-id
		return fmt.Sprintf("%s:vpn-connection/%s", ARNPrefix, *resource.ResourceName)
	case EC2VPNGatewayRType:
		// arn:aws:ec2:region:account-id:vpn-gateway/vgw-id
		return fmt.Sprintf("%s:vpn-gateway/%s", ARNPrefix, *resource.ResourceName)
	case ElasticLoadBalancingLoadBalancerRType:
		// arn:aws:elasticloadbalancing:region:account-id:loadbalancer/name
		return fmt.Sprintf("%s:loadbalancer/%s", ARNPrefix, *resource.ResourceName)
	case IAMAccessKeyRType:
		// arn:aws:iam::account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case IAMAccountAliasRType:
		// arn:aws:iam::account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case IAMGroupRType:
		// arn:aws:iam::account-id:group/group-name
		return fmt.Sprintf("%s:group/%s", ARNPrefix, *resource.ResourceName)
	case IAMInstanceProfileRType:
		// NOTE: will not tag
		// arn:aws:iam::account-id:instance-profile/instance-profile-name
		// NOTE: *resource.ResourceName gives non-deterministic results: sometimes
		// an ID, sometimes the full ARN
		// rName := *resource.ResourceName
		// if strings.HasPrefix(rName, "arn:aws:iam::") {
		// 	return rName
		// }
		// return fmt.Sprintf("%s:instance-profile/%s", ARNPrefix, *resource.ResourceName)
		break
	case IAMMfaDeviceRType:
		// arn:aws:iam::account-id:mfa/virtual-device-name
		return fmt.Sprintf("%s:mfa/%s", ARNPrefix, *resource.ResourceName)
	case IAMOpenIDConnectProviderRType:
		// arn:aws:iam::account-id:oidc-provider/provider-name
		return fmt.Sprintf("%s:oidc-provider/%s", ARNPrefix, *resource.ResourceName)
	case IAMPolicyRType:
		// NOTE: will not tag
		// arn:aws:iam::account-id:policy/policy-name
		// return fmt.Sprintf("%s:policy/%s", ARNPrefix, *resource.ResourceName)
		break
	case IAMRoleRType:
		// NOTE: will not tag
		// arn:aws:iam::account-id:role/role-name
		// return fmt.Sprintf("%s:role/%s", ARNPrefix, *resource.ResourceName)
		break
	case IAMSamlProviderRType:
		// arn:aws:iam::account-id:saml-provider/provider-name
		return fmt.Sprintf("%s:saml-provider/%s", ARNPrefix, *resource.ResourceName)
	case IAMServerCertificateRType:
		// arn:aws:iam::account-id:server-certificate/certificate-name
		return fmt.Sprintf("%s:server-certificate/%s", ARNPrefix, *resource.ResourceName)
	case IAMSigningCertificateRType:
		// arn:aws:iam::account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case IAMSshPublicKeyRType:
		// arn:aws:iam::account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case IAMUserRType:
		// arn:aws:iam::account-id:user/user-name
		return fmt.Sprintf("%s:user/%s", ARNPrefix, *resource.ResourceName)
		// return parsedEvent.Get("userIdentity.arn").String()
	case RedshiftClusterRType:
		// arn:aws:redshift:region:account-id:cluster:clustername
		return fmt.Sprintf("%s:cluster:%s", ARNPrefix, *resource.ResourceName)
	case RedshiftClusterParameterGroupRType:
		// arn:aws:redshift:region:account-id:parametergroup:parametergroupname
		return fmt.Sprintf("%s:parametergroup:%s", ARNPrefix, *resource.ResourceName)
	case RedshiftClusterSecurityGroupRType:
		// arn:aws:redshift:region:account-id:securitygroup:securitygroupname
		return fmt.Sprintf("%s:securitygroup:%s", ARNPrefix, *resource.ResourceName)
	case RedshiftClusterSnapshotRType:
		// arn:aws:redshift:region:account-id:snapshot:clustername/snapshotname
		// cn := parsedEvent.Get("responseElements.clustername")
		// ssn := parsedEvent.Get("responseElements.clustername.snapshotname")
		// return fmt.Sprintf("%s:snapshot:%s/%s", ARNPrefix, cn, ssn)
		break
	case RedshiftClusterSubnetGroupRType:
		// arn:aws:redshift:region:account-id:subnetgroup:subnetgroupname
		return fmt.Sprintf("%s:subnetgroup:%s", ARNPrefix, *resource.ResourceName)
	case RedshiftEventSubscriptionRType:
		// arn:aws:redshift:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case RedshiftHsmClientCertificateRType:
		// arn:aws:redshift:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case RedshiftHsmConfigurationRType:
		// arn:aws:redshift:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case RDSDBClusterRType:
		// arn:aws:rds:region:account-id:cluster:db-cluster-name
		return fmt.Sprintf("%s:cluster:%s", ARNPrefix, *resource.ResourceName)
	case RDSDBClusterOptionGroupRType:
		// arn:aws:rds:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case RDSDBClusterParameterGroupRType:
		// arn:aws:rds:region:account-id:cluster-pg:cluster-parameter-group-name
		return fmt.Sprintf("%s:cluster-pg:%s", ARNPrefix, *resource.ResourceName)
	case RDSDBClusterSnapshotRType:
		// arn:aws:rds:region:account-id:cluster-snapshot:cluster-snapshot-name
		return fmt.Sprintf("%s:cluster-snapshot:%s", ARNPrefix, *resource.ResourceName)
	case RDSDBInstanceRType:
		// arn:aws:rds:region:account-id:db:db-instance-name
		return fmt.Sprintf("%s:db:%s", ARNPrefix, *resource.ResourceName)
	case RDSDBOptionGroupRType:
		// arn:aws:rds:region:account-id:og:option-group-name
		return fmt.Sprintf("%s:og:%s", ARNPrefix, *resource.ResourceName)
	case RDSDBParameterGroupRType:
		// arn:aws:rds:region:account-id:pg:parameter-group-name
		return fmt.Sprintf("%s:pg:%s", ARNPrefix, *resource.ResourceName)
	case RDSDBSecurityGroupRType:
		// arn:aws:rds:region:account-id:secgrp:security-group-name
		return fmt.Sprintf("%s:secgrp:%s", ARNPrefix, *resource.ResourceName)
	case RDSDBSnapshotRType:
		// arn:aws:rds:region:account-id:snapshot:snapshot-name
		return fmt.Sprintf("%s:snapshot:%s", ARNPrefix, *resource.ResourceName)
	case RDSDBSubnetGroupRType:
		// arn:aws:rds:region:account-id:subgrp:subnet-group-name
		return fmt.Sprintf("%s:subgrp:%s", ARNPrefix, *resource.ResourceName)
	case RDSEventSubscriptionRType:
		// arn:aws:rds:region:account-id:es:subscription-name
		return fmt.Sprintf("%s:es:%s", ARNPrefix, *resource.ResourceName)
	case RDSReservedDBInstanceRType:
		// arn:aws:rds:region:account-id:
		// return fmt.Sprintf("%s:resource/%s", ARNPrefix, *resource.ResourceName)
		break
	case Route53ChangeRType:
		// arn:aws:route53:::change/changeid
		// cid := parsedEvent.Get("responseElements.changeid")
		// return fmt.Sprintf("arn:aws:route53:::change/%s", cid)
		break
	case Route53HostedZoneRType:
		// arn:aws:route53:::hostedzone/zoneid
		hzSplit := strings.Split(*resource.ResourceName, "/hostedzone/")
		if len(hzSplit) != 2 {
			return ""
		}
		return fmt.Sprintf("arn:aws:route53:::hostedzone/%s", hzSplit[1])
	case S3BucketRType:
		// arn:aws:s3:::bucket-name
		return fmt.Sprintf("arn:aws:s3:::%s", *resource.ResourceName)
	}
	return ""
}

func arnToID(pattern, ARN string) string {
	id := strings.Split(ARN, pattern)
	if len(id) == 2 {
		return id[1]
	}
	return ""
}

// MapARNToRTypeAndRName maps ARN to ResourceType and an identifying ResourceName
func MapARNToRTypeAndRName(ARN string) (string, string) {
	sfx := ""
	switch {
	case strings.HasPrefix(ARN, "arn:aws:autoscaling:"):
		erASG := re.MustCompile("arn:aws:autoscaling:[^:]+:[^:]+:(.+)")
		m := erASG.FindStringSubmatch(ARN)
		if len(m) == 2 {
			sfx = m[1]
		} else {
			return "", ""
		}
		switch {
		case strings.HasPrefix(sfx, "autoScalingGroup:"):
			return AutoScalingGroupRType, arnToID("autoScalingGroupName/", sfx)
		case strings.HasPrefix(sfx, "launchConfiguration:"):
			return AutoScalingLaunchConfigurationRType, arnToID("launchConfigurationName/", sfx)
		// TODO: get asg name and policy name, which are both req'd for deletion
		case strings.HasPrefix(sfx, "scalingPolicy:"):
			return AutoScalingPolicyRType, arnToID("policyname/", sfx)
		case strings.HasPrefix(sfx, "scheduledUpdateGroupAction:"):
			return AutoScalingScheduledActionRType, arnToID("scheduledActionName/", sfx)
		}

	case strings.HasPrefix(ARN, "arn:aws:acm"):
		return ACMCertificateRType, arnToID("certificate/", ARN)

	case strings.HasPrefix(ARN, "arn:aws:cloudtrail"):
		return CloudTrailTrailRType, arnToID("trail/", ARN)

	case strings.HasPrefix(ARN, "arn:aws:ec2:"):
		erEC2 := re.MustCompile("arn:aws:ec2:[^:]+:(?:[^:]+:)?(.+)")
		m := erEC2.FindStringSubmatch(ARN)
		if len(m) == 2 {
			sfx = m[1]
		} else {
			return "", ""
		}
		switch {
		// case strings.HasPrefix(sfx, "image/"):
		// 	return EC2AmiRType, arnToID("image/", sfx)
		case strings.HasPrefix(sfx, "customer-gateway/"):
			return EC2CustomerGatewayRType, arnToID("customer-gateway/", sfx)
		case strings.HasPrefix(sfx, "dhcp-options/"):
			return EC2DHCPOptionsRType, arnToID("dhcp-options/", sfx)
		case strings.HasPrefix(sfx, "dedicated-host/"):
			return EC2HostRType, arnToID("dedicated-host/", sfx)
		case strings.HasPrefix(sfx, "instance/"):
			return EC2InstanceRType, arnToID("instance/", sfx)
		case strings.HasPrefix(sfx, "internet-gateway/"):
			return EC2InternetGatewayRType, arnToID("internet-gateway/", sfx)
		case strings.HasPrefix(sfx, "key-pair/"):
			return EC2KeyPairRType, arnToID("key-pair/", sfx)
		case strings.HasPrefix(sfx, "network-acl/"):
			return EC2NetworkACLRType, arnToID("network-acl/", sfx)
		case strings.HasPrefix(sfx, "network-interface/"):
			return EC2NetworkInterfaceRType, arnToID("network-interface/", sfx)
		case strings.HasPrefix(sfx, "placement-group/"):
			return EC2PlacementGroupRType, arnToID("placement-group/", sfx)
		case strings.HasPrefix(sfx, "route-table/"):
			return EC2RouteTableRType, arnToID("route-table/", sfx)
		case strings.HasPrefix(sfx, "security-group/"):
			return EC2SecurityGroupRType, arnToID("security-group/", sfx)
		case strings.HasPrefix(sfx, "snapshot/"):
			return EC2SnapshotRType, arnToID("snapshot/", sfx)
		case strings.HasPrefix(sfx, "subnet/"):
			return EC2SubnetRType, arnToID("subnet/", sfx)
		case strings.HasPrefix(sfx, "volume/"):
			return EC2VolumeRType, arnToID("volume/", sfx)
		case strings.HasPrefix(sfx, "vpc/"):
			return EC2VPCRType, arnToID("vpc/", sfx)
		case strings.HasPrefix(sfx, "vpc-peering-connection/"):
			return EC2VPCPeeringConnectionRType, arnToID("vpc-peering-connection/", sfx)
		case strings.HasPrefix(sfx, "vpn-connection/"):
			return EC2VPNConnectionRType, arnToID("vpn-connection/", sfx)
		case strings.HasPrefix(sfx, "vpn-gateway/"):
			return EC2VPNGatewayRType, arnToID("vpn-gateway/", sfx)
		}

	case strings.HasPrefix(ARN, "arn:aws:elasticloadbalancing"):
		return ElasticLoadBalancingLoadBalancerRType, arnToID("loadbalancer/", ARN)

	case strings.HasPrefix(ARN, "arn:aws:iam::"):
		erIAM := re.MustCompile("arn:aws:iam::[^:]+:(.+)")
		m := erIAM.FindStringSubmatch(ARN)
		if len(m) == 2 {
			sfx = m[1]
		} else {
			return "", ""
		}
		switch {
		case strings.HasPrefix(sfx, "group/"):
			return IAMGroupRType, arnToID("group/", sfx)
		case strings.HasPrefix(sfx, "instance-profile/"):
			return IAMInstanceProfileRType, arnToID("instance-profile/", sfx)
		case strings.HasPrefix(sfx, "mfa/"):
			return IAMMfaDeviceRType, arnToID("mfa/", sfx)
		case strings.HasPrefix(sfx, "oidc-provider/"):
			return IAMOpenIDConnectProviderRType, arnToID("oidc-provider/", sfx)
		case strings.HasPrefix(sfx, "policy/"):
			return IAMPolicyRType, arnToID("policy/", sfx)
		case strings.HasPrefix(sfx, "role/"):
			return IAMRoleRType, arnToID("role/", sfx)
		case strings.HasPrefix(sfx, "saml-provider/"):
			return IAMSamlProviderRType, arnToID("saml-provider//", sfx)
		case strings.HasPrefix(sfx, "server-certificate/"):
			return IAMServerCertificateRType, arnToID("server-certificate/", sfx)
		case strings.HasPrefix(sfx, "user/"):
			return IAMUserRType, arnToID("user/", sfx)
		}

	case strings.HasPrefix(ARN, "arn:aws:route53:::"):
		m := strings.Split(ARN, "arn:aws:route53:::")
		if len(m) == 2 {
			sfx = m[1]
		} else {
			return "", ""
		}
		switch {
		case strings.HasPrefix(sfx, "change/"):
			return Route53ChangeRType, arnToID("change/", sfx)
		case strings.HasPrefix(sfx, "hostedzone/"):
			return Route53HostedZoneRType, arnToID("hostedzone/", sfx)
		}

	case strings.HasPrefix(ARN, "arn:aws:s3:::"):
		return S3BucketRType, arnToID(":::", ARN)
	}
	return "", ""
}

package arn

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/tidwall/gjson"
)

var testSubsetRTypes = []string{
	AutoScalingGroupRType,
	ACMCertificateRType,
	CloudTrailTrailRType,
	CodePipelinePipelineRType,
	EC2InstanceRType,
	ElasticLoadBalancingLoadBalancerRType,
	IAMGroupRType,
	RedshiftClusterRType,
	RDSDBClusterRType,
	Route53HostedZoneRType,
	S3BucketRType,
}

var testAllNamespaces = []string{
	"autoscaling",
	"acm",
	"cloudtrail",
	"codepipeline",
	"ec2",
	"elasticloadbalancing",
	"iam",
	"redshift",
	"rds",
	"route53",
	"s3",
}

func TestNamespaceForResource(t *testing.T) {
	for i, rt := range testSubsetRTypes {
		tns := NamespaceForResource(rt)
		if testAllNamespaces[i] != tns {
			t.Errorf("NamespaceForResource failed\nwanted: namespace=%s; got: namespace=%s\n", testAllNamespaces[i], tns)
		}
	}
}

var testConstructedARNs = []string{
	"arn:aws:acm:us-east-1:12345678910:certificate/certificate-id",
	"arn:aws:cloudtrail:us-east-1:12345678910:trail/trailname",
	"arn:aws:codepipeline:us-east-1:12345678910:resource-specifier",
	"arn:aws:ec2:us-east-1::image/image-id",
	"arn:aws:ec2:us-east-1:12345678910:customer-gateway/cgw-id",
	"arn:aws:ec2:us-east-1:12345678910:dhcp-options/dhcp-options-id",
	"arn:aws:ec2:us-east-1:12345678910:dedicated-host/host-id",
	"arn:aws:ec2:us-east-1:12345678910:instance/instance-id",
	"arn:aws:ec2:us-east-1:12345678910:internet-gateway/igw-id",
	"arn:aws:ec2:us-east-1:12345678910:key-pair/key-pair-name",
	"arn:aws:ec2:us-east-1:12345678910:network-acl/nacl-id",
	"arn:aws:ec2:us-east-1:12345678910:network-interface/eni-id",
	"arn:aws:ec2:us-east-1:12345678910:placement-group/placement-group-name",
	"arn:aws:ec2:us-east-1:12345678910:route-table/route-table-id",
	"arn:aws:ec2:us-east-1:12345678910:security-group/security-group-id",
	"arn:aws:ec2:us-east-1:12345678910:snapshot/snapshot-id",
	"arn:aws:ec2:us-east-1:12345678910:subnet/subnet-id",
	"arn:aws:ec2:us-east-1:12345678910:volume/volume-id",
	"arn:aws:ec2:us-east-1:12345678910:vpc/vpc-id",
	"arn:aws:ec2:us-east-1:12345678910:vpc-peering-connection/vpc-peering-connection-id",
	"arn:aws:ec2:us-east-1:12345678910:vpn-connection/vpn-id",
	"arn:aws:ec2:us-east-1:12345678910:vpn-gateway/vgw-id",
	"arn:aws:elasticloadbalancing:us-east-1:12345678910:loadbalancer/name",
	"arn:aws:iam::12345678910:group/group-name",
	"arn:aws:iam::12345678910:mfa/virtual-device-name",
	"arn:aws:iam::12345678910:oidc-provider/provider-name",
	"arn:aws:iam::12345678910:saml-provider/provider-name",
	"arn:aws:iam::12345678910:server-certificate/certificate-name",
	"arn:aws:redshift:us-east-1:12345678910:cluster:clustername",
	"arn:aws:redshift:us-east-1:12345678910:parametergroup:parametergroupname",
	"arn:aws:redshift:us-east-1:12345678910:securitygroup:securitygroupname",
	"arn:aws:redshift:us-east-1:12345678910:subnetgroup:subnetgroupname",
	"arn:aws:rds:us-east-1:12345678910:cluster:db-cluster-name",
	"arn:aws:rds:us-east-1:12345678910:cluster-pg:cluster-parameter-group-name",
	"arn:aws:rds:us-east-1:12345678910:cluster-snapshot:cluster-snapshot-name",
	"arn:aws:rds:us-east-1:12345678910:db:db-instance-name",
	"arn:aws:rds:us-east-1:12345678910:og:option-group-name",
	"arn:aws:rds:us-east-1:12345678910:pg:parameter-group-name",
	"arn:aws:rds:us-east-1:12345678910:secgrp:security-group-name",
	"arn:aws:rds:us-east-1:12345678910:snapshot:snapshot-name",
	"arn:aws:rds:us-east-1:12345678910:subgrp:subnet-group-name",
	"arn:aws:rds:us-east-1:12345678910:es:subscription-name",
	"arn:aws:route53:::hostedzone/zoneid",
	"arn:aws:s3:::bucket-name",
}

var testCloudTrailResources = []*cloudtrail.Resource{
	{ResourceType: aws.String(ACMCertificateRType), ResourceName: aws.String("certificate-id")},
	{ResourceType: aws.String(CloudTrailTrailRType), ResourceName: aws.String("trailname")},
	{ResourceType: aws.String(CodePipelinePipelineRType), ResourceName: aws.String("resource-specifier")},
	{ResourceType: aws.String(EC2AMIRType), ResourceName: aws.String("image-id")},
	{ResourceType: aws.String(EC2CustomerGatewayRType), ResourceName: aws.String("cgw-id")},
	{ResourceType: aws.String(EC2DHCPOptionsRType), ResourceName: aws.String("dhcp-options-id")},
	{ResourceType: aws.String(EC2HostRType), ResourceName: aws.String("host-id")},
	{ResourceType: aws.String(EC2InstanceRType), ResourceName: aws.String("instance-id")},
	{ResourceType: aws.String(EC2InternetGatewayRType), ResourceName: aws.String("igw-id")},
	{ResourceType: aws.String(EC2KeyPairRType), ResourceName: aws.String("key-pair-name")},
	{ResourceType: aws.String(EC2NetworkACLRType), ResourceName: aws.String("nacl-id")},
	{ResourceType: aws.String(EC2NetworkInterfaceRType), ResourceName: aws.String("eni-id")},
	{ResourceType: aws.String(EC2PlacementGroupRType), ResourceName: aws.String("placement-group-name")},
	{ResourceType: aws.String(EC2RouteTableRType), ResourceName: aws.String("route-table-id")},
	{ResourceType: aws.String(EC2SecurityGroupRType), ResourceName: aws.String("security-group-id")},
	{ResourceType: aws.String(EC2SnapshotRType), ResourceName: aws.String("snapshot-id")},
	{ResourceType: aws.String(EC2SubnetRType), ResourceName: aws.String("subnet-id")},
	{ResourceType: aws.String(EC2VolumeRType), ResourceName: aws.String("volume-id")},
	{ResourceType: aws.String(EC2VPCRType), ResourceName: aws.String("vpc-id")},
	{ResourceType: aws.String(EC2VPCPeeringConnectionRType), ResourceName: aws.String("vpc-peering-connection-id")},
	{ResourceType: aws.String(EC2VPNConnectionRType), ResourceName: aws.String("vpn-id")},
	{ResourceType: aws.String(EC2VPNGatewayRType), ResourceName: aws.String("vgw-id")},
	{ResourceType: aws.String(ElasticLoadBalancingLoadBalancerRType), ResourceName: aws.String("name")},
	{ResourceType: aws.String(IAMGroupRType), ResourceName: aws.String("group-name")},
	{ResourceType: aws.String(IAMMfaDeviceRType), ResourceName: aws.String("virtual-device-name")},
	{ResourceType: aws.String(IAMOpenIDConnectProviderRType), ResourceName: aws.String("provider-name")},
	{ResourceType: aws.String(IAMSamlProviderRType), ResourceName: aws.String("provider-name")},
	{ResourceType: aws.String(IAMServerCertificateRType), ResourceName: aws.String("certificate-name")},
	{ResourceType: aws.String(RedshiftClusterRType), ResourceName: aws.String("clustername")},
	{ResourceType: aws.String(RedshiftClusterParameterGroupRType), ResourceName: aws.String("parametergroupname")},
	{ResourceType: aws.String(RedshiftClusterSecurityGroupRType), ResourceName: aws.String("securitygroupname")},
	{ResourceType: aws.String(RedshiftClusterSubnetGroupRType), ResourceName: aws.String("subnetgroupname")},
	{ResourceType: aws.String(RDSDBClusterRType), ResourceName: aws.String("db-cluster-name")},
	{ResourceType: aws.String(RDSDBClusterParameterGroupRType), ResourceName: aws.String("cluster-parameter-group-name")},
	{ResourceType: aws.String(RDSDBClusterSnapshotRType), ResourceName: aws.String("cluster-snapshot-name")},
	{ResourceType: aws.String(RDSDBInstanceRType), ResourceName: aws.String("db-instance-name")},
	{ResourceType: aws.String(RDSDBOptionGroupRType), ResourceName: aws.String("option-group-name")},
	{ResourceType: aws.String(RDSDBParameterGroupRType), ResourceName: aws.String("parameter-group-name")},
	{ResourceType: aws.String(RDSDBSecurityGroupRType), ResourceName: aws.String("security-group-name")},
	{ResourceType: aws.String(RDSDBSnapshotRType), ResourceName: aws.String("snapshot-name")},
	{ResourceType: aws.String(RDSDBSubnetGroupRType), ResourceName: aws.String("subnet-group-name")},
	{ResourceType: aws.String(RDSEventSubscriptionRType), ResourceName: aws.String("subscription-name")},
	{ResourceType: aws.String(Route53HostedZoneRType), ResourceName: aws.String("/hostedzone/zoneid")},
	{ResourceType: aws.String(S3BucketRType), ResourceName: aws.String("bucket-name")},
}

func TestMapResourceTypeToARN(t *testing.T) {
	jsonToParse := `{"awsRegion": "us-east-1",
	"userIdentity":{"accountId":"12345678910"},
	"responseElements":{"trailARN":"arn:aws:cloudtrail:us-east-1:12345678910:trail/trailname"}
	}`
	parsedEvent := gjson.Parse(jsonToParse)

	for i, arn := range testConstructedARNs {
		pARN := MapResourceTypeToARN(testCloudTrailResources[i], parsedEvent)
		if arn != pARN {
			t.Errorf("MapResourceTypeToARN failed\nwanted: arn=%s; got: arn=%s\n", arn, pARN)
		}
	}
}

type patternDestructResult struct {
	Pattern string
	ARN     string
	Result  string
}

func testarnToID(t *testing.T) {
	patternMap := []patternDestructResult{
		{":::", "arn:aws:s3:::bucket-name", "bucket-name"},
		{"instance/", "arn:aws:ec2:us-east-1:12345678910:instance/instance-id", "instance-id"},
		{"group/", "arn:aws:iam::12345678910:group/group-id", "group-id"},
	}
	for _, m := range patternMap {
		res := arnToID(m.Pattern, m.ARN)
		if m.Result != res {
			t.Errorf("arnToID failed\nwanted: name=%s; got: name=%s\n", m.Result, res)
		}
	}
}

var testDestructedARNs = []string{
	"arn:aws:autoscaling:us-east-1:12345678910:autoScalingGroup:groupid:autoScalingGroupName/groupfriendlyname",
	"arn:aws:autoscaling:us-east-1:12345678910:launchConfiguration:launchconfigid:launchConfigurationName/launchconfigfriendlyname",
	"arn:aws:autoscaling:us-east-1:12345678910:scalingPolicy:policyid:autoScalingGroupName/groupfriendlyname:policyname/policyfriendlyname",
	"arn:aws:autoscaling:us-east-1:12345678910:scheduledUpdateGroupAction:scheduleactionid:autoScalingGroupName/autoscalinggroupfriendlyname:scheduledActionName/scheduledactionfriendlyname",
	"arn:aws:acm:us-east-1:12345678910:certificate/certificate-id",
	"arn:aws:cloudtrail:us-east-1:12345678910:trail/trailname",
	"arn:aws:ec2:us-east-1:12345678910:customer-gateway/cgw-id",
	"arn:aws:ec2:us-east-1:12345678910:dhcp-options/dhcp-options-id",
	"arn:aws:ec2:us-east-1:12345678910:dedicated-host/host-id",
	"arn:aws:ec2:us-east-1:12345678910:instance/instance-id",
	"arn:aws:ec2:us-east-1:12345678910:internet-gateway/igw-id",
	"arn:aws:ec2:us-east-1:12345678910:key-pair/key-pair-name",
	"arn:aws:ec2:us-east-1:12345678910:network-acl/nacl-id",
	"arn:aws:ec2:us-east-1:12345678910:network-interface/eni-id",
	"arn:aws:ec2:us-east-1:12345678910:placement-group/placement-group-name",
	"arn:aws:ec2:us-east-1:12345678910:route-table/route-table-id",
	"arn:aws:ec2:us-east-1:12345678910:security-group/security-group-id",
	"arn:aws:ec2:us-east-1:12345678910:snapshot/snapshot-id",
	"arn:aws:ec2:us-east-1:12345678910:subnet/subnet-id",
	"arn:aws:ec2:us-east-1:12345678910:volume/volume-id",
	"arn:aws:ec2:us-east-1:12345678910:vpc/vpc-id",
	"arn:aws:ec2:us-east-1:12345678910:vpc-peering-connection/vpc-peering-connection-id",
	"arn:aws:ec2:us-east-1:12345678910:vpn-connection/vpn-id",
	"arn:aws:ec2:us-east-1:12345678910:vpn-gateway/vgw-id",
	"arn:aws:elasticloadbalancing:us-east-1:12345678910:loadbalancer/name",
	"arn:aws:iam::12345678910:group/group-name",
	"arn:aws:iam::12345678910:instance-profile/instance-profile-name",
	"arn:aws:iam::12345678910:mfa/virtual-device-name",
	"arn:aws:iam::12345678910:oidc-provider/provider-name",
	"arn:aws:iam::12345678910:policy/policy-name",
	"arn:aws:iam::12345678910:role/role-name",
	"arn:aws:iam::12345678910:saml-provider/provider-name",
	"arn:aws:iam::12345678910:server-certificate/certificate-name",
	"arn:aws:iam::12345678910:user/user-name",
	"arn:aws:route53:::change/changeid",
	"arn:aws:route53:::hostedzone/zoneid",
	"arn:aws:s3:::bucket-name",
}

// Want order of values maintained
type testRTypeRNameMap struct {
	rType string
	rName string
}

var testARNResults = []testRTypeRNameMap{
	{AutoScalingGroupRType, "groupfriendlyname"},
	{AutoScalingLaunchConfigurationRType, "launchconfigfriendlyname"},
	{AutoScalingPolicyRType, "policyfriendlyname"},
	{AutoScalingScheduledActionRType, "scheduledactionfriendlyname"},
	{ACMCertificateRType, "certificate-id"},
	{CloudTrailTrailRType, "trailname"},
	{EC2CustomerGatewayRType, "cgw-id"},
	{EC2DHCPOptionsRType, "dhcp-options-id"},
	{EC2HostRType, "host-id"},
	{EC2InstanceRType, "instance-id"},
	{EC2InternetGatewayRType, "igw-id"},
	{EC2KeyPairRType, "key-pair-name"},
	{EC2NetworkACLRType, "nacl-id"},
	{EC2NetworkInterfaceRType, "eni-id"},
	{EC2PlacementGroupRType, "placement-group-name"},
	{EC2RouteTableRType, "route-table-id"},
	{EC2SecurityGroupRType, "security-group-id"},
	{EC2SnapshotRType, "snapshot-id"},
	{EC2SubnetRType, "subnet-id"},
	{EC2VolumeRType, "volume-id"},
	{EC2VPCRType, "vpc-id"},
	{EC2VPCPeeringConnectionRType, "vpc-peering-connection-id"},
	{EC2VPNConnectionRType, "vpn-id"},
	{EC2VPNGatewayRType, "vgw-id"},
	{ElasticLoadBalancingLoadBalancerRType, "name"},
	{IAMGroupRType, "group-name"},
	{IAMInstanceProfileRType, "instance-profile-name"},
	{IAMMfaDeviceRType, "virtual-device-name"},
	{IAMOpenIDConnectProviderRType, "provider-name"},
	{IAMPolicyRType, "policy-name"},
	{IAMRoleRType, "role-name"},
	{IAMSamlProviderRType, "provider-name"},
	{IAMServerCertificateRType, "certificate-name"},
	{IAMUserRType, "user-name"},
	{Route53ChangeRType, "changeid"},
	{Route53HostedZoneRType, "zoneid"},
	{S3BucketRType, "bucket-name"},
}

func TestMapARNToRTypeAndRName(t *testing.T) {
	for i, tar := range testARNResults {
		rType, rName := MapARNToRTypeAndRName(testDestructedARNs[i])
		if rType != tar.rType || rName != rName {
			t.Errorf("MapARNToRTypeAndRName failed:\nwanted: rType=%s, rName=%s; got: rType=%s, rName=%s\n", tar.rType, tar.rName, rType, rName)
		}
	}
}

func TestMapARNToRTypeAndRNameMalformedARN(t *testing.T) {
	testMalformedARNsInput := []string{
		"arn:aws:ec2:us-east-1:12345678910:instance/",
		"arn:aws:ec2:us-east-1::instance/",
		"arn:aws:ec2:us-east-1:12345678910:/",
		"arn:aws:route53::us-east-1:hostedzone/zoneid",
		"arn:aws:s3::bucket-name",
	}

	for _, ma := range testMalformedARNsInput {
		_, rName := MapARNToRTypeAndRName(ma)
		if rName != "" {
			t.Errorf("MapARNToRTypeAndRName failed:\nwanted: rName=\"\"; got: rName=%s\n", rName)
		}
	}
}

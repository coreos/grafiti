package arn

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/tidwall/gjson"
)

func TestNamespaceForResource(t *testing.T) {
	cases := []struct {
		Input    ResourceType
		Expected string
	}{
		{AutoScalingGroupRType, AutoScalingNamespace},
		{ACMCertificateRType, ACMNamespace},
		{CloudTrailTrailRType, CloudTrailNamespace},
		{CodePipelinePipelineRType, CodePipelineNamespace},
		{EC2InstanceRType, EC2Namespace},
		{ElasticLoadBalancingLoadBalancerRType, ElasticLoadBalancingNamespace},
		{IAMInstanceProfileRType, IAMNamespace},
		{RedshiftClusterRType, RedshiftNamespace},
		{RDSDBClusterRType, RDSNamespace},
		{Route53HostedZoneRType, Route53Namespace},
		{S3BucketRType, S3Namespace},
	}

	for _, c := range cases {
		ns := NamespaceForResource(c.Input)
		if c.Expected != ns {
			t.Errorf("NamespaceForResource failed\nwanted %s\ngot %s\n", c.Expected, ns)
		}
	}
}

var testCloudTrailResources = []*cloudtrail.Resource{}

func TestMapResourceTypeToARN(t *testing.T) {
	cases := []struct {
		Expected  ResourceARN
		InputType ResourceType
		InputName ResourceName
	}{
		{
			Expected:  "arn:aws:acm:us-east-1:12345678910:certificate/certificate-id",
			InputType: ACMCertificateRType,
			InputName: "certificate-id",
		},
		{
			Expected:  "arn:aws:cloudtrail:us-east-1:12345678910:trail/trailname",
			InputType: CloudTrailTrailRType,
			InputName: "trailname",
		},
		{
			Expected:  "arn:aws:codepipeline:us-east-1:12345678910:resource-specifier",
			InputType: CodePipelinePipelineRType,
			InputName: "resource-specifier",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1::image/image-id",
			InputType: EC2AMIRType,
			InputName: "image-id",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:customer-gateway/cgw-id",
			InputType: EC2CustomerGatewayRType,
			InputName: "cgw-id",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:dhcp-options/dhcp-options-id",
			InputType: EC2DHCPOptionsRType,
			InputName: "dhcp-options-id",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:dedicated-host/host-id",
			InputType: EC2HostRType,
			InputName: "host-id",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:instance/instance-id",
			InputType: EC2InstanceRType,
			InputName: "instance-id",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:internet-gateway/igw-id",
			InputType: EC2InternetGatewayRType,
			InputName: "igw-id",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:key-pair/key-pair-name",
			InputType: EC2KeyPairRType,
			InputName: "key-pair-name",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:network-acl/nacl-id",
			InputType: EC2NetworkACLRType,
			InputName: "nacl-id",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:network-interface/eni-id",
			InputType: EC2NetworkInterfaceRType,
			InputName: "eni-id",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:placement-group/placement-group-name",
			InputType: EC2PlacementGroupRType,
			InputName: "placement-group-name",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:route-table/route-table-id",
			InputType: EC2RouteTableRType,
			InputName: "route-table-id",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:security-group/security-group-id",
			InputType: EC2SecurityGroupRType,
			InputName: "security-group-id",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:snapshot/snapshot-id",
			InputType: EC2SnapshotRType,
			InputName: "snapshot-id",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:subnet/subnet-id",
			InputType: EC2SubnetRType,
			InputName: "subnet-id",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:volume/volume-id",
			InputType: EC2VolumeRType,
			InputName: "volume-id",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:vpc/vpc-id",
			InputType: EC2VPCRType,
			InputName: "vpc-id",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:vpc-peering-connection/vpc-peering-connection-id",
			InputType: EC2VPCPeeringConnectionRType,
			InputName: "vpc-peering-connection-id",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:vpn-connection/vpn-id",
			InputType: EC2VPNConnectionRType,
			InputName: "vpn-id",
		},
		{
			Expected:  "arn:aws:ec2:us-east-1:12345678910:vpn-gateway/vgw-id",
			InputType: EC2VPNGatewayRType,
			InputName: "vgw-id",
		},
		{
			Expected:  "arn:aws:elasticloadbalancing:us-east-1:12345678910:loadbalancer/name",
			InputType: ElasticLoadBalancingLoadBalancerRType,
			InputName: "name",
		},
		{
			Expected:  "arn:aws:iam::12345678910:group/group-name",
			InputType: IAMGroupRType,
			InputName: "group-name",
		},
		{
			Expected:  "arn:aws:iam::12345678910:mfa/virtual-device-name",
			InputType: IAMMfaDeviceRType,
			InputName: "virtual-device-name",
		},
		{
			Expected:  "arn:aws:iam::12345678910:oidc-provider/provider-name",
			InputType: IAMOpenIDConnectProviderRType,
			InputName: "provider-name",
		},
		{
			Expected:  "arn:aws:iam::12345678910:saml-provider/provider-name",
			InputType: IAMSamlProviderRType,
			InputName: "provider-name",
		},
		{
			Expected:  "arn:aws:iam::12345678910:server-certificate/certificate-name",
			InputType: IAMServerCertificateRType,
			InputName: "certificate-name",
		},
		{
			Expected:  "arn:aws:redshift:us-east-1:12345678910:cluster:clustername",
			InputType: RedshiftClusterRType,
			InputName: "clustername",
		},
		{
			Expected:  "arn:aws:redshift:us-east-1:12345678910:parametergroup:parametergroupname",
			InputType: RedshiftClusterParameterGroupRType,
			InputName: "parametergroupname",
		},
		{
			Expected:  "arn:aws:redshift:us-east-1:12345678910:securitygroup:securitygroupname",
			InputType: RedshiftClusterSecurityGroupRType,
			InputName: "securitygroupname",
		},
		{
			Expected:  "arn:aws:redshift:us-east-1:12345678910:subnetgroup:subnetgroupname",
			InputType: RedshiftClusterSubnetGroupRType,
			InputName: "subnetgroupname",
		},
		{
			Expected:  "arn:aws:rds:us-east-1:12345678910:cluster:db-cluster-name",
			InputType: RDSDBClusterRType,
			InputName: "db-cluster-name",
		},
		{
			Expected:  "arn:aws:rds:us-east-1:12345678910:cluster-pg:cluster-parameter-group-name",
			InputType: RDSDBClusterParameterGroupRType,
			InputName: "cluster-parameter-group-name",
		},
		{
			Expected:  "arn:aws:rds:us-east-1:12345678910:cluster-snapshot:cluster-snapshot-name",
			InputType: RDSDBClusterSnapshotRType,
			InputName: "cluster-snapshot-name",
		},
		{
			Expected:  "arn:aws:rds:us-east-1:12345678910:db:db-instance-name",
			InputType: RDSDBInstanceRType,
			InputName: "db-instance-name",
		},
		{
			Expected:  "arn:aws:rds:us-east-1:12345678910:og:option-group-name",
			InputType: RDSDBOptionGroupRType,
			InputName: "option-group-name",
		},
		{
			Expected:  "arn:aws:rds:us-east-1:12345678910:pg:parameter-group-name",
			InputType: RDSDBParameterGroupRType,
			InputName: "parameter-group-name",
		},
		{
			Expected:  "arn:aws:rds:us-east-1:12345678910:secgrp:security-group-name",
			InputType: RDSDBSecurityGroupRType,
			InputName: "security-group-name",
		},
		{
			Expected:  "arn:aws:rds:us-east-1:12345678910:snapshot:snapshot-name",
			InputType: RDSDBSnapshotRType,
			InputName: "snapshot-name",
		},
		{
			Expected:  "arn:aws:rds:us-east-1:12345678910:subgrp:subnet-group-name",
			InputType: RDSDBSubnetGroupRType,
			InputName: "subnet-group-name",
		},
		{
			Expected:  "arn:aws:rds:us-east-1:12345678910:es:subscription-name",
			InputType: RDSEventSubscriptionRType,
			InputName: "subscription-name",
		},
		{
			Expected:  "arn:aws:route53:::hostedzone/zoneid",
			InputType: Route53HostedZoneRType,
			InputName: "/hostedzone/zoneid",
		},
		{
			Expected:  "arn:aws:s3:::bucket-name",
			InputType: S3BucketRType,
			InputName: "bucket-name",
		},
	}

	jsonToParse := `{"awsRegion": "us-east-1",
	"userIdentity":{"accountId":"12345678910"},
	"responseElements":{"trailARN":"arn:aws:cloudtrail:us-east-1:12345678910:trail/trailname"}
	}`
	parsedEvent := gjson.Parse(jsonToParse)

	for _, c := range cases {
		arn := MapResourceTypeToARN(c.InputType, c.InputName, parsedEvent)
		if c.Expected != arn {
			t.Errorf("MapResourceTypeToARN failed\nwanted %s\n got %s\n", c.Expected, arn)
		}
	}
}

func TestARNToID(t *testing.T) {
	cases := []struct {
		InputPattern string
		InputARN     string
		Expected     ResourceName
	}{
		{":::", "arn:aws:s3:::bucket-name", "bucket-name"},
		{"instance/", "arn:aws:ec2:us-east-1:12345678910:instance/instance-id", "instance-id"},
		{"group/", "arn:aws:iam::12345678910:group/group-id", "group-id"},
	}
	for _, c := range cases {
		res := arnToID(c.InputPattern, c.InputARN)
		if c.Expected != res {
			t.Errorf("arnToID failed\nwanted %s\n got %s\n", c.Expected, res)
		}
	}
}

func TestMapARNToRTypeAndRName(t *testing.T) {
	cases := []struct {
		InputARN     ResourceARN
		ExpectedType ResourceType
		ExpectedName ResourceName
	}{
		{
			"arn:aws:autoscaling:us-east-1:12345678910:autoScalingGroup:groupid:autoScalingGroupName/groupfriendlyname",
			AutoScalingGroupRType,
			"groupfriendlyname",
		},
		{
			"arn:aws:autoscaling:us-east-1:12345678910:launchConfiguration:launchconfigid:launchConfigurationName/launchconfigfriendlyname",
			AutoScalingLaunchConfigurationRType,
			"launchconfigfriendlyname",
		},
		{
			"arn:aws:ec2:us-east-1:12345678910:customer-gateway/cgw-id",
			EC2CustomerGatewayRType,
			"cgw-id",
		},
		{
			"arn:aws:ec2:us-east-1:12345678910:instance/instance-id",
			EC2InstanceRType,
			"instance-id",
		},
		{
			"arn:aws:ec2:us-east-1:12345678910:internet-gateway/igw-id",
			EC2InternetGatewayRType,
			"igw-id",
		},
		{
			"arn:aws:ec2:us-east-1:12345678910:network-acl/nacl-id",
			EC2NetworkACLRType,
			"nacl-id",
		},
		{
			"arn:aws:ec2:us-east-1:12345678910:network-interface/eni-id",
			EC2NetworkInterfaceRType,
			"eni-id",
		},
		{
			"arn:aws:ec2:us-east-1:12345678910:route-table/route-table-id",
			EC2RouteTableRType,
			"route-table-id",
		},
		{
			"arn:aws:ec2:us-east-1:12345678910:security-group/security-group-id",
			EC2SecurityGroupRType,
			"security-group-id",
		},
		{
			"arn:aws:ec2:us-east-1:12345678910:subnet/subnet-id",
			EC2SubnetRType,
			"subnet-id",
		},
		{
			"arn:aws:ec2:us-east-1:12345678910:vpc/vpc-id",
			EC2VPCRType,
			"vpc-id",
		},
		{
			"arn:aws:ec2:us-east-1:12345678910:vpn-connection/vpn-id",
			EC2VPNConnectionRType,
			"vpn-id",
		},
		{
			"arn:aws:ec2:us-east-1:12345678910:vpn-gateway/vgw-id",
			EC2VPNGatewayRType,
			"vgw-id",
		},
		{
			"arn:aws:elasticloadbalancing:us-east-1:12345678910:loadbalancer/elb-name",
			ElasticLoadBalancingLoadBalancerRType,
			"elb-name",
		},
		{
			"arn:aws:iam::12345678910:instance-profile/instance-profile-name",
			IAMInstanceProfileRType,
			"instance-profile-name",
		},
		{
			"arn:aws:iam::12345678910:policy/policy-name",
			IAMPolicyRType,
			"policy-name",
		},
		{
			"arn:aws:iam::12345678910:role/role-name",
			IAMRoleRType,
			"role-name",
		},
		{
			"arn:aws:iam::12345678910:user/user-name",
			IAMUserRType,
			"user-name",
		},
		{
			"arn:aws:route53:::hostedzone/zoneid",
			Route53HostedZoneRType,
			"zoneid",
		},
		{
			"arn:aws:s3:::bucket-name",
			S3BucketRType,
			"bucket-name",
		},
	}

	for _, c := range cases {
		rType, rName := MapARNToRTypeAndRName(c.InputARN)
		if rType != c.ExpectedType || rName != c.ExpectedName {
			t.Errorf("MapARNToRTypeAndRName failed\nwanted rType=%s, rName=%s\ngot rType=%s, rName=%s\n", c.ExpectedType, c.ExpectedName, rType, rName)
		}
	}
}

func TestMapARNToRTypeAndRNameMalformedARN(t *testing.T) {
	testMalformedARNsInput := ResourceARNs{
		"arn:aws:ec2:us-east-1:12345678910:instance/",
		"arn:aws:ec2:us-east-1::instance/",
		"arn:aws:ec2:us-east-1:12345678910:/",
		"arn:aws:route53::us-east-1:hostedzone/zoneid",
		"arn:aws:s3::bucket-name",
	}

	for _, ma := range testMalformedARNsInput {
		_, rName := MapARNToRTypeAndRName(ma)
		if rName != "" {
			t.Errorf("MapARNToRTypeAndRName failed\nwanted \"\"\ngot \"%s\"\n", rName)
		}
	}
}

package graph

import (
	"github.com/coreos/grafiti/arn"
	"github.com/coreos/grafiti/deleter"
)

// Hard-coded rounds of retrieval
var r0 = arn.ResourceTypes{
	arn.EC2VPCRType,
	arn.AutoScalingGroupRType,
	arn.Route53HostedZoneRType,
	arn.S3BucketRType,
}

var r1 = arn.ResourceTypes{
	arn.EC2NatGatewayRType,
	arn.EC2InternetGatewayRType,
	arn.EC2InstanceRType,
	arn.EC2SubnetRType,
	arn.EC2NetworkInterfaceRType,
	arn.EC2SecurityGroupRType,
	arn.EC2RouteTableRType,
	arn.AutoScalingLaunchConfigurationRType,
}

var r2 = arn.ResourceTypes{
	arn.EC2RouteTableAssociationRType,
	arn.EC2EIPRType,
	arn.EC2EIPAssociationRType,
	arn.EC2NetworkACLRType,
}

var rounds = []arn.ResourceTypes{r0, r1, r2}

// FillDependencyGraph creates a depGraph starting from an inital set of
// resources found by tags
func FillDependencyGraph(initDepMap map[arn.ResourceType]deleter.ResourceDeleter) {
	if initDepMap == nil {
		return
	}

	for _, round := range rounds {
		for _, r := range round {
			if _, ok := initDepMap[r]; ok {
				traverseDependencyGraph(r, initDepMap)
			}
		}
	}

	return
}

// traverseDependencyGraph traverses necesssary linkages of each resource
func traverseDependencyGraph(rt arn.ResourceType, depMap map[arn.ResourceType]deleter.ResourceDeleter) {
	if _, ok := depMap[rt]; !ok {
		return
	}

	switch rt {
	case arn.EC2VPCRType:
		// Get Subnets
		// Get Instances
		// Get IGW's
		// Get NGW's
		// Get Security Groups
		// Get Route Tables
		// Get Network Interfaces
	case arn.EC2SubnetRType:
		// Get Network ACL's
	case arn.EC2SecurityGroupRType:
		// Get SecurityGroup Rule
		// IP permissions Ingress/Egress will be deleted when deleting SecurityGroups
	case arn.EC2InstanceRType:
		// Get EBS Volumes
	case arn.EC2NetworkInterfaceRType:
		// Get EIP Addresses
		// Get EIP Associations
	case arn.ElasticLoadBalancingLoadBalancerRType:
	case arn.EC2RouteTableRType:
		// RouteTable Routes will be deleted when deleting a RouteTable
		// Get Subnet-RouteTable Association
	case arn.AutoScalingGroupRType:
		// Get autoscaling groups
		asgs, aerr := depMap[rt].(*deleter.AutoScalingGroupDeleter).RequestAutoScalingGroups()
		if aerr != nil || len(asgs) == 0 {
			break
		}

		// Get launch configurations
		lct := arn.ResourceType(arn.AutoScalingLaunchConfigurationRType)
		for _, asg := range asgs {
			depMap[lct].AddResourceNames(arn.ResourceName(*asg.LaunchConfigurationName))
		}
		// Get ELB's
		// Get IAM instance profiles
		// Get IAM roles
		// IAM RolePolicies will be deleted when deleting Roles
	case arn.Route53HostedZoneRType:
		// Route53 RecordSets will be deleted when deleting HostedZones
	case arn.S3BucketRType:
		// S3 Objects will be deleted when deleting a Bucket
	}
	return
}

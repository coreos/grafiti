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
func FillDependencyGraph(initDepMap map[arn.ResourceType]deleter.ResourceDeleter) map[arn.ResourceType]deleter.ResourceDeleter {
	if initDepMap == nil {
		return nil
	}

	for _, round := range rounds {
		for _, r := range round {
			if _, ok := initDepMap[r]; ok {
				initDepMap = traverseDependencyGraph(r, initDepMap)
			}
		}
	}

	return initDepMap
}

// traverseDependencyGraph traverses necesssary linkages of each resource
func traverseDependencyGraph(rt arn.ResourceType, depMap map[arn.ResourceType]deleter.ResourceDeleter) map[arn.ResourceType]deleter.ResourceDeleter {
	if _, ok := depMap[rt]; !ok {
		return depMap
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
		adrDel := depMap[rt].(*deleter.EC2NetworkInterfaceDeleter)
		adrs, err := adrDel.RequestEC2EIPAddressessFromNetworkInterfaces()
		if err != nil || len(adrs) == 0 {
			break
		}

		// Get EIP Allocations
		eipDel := &deleter.EC2ElasticIPAllocationDeleter{}
		if _, ok := depMap[arn.EC2EIPRType]; !ok {
			depMap[arn.EC2EIPRType] = eipDel
		}
		// Get EIP Associations
		eipaDel := &deleter.EC2ElasticIPAssocationDeleter{}
		if _, ok := depMap[arn.EC2EIPAssociationRType]; !ok {
			depMap[arn.EC2EIPAssociationRType] = eipaDel
		}
		for _, adr := range adrs {
			if adr.AllocationId != nil {
				eipDel.AddResourceNames(arn.ResourceName(*adr.AllocationId))
			}
			if adr.AssociationId != nil {
				eipaDel.AddResourceNames(arn.ResourceName(*adr.AssociationId))
			}
		}
	case arn.ElasticLoadBalancingLoadBalancerRType:
	case arn.EC2RouteTableRType:
		// RouteTable Routes will be deleted when deleting a RouteTable
		rtDel := depMap[rt].(*deleter.EC2RouteTableDeleter)
		rts, rerr := rtDel.RequestEC2RouteTables()
		if rerr != nil || len(rts) == 0 {
			break
		}

		// Get Subnet-RouteTable Association
		if _, ok := depMap[arn.EC2RouteTableAssociationRType]; !ok {
			depMap[arn.EC2RouteTableAssociationRType] = &deleter.EC2RouteTableAssociationDeleter{}
		}
		for _, rt := range rts {
			for _, rta := range rt.Associations {
				if rta.Main != nil && !*rta.Main {
					depMap[arn.EC2RouteTableAssociationRType].AddResourceNames(arn.ResourceName(*rta.RouteTableAssociationId))
				}
			}
		}
	case arn.AutoScalingGroupRType:
		// Get autoscaling groups
		asgDel := depMap[rt].(*deleter.AutoScalingGroupDeleter)
		asgs, err := asgDel.RequestAutoScalingGroups()
		if err != nil || len(asgs) == 0 {
			break
		}

		// Get launch configurations
		lcDel := &deleter.AutoScalingLaunchConfigurationDeleter{}
		if _, ok := depMap[arn.AutoScalingLaunchConfigurationRType]; !ok {
			depMap[arn.AutoScalingLaunchConfigurationRType] = lcDel
		}
		for _, asg := range asgs {
			lcDel.AddResourceNames(arn.ResourceName(*asg.LaunchConfigurationName))
		}
		// Get ELB's
	case arn.AutoScalingLaunchConfigurationRType:
		// Get IAM instance profiles
		lcDel := depMap[rt].(*deleter.AutoScalingLaunchConfigurationDeleter)
		lcs, err := lcDel.RequestAutoScalingLaunchConfigurations()
		if err != nil || len(lcs) == 0 {
			break
		}

		iprDel := &deleter.IAMInstanceProfileDeleter{}
		if _, ok := depMap[arn.IAMInstanceProfileRType]; !ok {
			depMap[arn.IAMInstanceProfileRType] = iprDel
		}
		for _, lc := range lcs {
			depMap[arn.IAMInstanceProfileRType].AddResourceNames(arn.ResourceName(*lc.IamInstanceProfile))
		}

		// Get IAM roles
		iprs, err := iprDel.RequestIAMInstanceProfilesFromLaunchConfigurations(lcs)
		if err != nil || len(iprs) == 0 {
			break
		}

		if _, ok := depMap[arn.IAMRoleRType]; !ok {
			depMap[arn.IAMRoleRType] = &deleter.IAMRoleDeleter{}
		}
		for _, ipr := range iprs {
			for _, rl := range ipr.Roles {
				depMap[arn.IAMRoleRType].AddResourceNames(arn.ResourceName(*rl.RoleName))
			}
		}
	case arn.IAMRoleRType:
		// IAM RolePolicies will be deleted when deleting Roles
	case arn.Route53HostedZoneRType:
		// Route53 RecordSets will be deleted when deleting HostedZones
	case arn.S3BucketRType:
		// S3 Objects will be deleted when deleting a Bucket
	}

	return depMap
}

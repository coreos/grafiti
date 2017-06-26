package graph

import (
	"fmt"

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
	arn.EC2VPNGatewayRType,
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
		vpcDel := depMap[rt].(*deleter.EC2VPCDeleter)

		// Ensures that no default VPC's are used
		vpcs, err := vpcDel.RequestEC2VPCs()
		if err != nil || len(vpcs) == 0 {
			break
		}

		vpcDel.ResourceNames = nil
		for _, vpc := range vpcs {
			fmt.Println("VPC:", *vpc.VpcId)
			vpcDel.AddResourceNames(arn.ResourceName(*vpc.VpcId))
		}

		// Get EC2 instances
		instances, _ := vpcDel.RequestEC2InstancesFromVPCs()
		if _, ok := depMap[arn.EC2InstanceRType]; !ok {
			depMap[arn.EC2InstanceRType] = deleter.InitResourceDeleter(arn.EC2InstanceRType)
		}
		for _, instance := range instances {
			fmt.Println("EC2InstanceRType:", *instance.InstanceId)
			depMap[arn.EC2InstanceRType].AddResourceNames(arn.ResourceName(*instance.InstanceId))
		}

		// Get EC2 internet gateways
		igws, _ := vpcDel.RequestEC2InternetGatewaysFromVPCs()
		if _, ok := depMap[arn.EC2InternetGatewayRType]; !ok {
			depMap[arn.EC2InternetGatewayRType] = deleter.InitResourceDeleter(arn.EC2InternetGatewayRType)
		}
		for _, igw := range igws {
			fmt.Println("EC2InternetGatewayRType:", *igw.InternetGatewayId)
			depMap[arn.EC2InternetGatewayRType].AddResourceNames(arn.ResourceName(*igw.InternetGatewayId))
		}

		// Get EC2 NAT gateways
		ngws, _ := vpcDel.RequestEC2NatGatewaysFromVPCs()
		if _, ok := depMap[arn.EC2NatGatewayRType]; !ok {
			depMap[arn.EC2NatGatewayRType] = deleter.InitResourceDeleter(arn.EC2NatGatewayRType)
		}
		for _, ngw := range ngws {
			fmt.Println("EC2NatGatewayRType:", *ngw.NatGatewayId)
			depMap[arn.EC2NatGatewayRType].AddResourceNames(arn.ResourceName(*ngw.NatGatewayId))
		}

		// Get EC2 network interfaces
		enis, _ := vpcDel.RequestEC2NetworkInterfacesFromVPCs()
		if _, ok := depMap[arn.EC2NetworkInterfaceRType]; !ok {
			depMap[arn.EC2NetworkInterfaceRType] = deleter.InitResourceDeleter(arn.EC2NetworkInterfaceRType)
		}
		for _, eni := range enis {
			fmt.Println("EC2NetworkInterfaceRType:", *eni.NetworkInterfaceId)
			depMap[arn.EC2NetworkInterfaceRType].AddResourceNames(arn.ResourceName(*eni.NetworkInterfaceId))
		}

		// Get Route Tables
		rts, _ := vpcDel.RequestEC2RouteTablesFromVPCs()
		if _, ok := depMap[arn.EC2RouteTableRType]; !ok {
			depMap[arn.EC2RouteTableRType] = deleter.InitResourceDeleter(arn.EC2RouteTableRType)
		}
		for _, rt := range rts {
			fmt.Println("EC2RouteTableRType:", *rt.RouteTableId)
			depMap[arn.EC2RouteTableRType].AddResourceNames(arn.ResourceName(*rt.RouteTableId))
		}

		// Get Security Groups
		sgs, _ := vpcDel.RequestEC2SecurityGroupsFromVPCs()
		if _, ok := depMap[arn.EC2SecurityGroupRType]; !ok {
			depMap[arn.EC2SecurityGroupRType] = deleter.InitResourceDeleter(arn.EC2SecurityGroupRType)
		}
		for _, sg := range sgs {
			fmt.Println("EC2SecurityGroupRType:", *sg.GroupId)
			depMap[arn.EC2SecurityGroupRType].AddResourceNames(arn.ResourceName(*sg.GroupId))
		}

		// Get Subnets
		sns, _ := vpcDel.RequestEC2SubnetsFromVPCs()
		if _, ok := depMap[arn.EC2SubnetRType]; !ok {
			depMap[arn.EC2SubnetRType] = deleter.InitResourceDeleter(arn.EC2SubnetRType)
		}
		for _, sn := range sns {
			fmt.Println("EC2SubnetRType:", *sn.SubnetId)
			depMap[arn.EC2SubnetRType].AddResourceNames(arn.ResourceName(*sn.SubnetId))
		}

		// Get VPN Gateways
		vgws, _ := vpcDel.RequestEC2VPNGatewaysFromVPCs()
		if _, ok := depMap[arn.EC2VPNGatewayRType]; !ok {
			depMap[arn.EC2VPNGatewayRType] = deleter.InitResourceDeleter(arn.EC2VPNGatewayRType)
		}
		for _, vgw := range vgws {
			fmt.Println("EC2VPNGatewayRType:", *vgw.VpnGatewayId)
			depMap[arn.EC2VPNGatewayRType].AddResourceNames(arn.ResourceName(*vgw.VpnGatewayId))
		}
	case arn.EC2VPNGatewayRType:
		vgwDel := depMap[rt].(*deleter.EC2VPNGatewayDeleter)
		// Get EC2 vpn connections
		vcs, err := vgwDel.RequestEC2VPNConnectionsFromVPNGateways()
		if err != nil || len(vcs) == 0 {
			break
		}

		if _, ok := depMap[arn.EC2VPNConnectionRType]; !ok {
			depMap[arn.EC2VPNConnectionRType] = deleter.InitResourceDeleter(arn.EC2VPNConnectionRType)
		}
		for _, vc := range vcs {
			fmt.Println("EC2VPNConnectionRType:", *vc.VpnConnectionId)
			depMap[arn.EC2VPNConnectionRType].AddResourceNames(arn.ResourceName(*vc.VpnConnectionId))
		}
	case arn.EC2SubnetRType:
		// Get Network ACL's
	case arn.EC2InstanceRType:
		instanceDel := depMap[rt].(*deleter.EC2InstanceDeleter)
		// Get IAM instance profiles
		iprs, err := instanceDel.RequestIAMInstanceProfilesFromInstances()
		if err != nil || len(iprs) == 0 {
			break
		}

		if _, ok := depMap[arn.IAMInstanceProfileRType]; !ok {
			depMap[arn.IAMInstanceProfileRType] = deleter.InitResourceDeleter(arn.IAMInstanceProfileRType)
		}
		if _, ok := depMap[arn.IAMRoleRType]; !ok {
			depMap[arn.IAMRoleRType] = deleter.InitResourceDeleter(arn.IAMRoleRType)
		}
		for _, ipr := range iprs {
			fmt.Println("IAMInstanceProfileRType:", *ipr.InstanceProfileName)
			depMap[arn.IAMInstanceProfileRType].AddResourceNames(arn.ResourceName(*ipr.InstanceProfileName))
			// Get IAM roles
			for _, rl := range ipr.Roles {
				fmt.Println("IAMRoleRType:", *rl.RoleName)
				depMap[arn.IAMRoleRType].AddResourceNames(arn.ResourceName(*rl.RoleName))
			}
		}
		// Get EBS Volumes
	case arn.EC2NetworkInterfaceRType:
		// Get EIP Addresses
		adrDel := depMap[rt].(*deleter.EC2NetworkInterfaceDeleter)
		adrs, err := adrDel.RequestEC2EIPAddressessFromNetworkInterfaces()
		if err != nil || len(adrs) == 0 {
			break
		}

		// Get EIP Allocations
		if _, ok := depMap[arn.EC2EIPRType]; !ok {
			depMap[arn.EC2EIPRType] = deleter.InitResourceDeleter(arn.EC2EIPRType)
		}
		// Get EIP Associations
		if _, ok := depMap[arn.EC2EIPAssociationRType]; !ok {
			depMap[arn.EC2EIPAssociationRType] = deleter.InitResourceDeleter(arn.EC2EIPAssociationRType)
		}
		for _, adr := range adrs {
			if adr.AllocationId != nil {
				fmt.Println("EC2EIPRType:", *adr.AllocationId)
				depMap[arn.EC2EIPRType].AddResourceNames(arn.ResourceName(*adr.AllocationId))
			}
			if adr.AssociationId != nil {
				fmt.Println("EC2EIPAssociationRType:", *adr.AssociationId)
				depMap[arn.EC2EIPAssociationRType].AddResourceNames(arn.ResourceName(*adr.AssociationId))
			}
		}
	case arn.EC2RouteTableRType:
		// RouteTable Routes will be deleted when deleting a RouteTable
		rtDel := depMap[rt].(*deleter.EC2RouteTableDeleter)
		rts, rerr := rtDel.RequestEC2RouteTables()
		if rerr != nil || len(rts) == 0 {
			break
		}

		// Get Subnet-RouteTable Association
		if _, ok := depMap[arn.EC2RouteTableAssociationRType]; !ok {
			depMap[arn.EC2RouteTableAssociationRType] = deleter.InitResourceDeleter(arn.EC2RouteTableAssociationRType)
		}
		for _, rt := range rts {
			for _, rta := range rt.Associations {
				if rta.Main != nil && !*rta.Main {
					fmt.Println("EC2RouteTableAssociationRType:", *rta.RouteTableAssociationId)
					depMap[arn.EC2RouteTableAssociationRType].AddResourceNames(arn.ResourceName(*rta.RouteTableAssociationId))
				}
			}
		}
	case arn.AutoScalingGroupRType:
		asgDel := depMap[rt].(*deleter.AutoScalingGroupDeleter)
		asgs, err := asgDel.RequestAutoScalingGroups()
		if err != nil || len(asgs) == 0 {
			break
		}

		// Get launch configurations
		if _, ok := depMap[arn.AutoScalingLaunchConfigurationRType]; !ok {
			depMap[arn.AutoScalingLaunchConfigurationRType] = deleter.InitResourceDeleter(arn.AutoScalingLaunchConfigurationRType)
		}
		// Get ELB's
		if _, ok := depMap[arn.ElasticLoadBalancingLoadBalancerRType]; !ok {
			depMap[arn.ElasticLoadBalancingLoadBalancerRType] = deleter.InitResourceDeleter(arn.ElasticLoadBalancingLoadBalancerRType)
		}
		for _, asg := range asgs {
			fmt.Println("AutoScalingGroupRType:", *asg.AutoScalingGroupName)
			fmt.Println("AutoScalingLaunchConfigurationRType:", *asg.LaunchConfigurationName)
			depMap[arn.AutoScalingLaunchConfigurationRType].AddResourceNames(arn.ResourceName(*asg.LaunchConfigurationName))
			for _, elbName := range asg.LoadBalancerNames {
				fmt.Println("ElasticLoadBalancingLoadBalancerRType:", *elbName)
				depMap[arn.ElasticLoadBalancingLoadBalancerRType].AddResourceNames(arn.ResourceName(*elbName))
			}
		}
	case arn.AutoScalingLaunchConfigurationRType:
		lcDel := depMap[rt].(*deleter.AutoScalingLaunchConfigurationDeleter)
		// Get IAM instance profiles
		iprs, err := lcDel.RequestIAMInstanceProfilesFromLaunchConfigurations()
		if err != nil || len(iprs) == 0 {
			break
		}

		if _, ok := depMap[arn.IAMInstanceProfileRType]; !ok {
			depMap[arn.IAMInstanceProfileRType] = deleter.InitResourceDeleter(arn.IAMInstanceProfileRType)
		}
		if _, ok := depMap[arn.IAMRoleRType]; !ok {
			depMap[arn.IAMRoleRType] = deleter.InitResourceDeleter(arn.IAMRoleRType)
		}
		for _, ipr := range iprs {
			fmt.Println("IAMInstanceProfileRType:", *ipr.InstanceProfileName)
			depMap[arn.IAMInstanceProfileRType].AddResourceNames(arn.ResourceName(*ipr.InstanceProfileName))
			// Get IAM roles
			for _, rl := range ipr.Roles {
				fmt.Println("IAMRoleRType:", *rl.RoleName)
				depMap[arn.IAMRoleRType].AddResourceNames(arn.ResourceName(*rl.RoleName))
			}
		}
	}

	return depMap
}

package graph

import (
	"github.com/coreos/grafiti/arn"
	"github.com/coreos/grafiti/describe"
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
func FillDependencyGraph(initDepMap *map[arn.ResourceType]arn.ResourceNames) {
	if initDepMap == nil {
		return
	}

	for _, round := range rounds {
		for _, r := range round {
			if _, ok := (*initDepMap)[r]; ok {
				traverseDependencyGraph(r, initDepMap)
			}
		}
	}

	return
}

// traverseDependencyGraph traverses necesssary linkages of each resource
func traverseDependencyGraph(rt arn.ResourceType, depMap *map[arn.ResourceType]arn.ResourceNames) {
	if depMap == nil {
		return
	}
	ids := (*depMap)[rt]
	if ids == nil {
		return
	}
	switch rt {
	case arn.EC2VPCRType:
		snt, irt := arn.ResourceType(arn.EC2SubnetRType), arn.ResourceType(arn.EC2InstanceRType)
		igwt, ngwt := arn.ResourceType(arn.EC2InternetGatewayRType), arn.ResourceType(arn.EC2NatGatewayRType)
		sgt, rtt := arn.ResourceType(arn.EC2SecurityGroupRType), arn.ResourceType(arn.EC2RouteTableRType)
		enit := arn.ResourceType(arn.EC2NetworkInterfaceRType)
		// Get Subnet
		sns, _ := describe.GetEC2SubnetsByVPCIDs(&ids)
		if sns != nil {
			for _, sn := range *sns {
				(*depMap)[snt] = append((*depMap)[snt], arn.ResourceName(*sn.SubnetId))
			}
		}
		irs, _ := describe.GetEC2InstanceReservationsByVPCIDs(&ids)
		if irs != nil {
			for _, ir := range *irs {
				for _, i := range ir.Instances {
					(*depMap)[irt] = append((*depMap)[irt], arn.ResourceName(*i.InstanceId))
				}
			}
		}
		igws, _ := describe.GetEC2InternetGatewaysByVPCIDs(&ids)
		if igws != nil {
			for _, igw := range *igws {
				(*depMap)[igwt] = append((*depMap)[igwt], arn.ResourceName(*igw.InternetGatewayId))
			}
		}
		ngws, _ := describe.GetEC2NatGatewaysByVPCIDs(&ids)
		if ngws != nil {
			for _, ngw := range *ngws {
				(*depMap)[ngwt] = append((*depMap)[ngwt], arn.ResourceName(*ngw.NatGatewayId))
			}
		}
		sgs, _ := describe.GetEC2SecurityGroupsByVPCIDs(&ids)
		if sgs != nil {
			for _, sg := range *sgs {
				(*depMap)[sgt] = append((*depMap)[sgt], arn.ResourceName(*sg.GroupId))
			}
		}
		rts, _ := describe.GetEC2RouteTablesByVPCIDs(&ids)
		if rts != nil {
			for _, rt := range *rts {
				(*depMap)[rtt] = append((*depMap)[rtt], arn.ResourceName(*rt.RouteTableId))
			}
		}
		enis, _ := describe.GetEC2NetworkInterfacesByVPCIDs(&ids)
		if enis != nil {
			for _, eni := range *enis {
				(*depMap)[enit] = append((*depMap)[enit], arn.ResourceName(*eni.NetworkInterfaceId))
			}
		}
	case arn.EC2SubnetRType:
		// Get Network ACL's
		nacls, nerr := describe.GetEC2NetworkACLsBySubnetIDs(&ids)
		if nerr != nil || nacls == nil {
			break
		}
		naclt := arn.ResourceType(arn.EC2NetworkACLRType)
		for _, nacl := range *nacls {
			if !*nacl.IsDefault {
				(*depMap)[naclt] = append((*depMap)[naclt], arn.ResourceName(*nacl.NetworkAclId))
			}
		}
	case arn.EC2SecurityGroupRType:
		// Get SecurityGroup Rule
		// IP permissions Ingress/Egress will be deleted when deleting SecurityGroups
	case arn.EC2InstanceRType:
		// Get EBS Volumes
	case arn.EC2NetworkInterfaceRType:
		adrs, aerr := describe.GetEC2EIPAddressesByENIIDs(&ids)
		if aerr != nil || adrs == nil {
			break
		}
		// Get EIP Addresses
		// Get EIP Associations
		eipt, eipat := arn.ResourceType(arn.EC2EIPRType), arn.ResourceType(arn.EC2EIPAssociationRType)
		for _, adr := range *adrs {
			if adr.AllocationId != nil {
				(*depMap)[eipt] = append((*depMap)[eipt], arn.ResourceName(*adr.AllocationId))
			}
			if adr.AssociationId != nil {
				(*depMap)[eipat] = append((*depMap)[eipat], arn.ResourceName(*adr.AssociationId))
			}
		}
	case arn.ElasticLoadBalancingLoadBalancerRType:
	case arn.EC2RouteTableRType:
		// RouteTable Routes will be deleted when deleting a RouteTable
		rts, rerr := describe.GetEC2RouteTablesByIDs(&ids)
		if rerr != nil || rts == nil {
			break
		}
		// Get Subnet-RouteTable Association
		rtat := arn.ResourceType(arn.EC2RouteTableAssociationRType)
		for _, rt := range *rts {
			for _, a := range rt.Associations {
				if !*a.Main {
					(*depMap)[rtat] = append((*depMap)[rtat], arn.ResourceName(*a.RouteTableAssociationId))
				}
			}
		}
	case arn.AutoScalingGroupRType:
		asgs, aerr := describe.GetAutoScalingGroupsByNames(&ids)
		if aerr != nil || asgs == nil {
			break
		}
		// Get AS LaunchConfigurations
		// Get ELB's
		lcs := make(arn.ResourceNames, 0, len(*asgs))
		lct := arn.ResourceType(arn.AutoScalingLaunchConfigurationRType)
		lbt := arn.ResourceType(arn.ElasticLoadBalancingLoadBalancerRType)
		for _, asg := range *asgs {
			lcs = append(lcs, arn.ResourceName(*asg.LaunchConfigurationName))
			(*depMap)[lct] = append((*depMap)[lct], arn.ResourceName(*asg.LaunchConfigurationName))
			for _, elb := range asg.LoadBalancerNames {
				(*depMap)[lbt] = append((*depMap)[lbt], arn.ResourceName(*elb))
			}
		}
		// Get IAM instance profiles
		iprs, ierr := describe.GetIAMInstanceProfilesByLaunchConfigNames(&lcs)
		if ierr != nil || iprs == nil {
			break
		}
		iprt, rlt := arn.ResourceType(arn.IAMInstanceProfileRType), arn.ResourceType(arn.IAMRoleRType)
		for _, ipr := range *iprs {
			(*depMap)[iprt] = append((*depMap)[iprt], arn.ResourceName(*ipr.InstanceProfileName))
			// Get IAM roles
			for _, rl := range ipr.Roles {
				(*depMap)[rlt] = append((*depMap)[rlt], arn.ResourceName(*rl.RoleName))
			}
		}
		// IAM RolePolicies will be deleted when deleting Roles
	case arn.Route53HostedZoneRType:
		// Route53 RecordSets will be deleted when deleting HostedZones
	case arn.S3BucketRType:
		// S3 Objects will be deleted when deleting a Bucket
	}
	return
}

package graph

import (
	"github.com/coreos/grafiti/arn"
	"github.com/coreos/grafiti/describe"
)

// DepGraph is the top-level container for a DepNode graph
type DepGraph struct {
	DepNodes *[]*DepNode
}

// DepNode has a Type (resource type) and a slice of ChildDepNodes that should
// only be travelled to after all Values (ex. *[]*ec2.Instance) have been deleted
type DepNode struct {
	Type          string
	ChildDepNodes *[]*DepNode
}

func newDepNode(t string, children *[]*DepNode) *DepNode {
	return &DepNode{
		Type:          t,
		ChildDepNodes: children,
	}
}

// Hard-coded rounds of retrieval
var r0 = []string{
	arn.EC2VPCRType,
	arn.AutoScalingGroupRType,
	arn.Route53HostedZoneRType,
}

var r1 = map[string][]string{
	arn.EC2VPCRType: []string{
		arn.EC2NatGatewayRType,
		arn.EC2InternetGatewayRType,
		arn.EC2InstanceRType,
		arn.EC2SubnetRType,
		arn.EC2NetworkInterfaceRType,
		arn.EC2SecurityGroupRType,
		arn.EC2RouteTableRType,
	},
	arn.AutoScalingGroupRType: []string{
		arn.AutoScalingLaunchConfigurationRType,
	},
}

var r2 = map[string][]string{
	arn.EC2RouteTableRType: []string{
		arn.EC2RouteTableAssociationRType,
	},
	arn.EC2NetworkInterfaceRType: []string{
		arn.EC2EIPRType,
		arn.EC2EIPAssociationRType,
	},
	arn.EC2SubnetRType: []string{
		arn.EC2NetworkACLRType,
	},
	// arn.AutoScalingLaunchConfigurationRType: []string{
	// 	arn.IAMInstanceProfileRType,
	// },
}

// var r3 = map[string][]string{
// 	arn.IAMInstanceProfileRType: []string{
// 		arn.IAMRoleRType,
// 	},
// }

var rounds = []map[string][]string{r1, r2}

// InitDepGraph creates a new dep graph
func InitDepGraph() DepGraph {
	dns := make([]*DepNode, 0, 3)
	dg := DepGraph{DepNodes: &dns}

	// Initial round
	for _, t := range r0 {
		ndn := newDepNode(t, nil)
		*dg.DepNodes = append(*dg.DepNodes, ndn)
	}
	// Subsequent rounds
	dnsp := dg.DepNodes
	var ndns []*DepNode
	for _, r := range rounds {
		tmp := *dnsp
		for _, dn := range tmp {
			if cts, ok := r[dn.Type]; ok {
				for _, ct := range cts {
					ndn := newDepNode(ct, nil)
					*dnsp = append(*dnsp, ndn)
					ndns = append(ndns, ndn)
				}
			}
		}
		dnsp = &ndns
		ndns = nil
	}
	return dg
}

// FillDependencyGraph creates a DepGraph starting from an inital set of
// resources found by tags
func FillDependencyGraph(initDepMap *map[string][]string) {
	if initDepMap == nil {
		return
	}

	depGraph := InitDepGraph()

	dns := depGraph.DepNodes
	var cdns []*DepNode
	for {
		cdns = nil
		for i, dn := range *dns {
			if _, ok := (*initDepMap)[dn.Type]; ok {
				traverseDependencyGraph(dn.Type, initDepMap)
			}
			if i == len(*dns)-1 {
				if cdns != nil {
					dns = &cdns
				}
				break
			}
			if dn.ChildDepNodes != nil {
				for _, cdn := range *dn.ChildDepNodes {
					cdns = append(cdns, cdn)
				}
			}
		}
		if cdns == nil {
			break
		}
	}

	return
}

// traverseDependencyGraph traverses necesssary linkages of each resource
func traverseDependencyGraph(rType string, depMap *map[string][]string) {
	if depMap == nil {
		return
	}
	ids := (*depMap)[rType]
	if ids == nil {
		return
	}
	switch rType {
	case arn.EC2VPCRType:
		snt, irt := arn.EC2SubnetRType, arn.EC2InstanceRType
		igwt, ngwt := arn.EC2InternetGatewayRType, arn.EC2NatGatewayRType
		sgt, rtt := arn.EC2SecurityGroupRType, arn.EC2RouteTableRType
		enit := arn.EC2NetworkInterfaceRType
		sns, _ := describe.GetEC2SubnetsByVPCIDs(&ids)
		if sns != nil {
			for _, sn := range *sns {
				(*depMap)[snt] = append((*depMap)[snt], *sn.SubnetId)
			}
		}
		irs, _ := describe.GetEC2InstanceReservationsByVPCIDs(&ids)
		if irs != nil {
			for _, ir := range *irs {
				for _, i := range ir.Instances {
					(*depMap)[irt] = append((*depMap)[irt], *i.InstanceId)
				}
			}
		}
		igws, _ := describe.GetEC2InternetGatewaysByVPCIDs(&ids)
		if igws != nil {
			for _, igw := range *igws {
				(*depMap)[igwt] = append((*depMap)[igwt], *igw.InternetGatewayId)
			}
		}
		ngws, _ := describe.GetEC2NatGatewaysByVPCIDs(&ids)
		if ngws != nil {
			for _, ngw := range *ngws {
				(*depMap)[ngwt] = append((*depMap)[ngwt], *ngw.NatGatewayId)
			}
		}
		sgs, _ := describe.GetEC2SecurityGroupsByVPCIDs(&ids)
		if sgs != nil {
			for _, sg := range *sgs {
				(*depMap)[sgt] = append((*depMap)[sgt], *sg.GroupId)
			}
		}
		rts, _ := describe.GetEC2RouteTablesByVPCIDs(&ids)
		if rts != nil {
			for _, rt := range *rts {
				(*depMap)[rtt] = append((*depMap)[rtt], *rt.RouteTableId)
			}
		}
		enis, _ := describe.GetEC2NetworkInterfacesByVPCIDs(&ids)
		if enis != nil {
			for _, eni := range *enis {
				(*depMap)[enit] = append((*depMap)[enit], *eni.NetworkInterfaceId)
			}
		}
		break
	case arn.EC2SubnetRType:
		// Get Network ACL's
		nacls, nerr := describe.GetEC2NetworkACLsBySubnetIDs(&ids)
		if nerr != nil || nacls == nil {
			break
		}
		naclt := arn.EC2NetworkACLRType
		for _, nacl := range *nacls {
			if !*nacl.IsDefault {
				(*depMap)[naclt] = append((*depMap)[naclt], *nacl.NetworkAclId)
			}
		}
		break
	case arn.EC2SecurityGroupRType:
		// Get SecurityGroup Rule
		// IP permissions Ingress/Egress will be deleted when deleting SecurityGroups
		break
	case arn.EC2InstanceRType:
		// Get EBS Volumes
		break
	case arn.EC2NetworkInterfaceRType:
		adrs, aerr := describe.GetEC2EIPAddressesByENIIDs(&ids)
		if aerr != nil || adrs == nil {
			break
		}
		// Get EC2EIPRType
		// Get EC2EIPAssociationRType
		eipt, eipat := arn.EC2EIPRType, arn.EC2EIPAssociationRType
		for _, adr := range *adrs {
			if adr.AllocationId != nil {
				(*depMap)[eipt] = append((*depMap)[eipt], *adr.AllocationId)
			}
			if adr.AssociationId != nil {
				(*depMap)[eipat] = append((*depMap)[eipat], *adr.AssociationId)
			}
		}
		break
	case arn.ElasticLoadBalancingLoadBalancerRType:
		// Get ELB Policies
		break
	case arn.EC2RouteTableRType:
		// RouteTable Routes will be deleted when deleting a RouteTable
		rts, rerr := describe.GetEC2RouteTables(&ids)
		if rerr != nil || rts == nil {
			break
		}
		// Get Subnet-RouteTable Association
		rtat := arn.EC2RouteTableAssociationRType
		for _, rt := range *rts {
			for _, a := range rt.Associations {
				if !*a.Main {
					(*depMap)[rtat] = append((*depMap)[rtat], *a.RouteTableAssociationId)
				}
			}
		}
		break
	case arn.AutoScalingGroupRType:
		asgs, aerr := describe.GetAutoScalingGroupsByNames(&ids)
		if aerr != nil || asgs == nil {
			break
		}
		// Get AS LaunchConfigurations
		// Get ELB's
		lcs := make([]string, 0, len(*asgs))
		lct := arn.AutoScalingLaunchConfigurationRType
		lbt := arn.ElasticLoadBalancingLoadBalancerRType
		for _, asg := range *asgs {
			lcs = append(lcs, *asg.LaunchConfigurationName)
			(*depMap)[lct] = append((*depMap)[lct], *asg.LaunchConfigurationName)
			for _, elb := range asg.LoadBalancerNames {
				(*depMap)[lbt] = append((*depMap)[lbt], *elb)
			}
		}
		// Get IAM instance profiles
		iprs, ierr := describe.GetIAMInstanceProfilesByLaunchConfigs(&lcs)
		if ierr != nil || iprs == nil {
			break
		}
		iprt, rlt := arn.IAMInstanceProfileRType, arn.IAMRoleRType
		for _, ipr := range *iprs {
			(*depMap)[iprt] = append((*depMap)[iprt], *ipr.InstanceProfileName)
			// Get IAM roles
			for _, rl := range ipr.Roles {
				(*depMap)[rlt] = append((*depMap)[rlt], *rl.RoleName)
			}
		}
		// IAM RolePolicies will be deleted when deleting Roles
		break
	case arn.Route53HostedZoneRType:
		// Route53 RecordSets will be deleted when deleting HostedZones
		break
	case arn.S3BucketRType:
		// S3 Objects will be deleted when deleting a Bucket
		break
	}
	return
}

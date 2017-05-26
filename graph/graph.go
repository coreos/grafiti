package graph

import (
	"github.com/coreos/grafiti/arn"
	"github.com/coreos/grafiti/describe"
)

// depGraph is the top-level container for a depNode graph
type depGraph struct {
	DepNodes *[]*depNode
}

// depNode has a Type (resource type) and a slice of ChildDepNodes that should
// only be travelled to after all Values (ex. *[]*ec2.Instance) have been deleted
type depNode struct {
	Type          string
	ChildDepNodes *[]*depNode
}

func newDepNode(t string, children *[]*depNode) *depNode {
	return &depNode{
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
}

var rounds = []map[string][]string{r1, r2}

// InitDepGraph creates a new dep graph
func initDepGraph() depGraph {
	nodes := make([]*depNode, 0)
	dg := depGraph{DepNodes: &nodes}

	// Initial round
	for _, t := range r0 {
		ndn := newDepNode(t, nil)
		*dg.DepNodes = append(*dg.DepNodes, ndn)
	}

	// Subsequent rounds
	tmpNodes := *dg.DepNodes
	var childNodes, roundNodes []*depNode
	for _, r := range rounds {
		for _, node := range tmpNodes {
			if cts, ok := r[node.Type]; ok {
				for _, ct := range cts {
					newNode := newDepNode(ct, nil)
					childNodes = append(childNodes, newNode)
					roundNodes = append(roundNodes, newNode)
				}
			}
			if node.ChildDepNodes == nil {
				node.ChildDepNodes = new([]*depNode)
				*node.ChildDepNodes = childNodes
			}
			childNodes = nil
		}
		tmpNodes = roundNodes
		roundNodes = nil
	}
	return dg
}

// FillDependencyGraph creates a depGraph starting from an inital set of
// resources found by tags
func FillDependencyGraph(initDepMap *map[string][]string) {
	if initDepMap == nil {
		return
	}

	depGraph := initDepGraph()

	tmpNodes := *depGraph.DepNodes
	var nextNodes []*depNode
	for {
		for _, node := range tmpNodes {
			if _, ok := (*initDepMap)[node.Type]; ok {
				traverseDependencyGraph(node.Type, initDepMap)
			}
			if node.ChildDepNodes != nil {
				for _, cnode := range *node.ChildDepNodes {
					nextNodes = append(nextNodes, cnode)
				}
			}
		}
		if nextNodes == nil {
			break
		}
		tmpNodes = nextNodes
		nextNodes = nil
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
		// Get Subnet
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
		eipt, eipat := arn.EC2EIPRType, arn.EC2EIPAssociationRType
		for _, adr := range *adrs {
			if adr.AllocationId != nil {
				(*depMap)[eipt] = append((*depMap)[eipt], *adr.AllocationId)
			}
			if adr.AssociationId != nil {
				(*depMap)[eipat] = append((*depMap)[eipat], *adr.AssociationId)
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
		rtat := arn.EC2RouteTableAssociationRType
		for _, rt := range *rts {
			for _, a := range rt.Associations {
				if !*a.Main {
					(*depMap)[rtat] = append((*depMap)[rtat], *a.RouteTableAssociationId)
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
		iprs, ierr := describe.GetIAMInstanceProfilesByLaunchConfigNames(&lcs)
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
	case arn.Route53HostedZoneRType:
		// Route53 RecordSets will be deleted when deleting HostedZones
	case arn.S3BucketRType:
		// S3 Objects will be deleted when deleting a Bucket
	}
	return
}

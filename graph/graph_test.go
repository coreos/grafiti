package graph

import (
	"reflect"
	"testing"

	"github.com/coreos/grafiti/arn"
)

func TestFillDependencyGraph(t *testing.T) {
	cases := []struct {
		Input    map[string][]string
		Expected map[string][]string
	}{
		{
			Input: map[string][]string{
				arn.EC2VPCRType:                         []string{},
				arn.AutoScalingGroupRType:               []string{},
				arn.Route53HostedZoneRType:              []string{},
				arn.S3BucketRType:                       []string{},
				arn.EC2NatGatewayRType:                  []string{},
				arn.EC2InternetGatewayRType:             []string{},
				arn.EC2InstanceRType:                    []string{},
				arn.EC2SubnetRType:                      []string{},
				arn.EC2NetworkInterfaceRType:            []string{},
				arn.EC2SecurityGroupRType:               []string{},
				arn.EC2RouteTableRType:                  []string{},
				arn.AutoScalingLaunchConfigurationRType: []string{},
				arn.EC2RouteTableAssociationRType:       []string{},
				arn.EC2EIPRType:                         []string{},
				arn.EC2EIPAssociationRType:              []string{},
				arn.EC2NetworkACLRType:                  []string{},
			},
			Expected: map[string][]string{
				arn.EC2VPCRType:                         []string{},
				arn.AutoScalingGroupRType:               []string{},
				arn.Route53HostedZoneRType:              []string{},
				arn.S3BucketRType:                       []string{},
				arn.EC2NatGatewayRType:                  []string{},
				arn.EC2InternetGatewayRType:             []string{},
				arn.EC2InstanceRType:                    []string{},
				arn.EC2SubnetRType:                      []string{},
				arn.EC2NetworkInterfaceRType:            []string{},
				arn.EC2SecurityGroupRType:               []string{},
				arn.EC2RouteTableRType:                  []string{},
				arn.AutoScalingLaunchConfigurationRType: []string{},
				arn.EC2RouteTableAssociationRType:       []string{},
				arn.EC2EIPRType:                         []string{},
				arn.EC2EIPAssociationRType:              []string{},
				arn.EC2NetworkACLRType:                  []string{},
			},
		},
	}

	for _, c := range cases {
		FillDependencyGraph(&c.Input)

		if !reflect.DeepEqual(c.Input, c.Expected) {
			t.Errorf("FillDependencyGraph failed\nwanted\n%s\ngot\n%s\n", c.Expected, c.Input)
		}
	}
}

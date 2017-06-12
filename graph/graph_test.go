package graph

import (
	"reflect"
	"testing"

	"github.com/coreos/grafiti/arn"
)

func TestFillDependencyGraph(t *testing.T) {
	cases := []struct {
		Input    map[arn.ResourceType]arn.ResourceNames
		Expected map[arn.ResourceType]arn.ResourceNames
	}{
		{
			Input: map[arn.ResourceType]arn.ResourceNames{
				arn.EC2VPCRType:                         arn.ResourceNames{},
				arn.AutoScalingGroupRType:               arn.ResourceNames{},
				arn.Route53HostedZoneRType:              arn.ResourceNames{},
				arn.S3BucketRType:                       arn.ResourceNames{},
				arn.EC2NatGatewayRType:                  arn.ResourceNames{},
				arn.EC2InternetGatewayRType:             arn.ResourceNames{},
				arn.EC2InstanceRType:                    arn.ResourceNames{},
				arn.EC2SubnetRType:                      arn.ResourceNames{},
				arn.EC2NetworkInterfaceRType:            arn.ResourceNames{},
				arn.EC2SecurityGroupRType:               arn.ResourceNames{},
				arn.EC2RouteTableRType:                  arn.ResourceNames{},
				arn.AutoScalingLaunchConfigurationRType: arn.ResourceNames{},
				arn.EC2RouteTableAssociationRType:       arn.ResourceNames{},
				arn.EC2EIPRType:                         arn.ResourceNames{},
				arn.EC2EIPAssociationRType:              arn.ResourceNames{},
				arn.EC2NetworkACLRType:                  arn.ResourceNames{},
			},
			Expected: map[arn.ResourceType]arn.ResourceNames{
				arn.EC2VPCRType:                         arn.ResourceNames{},
				arn.AutoScalingGroupRType:               arn.ResourceNames{},
				arn.Route53HostedZoneRType:              arn.ResourceNames{},
				arn.S3BucketRType:                       arn.ResourceNames{},
				arn.EC2NatGatewayRType:                  arn.ResourceNames{},
				arn.EC2InternetGatewayRType:             arn.ResourceNames{},
				arn.EC2InstanceRType:                    arn.ResourceNames{},
				arn.EC2SubnetRType:                      arn.ResourceNames{},
				arn.EC2NetworkInterfaceRType:            arn.ResourceNames{},
				arn.EC2SecurityGroupRType:               arn.ResourceNames{},
				arn.EC2RouteTableRType:                  arn.ResourceNames{},
				arn.AutoScalingLaunchConfigurationRType: arn.ResourceNames{},
				arn.EC2RouteTableAssociationRType:       arn.ResourceNames{},
				arn.EC2EIPRType:                         arn.ResourceNames{},
				arn.EC2EIPAssociationRType:              arn.ResourceNames{},
				arn.EC2NetworkACLRType:                  arn.ResourceNames{},
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

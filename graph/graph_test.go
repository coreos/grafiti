package graph

import (
	"reflect"
	"testing"

	"github.com/coreos/grafiti/arn"
	"github.com/coreos/grafiti/deleter"
)

func TestFillDependencyGraph(t *testing.T) {
	cases := []struct {
		Input    map[arn.ResourceType]deleter.ResourceDeleter
		Expected map[arn.ResourceType]deleter.ResourceDeleter
	}{
		{
			Input: map[arn.ResourceType]deleter.ResourceDeleter{
				arn.AutoScalingGroupRType:               &deleter.AutoScalingGroupDeleter{},
				arn.AutoScalingLaunchConfigurationRType: &deleter.AutoScalingLaunchConfigurationDeleter{},
			},
			Expected: map[arn.ResourceType]deleter.ResourceDeleter{
				arn.AutoScalingGroupRType:               &deleter.AutoScalingGroupDeleter{},
				arn.AutoScalingLaunchConfigurationRType: &deleter.AutoScalingLaunchConfigurationDeleter{},
			},
		},
	}

	for _, c := range cases {
		FillDependencyGraph(c.Input)

		if !reflect.DeepEqual(c.Input, c.Expected) {
			t.Errorf("FillDependencyGraph failed\nwanted\n%s\ngot\n%s\n", c.Expected, c.Input)
		}
	}
}

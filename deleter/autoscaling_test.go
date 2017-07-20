package deleter

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

func TestCreateInstanceProfileMap(t *testing.T) {
	cases := []struct {
		InputLCs []*autoscaling.LaunchConfiguration
		Expected map[string]struct{}
	}{
		{
			InputLCs: []*autoscaling.LaunchConfiguration{
				&autoscaling.LaunchConfiguration{
					IamInstanceProfile: aws.String("instance-profile-name-1"),
				},
				&autoscaling.LaunchConfiguration{
					IamInstanceProfile: aws.String("arn:aws:iam:::instance-profile/instance-profile-name-2"),
				},
			},
			Expected: map[string]struct{}{
				"instance-profile-name-1": {},
				"instance-profile-name-2": {},
			},
		},
	}

	for i, c := range cases {
		got := createInstanceProfileMap(c.InputLCs)
		if !reflect.DeepEqual(c.Expected, got) {
			t.Errorf("createInstanceProfileMap case %d failed\nwanted %s\ngot %s\n", i+i, c.Expected, got)
		}
	}
}

package deleter

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/coreos/grafiti/arn"
	"github.com/spf13/viper"
)

const drStr = "(dry-run)"

func setUpAWSSession() *session.Session {
	return session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(viper.GetString("grafiti.az")),
		},
	))
}

// DeleteConfig holds configuration info for resource deletion
type DeleteConfig struct {
	IgnoreErrors bool
	DryRun       bool
	ResourceType string
	AWSSession   *session.Session
}

// DelFunc takes a *DeleteConfig and a string, and returns an error
type DelFunc func(*DeleteConfig, string) error

// A ResourceDeleter is any type that can delete itself from AWS and describe
// itself using an AWS request
type ResourceDeleter interface {
	// Adds resource names to the ResourceDeleter
	AddResourceNames(...arn.ResourceName)
	// Delete resources using DeleteConfig info
	DeleteResources(*DeleteConfig) error
}

// InitResourceDeleter creates a ResourceDeleter using a type defined in
// the `deleter` package
func InitResourceDeleter(t arn.ResourceType) ResourceDeleter {
	switch t {
	case arn.AutoScalingGroupRType:
		return &AutoScalingGroupDeleter{ResourceType: t}
	case arn.AutoScalingLaunchConfigurationRType:
		return &AutoScalingLaunchConfigurationDeleter{ResourceType: t}
	}
	return nil
}

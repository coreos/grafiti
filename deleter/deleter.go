package deleter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/coreos/grafiti/arn"
	"github.com/coreos/grafiti/deleter/retryer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const drStr = "(dry-run)"

func setUpAWSSession() *session.Session {
	maxRetries := viper.GetInt("maxNumRequestRetries")
	return session.Must(session.NewSession(
		&aws.Config{
			Region:  aws.String(viper.GetString("region")),
			Retryer: retryer.DeleteRetryer{NumMaxRetries: maxRetries},
		},
	))
}

// CalcChunk calculates the ending index of a slice
func CalcChunk(curr, size, chunk int) int {
	if curr+chunk > size {
		return size
	}
	return curr + chunk
}

// DeleteConfig holds configuration info for resource deletion
type DeleteConfig struct {
	DryRun       bool
	IgnoreErrors bool
	Logger       logrus.FieldLogger
}

// InitRequestLogger creates a logrus.FieldLogger that logs to a file at path,
// or os.Stdout if an error occurs opening the file
func InitRequestLogger(path string) logrus.FieldLogger {
	logger := &logrus.Logger{
		Out:       os.Stdout,
		Formatter: &logrus.JSONFormatter{},
		Level:     logrus.InfoLevel,
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0664)
	if err == nil {
		logger.Out = f
	} else {
		logger.Infof("Failed to open file %s for logging, using stdout instead", path)
	}

	return logger
}

// LogEntry maps potential log entry fields to a Go struct. Add fields here when
// creating fields in ResourceDeleter.DeleteResources implementations
type LogEntry struct {
	Error              error            `json:"error"`
	ResourceType       arn.ResourceType `json:"resource_type"`
	ResourceName       arn.ResourceName `json:"resource_name"`
	AWSErrorCode       string           `json:"aws_err_code,omitempty"`
	AWSErrorMsg        string           `json:"aws_err_msg,omitempty"`
	ErrMsg             string           `json:"err_msg,omitempty"`
	ParentResourceType arn.ResourceType `json:"parent_resource_type,omitempty"`
	ParentResourceName arn.ResourceName `json:"parent_resource_name,omitempty"`
}

// Log errors to a DeleteConfig.Logger
func (c *DeleteConfig) logRequestError(rt arn.ResourceType, rn interface{}, err error, extraFields ...logrus.Fields) {
	fields := logrus.Fields{
		"error":         err,
		"resource_type": rt,
		"resource_name": rn,
	}

	if aerr, ok := err.(awserr.Error); ok {
		fields["aws_err_code"] = aerr.Code()
		fields["aws_err_msg"] = aerr.Message()
	} else {
		fields["err_msg"] = err.Error()
	}

	// Allow overwriting old fields or adding extra fields if desired
	for _, ef := range extraFields {
		for fk, fv := range ef {
			fields[fk] = fv
		}
	}

	failMsg := fmt.Sprintf("Failed to delete %s \"%v\"", rt, rn)
	if fields["parent_resource_type"] != nil && fields["parent_resource_name"] != nil {
		failMsg += fmt.Sprintf("from %s \"%s\"", fields["parent_resource_type"], fields["parent_resource_name"])
	}
	fmt.Printf("%s: %s\n", failMsg, err.Error())

	c.Logger.WithFields(fields).Info("Resource request failed.")
}

// Log request data to a DeleteConfig.Logger
func (c *DeleteConfig) logRequestSuccess(rt arn.ResourceType, rn interface{}, extraFields ...logrus.Fields) {
	fields := logrus.Fields{
		"error":         nil,
		"resource_type": rt,
		"resource_name": rn,
	}

	// Allow overwriting old fields or adding extra fields if desired
	for _, ef := range extraFields {
		for fk, fv := range ef {
			fields[fk] = fv
		}
	}

	c.Logger.WithFields(fields).Info("Resource request was successful.")
}

// LogFormatFunc formats LogEntry structs into a string
type LogFormatFunc func(*LogEntry) string

// PrintLogFileReport prints LogEntry structs with format determined by lff
func PrintLogFileReport(reader io.Reader, f LogFormatFunc) {
	dec := json.NewDecoder(reader)

	for {
		e, isEOF, err := decodeLogEntry(dec)
		if err != nil {
			fmt.Println("Error decoding log entry:", err.Error())
			break
		}
		if isEOF {
			break
		}
		if e == nil {
			continue
		}

		if entryStr := f(e); entryStr != "" {
			fmt.Println(entryStr)
		}
	}

	return
}

func decodeLogEntry(decoder *json.Decoder) (*LogEntry, bool, error) {
	var decoded LogEntry
	if err := decoder.Decode(&decoded); err != nil {
		if err == io.EOF {
			return &decoded, true, nil
		}
		return nil, false, err
	}
	return &decoded, false, nil
}

// Handle dry run printing
const drCode = "DryRunOperation"

func isDryRun(err error) bool {
	aerr, ok := err.(awserr.Error)
	return ok && aerr.Code() == drCode
}

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
	case arn.EC2CustomerGatewayRType:
		return &EC2CustomerGatewayDeleter{ResourceType: t}
	case arn.EC2EIPRType:
		return &EC2ElasticIPAllocationDeleter{ResourceType: t}
	case arn.EC2EIPAssociationRType:
		return &EC2ElasticIPAssocationDeleter{ResourceType: t}
	case arn.EC2InstanceRType:
		return &EC2InstanceDeleter{ResourceType: t}
	case arn.EC2InternetGatewayAttachmentRType:
		return &EC2InternetGatewayAttachmentDeleter{ResourceType: t}
	case arn.EC2InternetGatewayRType:
		return &EC2InternetGatewayDeleter{ResourceType: t}
	case arn.EC2NatGatewayRType:
		return &EC2NatGatewayDeleter{ResourceType: t}
	case arn.EC2NetworkACLRType:
		return &EC2NetworkACLDeleter{ResourceType: t}
	case arn.EC2NetworkInterfaceRType:
		return &EC2NetworkInterfaceDeleter{ResourceType: t}
	case arn.EC2NetworkInterfaceAttachmentRType:
		return &EC2NetworkInterfaceAttachmentDeleter{ResourceType: t}
	case arn.EC2RouteTableAssociationRType:
		return &EC2RouteTableAssociationDeleter{ResourceType: t}
	case arn.EC2RouteTableRType:
		return &EC2RouteTableDeleter{ResourceType: t}
	case arn.EC2SecurityGroupRType:
		return &EC2SecurityGroupDeleter{ResourceType: t}
	case arn.EC2SubnetRType:
		return &EC2SubnetDeleter{ResourceType: t}
	case arn.EC2VolumeRType:
		return &EC2VolumeDeleter{ResourceType: t}
	case arn.EC2VPCCIDRAssociationRType:
		return &EC2VPCCIDRBlockAssociationDeleter{ResourceType: t}
	case arn.EC2VPCRType:
		return &EC2VPCDeleter{ResourceType: t}
	case arn.EC2VPNGatewayRType:
		return &EC2VPNGatewayDeleter{ResourceType: t}
	case arn.EC2VPNConnectionRType:
		return &EC2VPNConnectionDeleter{ResourceType: t}
	case arn.ElasticLoadBalancingLoadBalancerRType:
		return &ElasticLoadBalancingLoadBalancerDeleter{ResourceType: t}
	case arn.IAMInstanceProfileRType:
		return &IAMInstanceProfileDeleter{ResourceType: t}
	case arn.IAMRoleRType:
		return &IAMRoleDeleter{ResourceType: t}
	case arn.IAMPolicyRType:
		return &IAMRolePolicyDeleter{ResourceType: t}
	case arn.Route53HostedZoneRType:
		return &Route53HostedZoneDeleter{ResourceType: t}
	case arn.S3BucketRType:
		return &S3BucketDeleter{ResourceType: t}
	}

	fmt.Printf("Resource type %s does not implement a ResourceDeleter\n", t)
	return nil
}

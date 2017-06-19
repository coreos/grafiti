package deleter

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/coreos/grafiti/arn"
)

// EC2CustomerGatewayDeleter represents a collection of AWS EC2 customer gateways
type EC2CustomerGatewayDeleter struct {
	Client        ec2iface.EC2API
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *EC2CustomerGatewayDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// AddResourceNames adds EC2 customer gateway names to ResourceNames
func (rd *EC2CustomerGatewayDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes customer gateways from AWS
func (rd *EC2CustomerGatewayDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	fmtStr := "Deleted EC2 CustomerGateway"
	if cfg.DryRun {
		fmtStr = drStr + " " + fmtStr
	}

	if rd.Client == nil {
		rd.Client = ec2.New(setUpAWSSession())
	}

	var params *ec2.DeleteCustomerGatewayInput
	for _, n := range rd.ResourceNames {
		params = &ec2.DeleteCustomerGatewayInput{
			CustomerGatewayId: n.AWSString(),
			DryRun:            aws.Bool(cfg.DryRun),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.Client.DeleteCustomerGatewayWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.EC2CustomerGatewayRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, n)
	}

	return nil
}

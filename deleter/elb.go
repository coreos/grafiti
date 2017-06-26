package deleter

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elb/elbiface"
	"github.com/coreos/grafiti/arn"
)

// ElasticLoadBalancingLoadBalancerDeleter represents a collection of AWS elastic load balancers
type ElasticLoadBalancingLoadBalancerDeleter struct {
	Client        elbiface.ELBAPI
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *ElasticLoadBalancingLoadBalancerDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "ResourceNames": %v}`, rd.ResourceType, rd.ResourceNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *ElasticLoadBalancingLoadBalancerDeleter) GetClient() elbiface.ELBAPI {
	if rd.Client == nil {
		rd.Client = elb.New(setUpAWSSession())
	}
	return rd.Client
}

// AddResourceNames adds elastic load balancer names to ResourceNames
func (rd *ElasticLoadBalancingLoadBalancerDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes elastic load balancers from AWS
func (rd *ElasticLoadBalancingLoadBalancerDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	lbs, rerr := rd.RequestElasticLoadBalancers()
	if rerr != nil && !cfg.IgnoreErrors {
		return rerr
	}

	fmtStr := "Deleted ElasticLoadBalancer"

	var params *elb.DeleteLoadBalancerInput
	for _, lb := range lbs {
		if cfg.DryRun {
			fmt.Println(drStr, fmtStr, *lb.LoadBalancerName)
			continue
		}

		params = &elb.DeleteLoadBalancerInput{
			LoadBalancerName: lb.LoadBalancerName,
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteLoadBalancerWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.ElasticLoadBalancingLoadBalancerRType, arn.ResourceName(*lb.LoadBalancerName), err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, *lb.LoadBalancerName)
	}

	// Wait for ELB's to delete
	time.Sleep(time.Duration(1) * time.Minute)
	return nil
}

// RequestElasticLoadBalancers requests elastic load balancers by name from the AWS API
func (rd *ElasticLoadBalancingLoadBalancerDeleter) RequestElasticLoadBalancers() ([]*elb.LoadBalancerDescription, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	params := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: rd.ResourceNames.AWSStringSlice(),
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeLoadBalancersWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	return resp.LoadBalancerDescriptions, nil
}

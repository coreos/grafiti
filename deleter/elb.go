package deleter

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elb/elbiface"
	"github.com/coreos/grafiti/arn"
)

func isELBNotFoundError(err error) bool {
	aerr, ok := err.(awserr.Error)
	return ok && aerr.Code() == elb.ErrCodeAccessPointNotFoundException
}

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
		nameStr := aws.StringValue(lb.LoadBalancerName)

		if cfg.DryRun {
			fmt.Println(drStr, fmtStr, nameStr)
			continue
		}

		params = &elb.DeleteLoadBalancerInput{
			LoadBalancerName: lb.LoadBalancerName,
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteLoadBalancerWithContext(ctx, params)
		if err != nil {
			cfg.logRequestError(arn.ElasticLoadBalancingLoadBalancerRType, nameStr, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		cfg.logRequestSuccess(arn.ElasticLoadBalancingLoadBalancerRType, nameStr)
		fmt.Println(fmtStr, nameStr)
	}

	return nil
}

// RequestElasticLoadBalancers requests elastic load balancers by name from the AWS API
func (rd *ElasticLoadBalancingLoadBalancerDeleter) RequestElasticLoadBalancers() ([]*elb.LoadBalancerDescription, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	var elbs []*elb.LoadBalancerDescription
	// Requesting a batch of resources that contains at least one already deleted
	// resource will return an error and no resources. To guard against this,
	// request them one by one
	for _, name := range rd.ResourceNames {
		var err error
		if elbs, err = rd.requestElasticLoadBalancer(name, elbs); err != nil && !isELBNotFoundError(err) {
			return elbs, err
		}
	}

	return elbs, nil
}

func (rd *ElasticLoadBalancingLoadBalancerDeleter) requestElasticLoadBalancer(rn arn.ResourceName, elbs []*elb.LoadBalancerDescription) ([]*elb.LoadBalancerDescription, error) {
	params := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{rn.AWSString()},
	}

	ctx := aws.BackgroundContext()
	resp, err := rd.GetClient().DescribeLoadBalancersWithContext(ctx, params)
	if err != nil {
		fmt.Printf("{\"error\": \"%s\"}\n", err)
		return elbs, err
	}

	elbs = append(elbs, resp.LoadBalancerDescriptions...)

	return elbs, nil
}

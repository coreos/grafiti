package deleter

import (
	"fmt"

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

	return nil
}

// RequestElasticLoadBalancers requests elastic load balancers by name from the AWS API
func (rd *ElasticLoadBalancingLoadBalancerDeleter) RequestElasticLoadBalancers() ([]*elb.LoadBalancerDescription, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	size, chunk := len(rd.ResourceNames), 20
	elbs := make([]*elb.LoadBalancerDescription, 0)
	var err error
	// Can only filter in batches of 20
	for i := 0; i < size; i += chunk {
		stop := CalcChunk(i, size, chunk)
		elbs, err = rd.requestElasticLoadBalancers(rd.ResourceNames[i:stop], elbs)
		if err != nil {
			return elbs, err
		}
	}

	return elbs, nil
}

// Requesting elastic load balancers using filters prevents API errors caused by
// requesting non-existent elastic load balancers
func (rd *ElasticLoadBalancingLoadBalancerDeleter) requestElasticLoadBalancers(chunk arn.ResourceNames, elbs []*elb.LoadBalancerDescription) ([]*elb.LoadBalancerDescription, error) {
	params := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: chunk.AWSStringSlice(),
		PageSize:          aws.Int64(50),
	}

	for {
		ctx := aws.BackgroundContext()
		resp, err := rd.GetClient().DescribeLoadBalancersWithContext(ctx, params)
		if err != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", err)
			return elbs, err
		}

		elbs = append(elbs, resp.LoadBalancerDescriptions...)

		if resp.NextMarker == nil || *resp.NextMarker == "" {
			break
		}

		params.Marker = resp.NextMarker
	}

	return elbs, nil
}

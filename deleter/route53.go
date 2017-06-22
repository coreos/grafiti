package deleter

import (
	"fmt"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/coreos/grafiti/arn"
)

// Route53HostedZoneDeleter represents an AWS route53 hosted zone
type Route53HostedZoneDeleter struct {
	Client        route53iface.Route53API
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *Route53HostedZoneDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// AddResourceNames adds route53 hosted zone names to Names
func (rd *Route53HostedZoneDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes hosted zones from AWS
// NOTE: must delete all non-default resource record sets before deleting a
// hosted zone. Will receive HostedZoneNotEmpty otherwise
func (rd *Route53HostedZoneDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	rrsDeleter := Route53ResourceRecordSetDeleter{ResourceNames: rd.ResourceNames}
	if rerr := rrsDeleter.DeleteResources(cfg); rerr != nil {
		return rerr
	}

	fmtStr := "Deleted Route53 HostedZone"
	if cfg.DryRun {
		for _, id := range rd.ResourceNames {
			fmt.Println(drStr, fmtStr, id)
		}
		return nil
	}

	if rd.Client == nil {
		rd.Client = route53.New(setUpAWSSession())
	}

	var params *route53.DeleteHostedZoneInput
	for _, n := range rd.ResourceNames {
		params = &route53.DeleteHostedZoneInput{
			Id: n.AWSString(),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.Client.DeleteHostedZoneWithContext(ctx, params)
		if err != nil {
			cfg.logDeleteError(arn.Route53HostedZoneRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		fmt.Println(fmtStr, n)
	}
	return nil
}

// DeletePrivateRoute53HostedZones deletes only private hosted zones
func (rd *Route53HostedZoneDeleter) DeletePrivateRoute53HostedZones(cfg *DeleteConfig) error {
	hzs, err := rd.RequestRoute53HostedZones()
	if err != nil && !cfg.IgnoreErrors {
		return err
	}
	if len(hzs) == 0 {
		return nil
	}

	privIDs := make(arn.ResourceNames, 0)
	for _, hz := range hzs {
		if *hz.Config.PrivateZone {
			hzSplit := strings.Split(*hz.Id, "/hostedzone/")
			if len(hzSplit) != 2 {
				continue
			}
			privIDs = append(privIDs, arn.ResourceName(hzSplit[1]))
		}
	}

	rd.ResourceNames = privIDs

	return rd.DeleteResources(cfg)
}

// RequestRoute53HostedZones requests resources from the AWS API and returns route53
// hosted zones by names
func (rd *Route53HostedZoneDeleter) RequestRoute53HostedZones() ([]*route53.HostedZone, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	// Only way to filter is iteratively (no query parameters)
	want := map[arn.ResourceName]struct{}{}
	for _, id := range rd.ResourceNames {
		if _, ok := want["/hostedzone/"+id]; !ok {
			want["/hostedzone/"+id] = struct{}{}
		}
	}

	wantedHZs := make([]*route53.HostedZone, 0)
	hzs, err := rd.RequestAllRoute53HostedZones()
	if err != nil {
		return nil, err
	}

	for _, hz := range hzs {
		if _, ok := want[arn.ResourceName(*hz.Id)]; ok {
			wantedHZs = append(wantedHZs, hz)
		}
	}

	return wantedHZs, nil
}

// RequestAllRoute53HostedZones retrieves a list of all hosted zones
func (rd *Route53HostedZoneDeleter) RequestAllRoute53HostedZones() ([]*route53.HostedZone, error) {
	hzs := make([]*route53.HostedZone, 0)
	params := &route53.ListHostedZonesInput{
		MaxItems: aws.String("100"),
	}

	if rd.Client == nil {
		rd.Client = route53.New(setUpAWSSession())
	}

	for {
		ctx := aws.BackgroundContext()
		resp, err := rd.Client.ListHostedZonesWithContext(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, hz := range resp.HostedZones {
			hzs = append(hzs, hz)
		}

		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}

		params.Marker = resp.NextMarker
	}

	return hzs, nil
}

// Route53ResourceRecordSetDeleter represents an AWS route53 resource record set
type Route53ResourceRecordSetDeleter struct {
	Client        route53iface.Route53API
	ResourceType  arn.ResourceType
	ResourceNames arn.ResourceNames
}

func (rd *Route53ResourceRecordSetDeleter) String() string {
	return fmt.Sprintf(`{"Type": "%s", "Names": %v}`, rd.ResourceType, rd.ResourceNames)
}

// AddResourceNames adds route53 resource record set names to Names
func (rd *Route53ResourceRecordSetDeleter) AddResourceNames(ns ...arn.ResourceName) {
	rd.ResourceNames = append(rd.ResourceNames, ns...)
}

// DeleteResources deletes route53 resource record sets from AWS
func (rd *Route53ResourceRecordSetDeleter) DeleteResources(cfg *DeleteConfig) error {
	if len(rd.ResourceNames) == 0 {
		return nil
	}

	rrsMap, rerr := rd.RequestRoute53ResourceRecordSets()
	if rerr != nil && !cfg.IgnoreErrors {
		return rerr
	}
	if len(rrsMap) == 0 {
		return nil
	}

	fmtStr := "Deleted Route53 ResourceRecordSet"
	if cfg.DryRun {
		for hz, rrss := range rrsMap {
			for _, rrs := range rrss {
				fmt.Printf("%s %s %s (HZ %s)\n", drStr, fmtStr, rrs, hz)
			}
		}
		return nil
	}

	if rd.Client == nil {
		rd.Client = route53.New(setUpAWSSession())
	}

	var (
		changes []*route53.Change
		params  *route53.ChangeResourceRecordSetsInput
	)
	for hz, rrss := range rrsMap {
		changes = make([]*route53.Change, 0, len(rrss))
		for _, rrs := range rrss {
			changes = append(changes, &route53.Change{
				Action:            aws.String(route53.ChangeActionDelete),
				ResourceRecordSet: rrs,
			})
		}

		params = &route53.ChangeResourceRecordSetsInput{
			ChangeBatch:  &route53.ChangeBatch{Changes: changes},
			HostedZoneId: hz.AWSString(),
		}

		// Prevent throttling
		time.Sleep(cfg.BackoffTime)

		ctx := aws.BackgroundContext()
		_, err := rd.Client.ChangeResourceRecordSetsWithContext(ctx, params)
		if err != nil {
			for _, rrs := range rrss {
				cfg.logDeleteError(arn.Route53ResourceRecordSetRType, arn.ResourceName(*rrs.Name), err, logrus.Fields{
					"parent_resource_type": arn.Route53HostedZoneRType,
					"parent_resource_name": hz.String(),
				})
			}
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		for _, rrs := range rrss {
			fmt.Printf("%s %s (HZ %s)\n", fmtStr, *rrs.Name, hz)
		}
	}

	return nil
}

// RequestRoute53ResourceRecordSets requests route53 resource record sets by
// hosted zone names from the AWS API and returns a map of hosted zones to
// resource record sets
func (rd *Route53ResourceRecordSetDeleter) RequestRoute53ResourceRecordSets() (map[arn.ResourceName][]*route53.ResourceRecordSet, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	rrsMap := make(map[arn.ResourceName][]*route53.ResourceRecordSet)
	var params *route53.ListResourceRecordSetsInput

	if rd.Client == nil {
		rd.Client = route53.New(setUpAWSSession())
	}

	for _, hz := range rd.ResourceNames {
		params = &route53.ListResourceRecordSetsInput{
			HostedZoneId: hz.AWSString(),
			MaxItems:     aws.String("100"),
		}

		for {
			ctx := aws.BackgroundContext()
			resp, err := rd.Client.ListResourceRecordSetsWithContext(ctx, params)
			if err != nil {
				return nil, err
			}

			for _, rrs := range resp.ResourceRecordSets {
				// Cannot delete NS/SOA type record sets
				if *rrs.Type == "NS" || *rrs.Type == "SOA" {
					continue
				}

				rrsMap[hz] = append(rrsMap[hz], rrs)
			}

			if resp.IsTruncated == nil || !*resp.IsTruncated {
				break
			}

			params.StartRecordIdentifier = resp.NextRecordIdentifier
			params.StartRecordType = resp.NextRecordType
			params.StartRecordName = resp.NextRecordName
		}
	}

	return rrsMap, nil
}

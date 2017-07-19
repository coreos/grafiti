package deleter

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/coreos/grafiti/arn"
	"github.com/sirupsen/logrus"
)

const (
	hostedZonePrefix = "/hostedzone/"
	recordSetNSType  = "NS"
	recordSetSOAType = "SOA"
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

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *Route53HostedZoneDeleter) GetClient() route53iface.Route53API {
	if rd.Client == nil {
		rd.Client = route53.New(setUpAWSSession())
	}
	return rd.Client
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

	fmtStr := "Deleted Route53 HostedZone"

	var (
		params     *route53.DeleteHostedZoneInput
		rrsDeleter Route53ResourceRecordSetDeleter
	)
	for _, n := range rd.ResourceNames {
		if cfg.DryRun {
			fmt.Println(drStr, fmtStr, n)
			continue
		}

		rrsDeleter = Route53ResourceRecordSetDeleter{HostedZoneID: n}
		if rerr := rrsDeleter.DeleteResources(cfg); rerr != nil {
			return rerr
		}

		params = &route53.DeleteHostedZoneInput{
			Id: n.AWSString(),
		}

		ctx := aws.BackgroundContext()
		_, err := rd.GetClient().DeleteHostedZoneWithContext(ctx, params)
		if err != nil {
			cfg.logRequestError(arn.Route53HostedZoneRType, n, err)
			if cfg.IgnoreErrors {
				continue
			}
			return err
		}

		cfg.logRequestSuccess(arn.Route53HostedZoneRType, n)
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

	rd.ResourceNames = filterPrivateHostedZones(hzs)

	return rd.DeleteResources(cfg)
}

// SplitHostedZoneID splits a hosted zones' AWS ID, which might be prefixed with
// "/hostedzone/", into the actual ID (the suffix)
func SplitHostedZoneID(hzID string) arn.ResourceName {
	if strings.HasPrefix(hzID, hostedZonePrefix) {
		hzSplit := strings.Split(hzID, hostedZonePrefix)
		if len(hzSplit) < 2 {
			return ""
		}
		return arn.ResourceName(hzSplit[1])
	}
	return arn.ResourceName(hzID)
}

func filterPrivateHostedZones(hzs []*route53.HostedZone) arn.ResourceNames {
	privateHZs := make(arn.ResourceNames, 0)
	for _, hz := range hzs {
		if hz.Config.PrivateZone != nil && *hz.Config.PrivateZone {
			privateHZs = append(privateHZs, SplitHostedZoneID(*hz.Id))
		}
	}
	return privateHZs
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

	for {
		ctx := aws.BackgroundContext()
		resp, err := rd.GetClient().ListHostedZonesWithContext(ctx, params)
		if err != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", err)
			return hzs, err
		}

		hzs = append(hzs, resp.HostedZones...)

		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}

		params.Marker = resp.NextMarker
	}

	return hzs, nil
}

// Route53ResourceRecordSetDeleter represents an AWS route53 resource record set
type Route53ResourceRecordSetDeleter struct {
	Client             route53iface.Route53API
	ResourceType       arn.ResourceType
	HostedZoneID       arn.ResourceName
	ResourceRecordSets []*route53.ResourceRecordSet
}

func (rd *Route53ResourceRecordSetDeleter) String() string {
	rrsNames := make([]string, 0, len(rd.ResourceRecordSets))
	for _, rrs := range rd.ResourceRecordSets {
		rrsNames = append(rrsNames, aws.StringValue(rrs.Name))
	}
	return fmt.Sprintf(`{"Type": "%s", "HostedZoneID": "%s", "ResourceRecordSetNames": %v}`, rd.ResourceType, rd.HostedZoneID, rrsNames)
}

// GetClient returns an AWS Client, and initalizes one if one has not been
func (rd *Route53ResourceRecordSetDeleter) GetClient() route53iface.Route53API {
	if rd.Client == nil {
		rd.Client = route53.New(setUpAWSSession())
	}
	return rd.Client
}

// DeleteResources deletes route53 resource record sets from AWS
func (rd *Route53ResourceRecordSetDeleter) DeleteResources(cfg *DeleteConfig) error {
	if rd.HostedZoneID == "" {
		return nil
	}

	rrss, rerr := rd.RequestRoute53ResourceRecordSets()
	if rerr != nil && !cfg.IgnoreErrors {
		return rerr
	}
	if len(rrss) == 0 {
		return nil
	}

	fmtStr := "Deleted Route53 ResourceRecordSet"

	changes := make([]*route53.Change, 0, len(rrss))
	for _, rrs := range rrss {
		if cfg.DryRun {
			fmt.Printf("%s %s %s from HostedZone %s\n", drStr, fmtStr, *rrs.Name, rd.HostedZoneID)
			continue
		}

		changes = append(changes, &route53.Change{
			Action:            aws.String(route53.ChangeActionDelete),
			ResourceRecordSet: rrs,
		})
	}

	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  &route53.ChangeBatch{Changes: changes},
		HostedZoneId: rd.HostedZoneID.AWSString(),
	}

	ctx := aws.BackgroundContext()
	_, err := rd.GetClient().ChangeResourceRecordSetsWithContext(ctx, params)
	if err != nil {
		for _, rrs := range rrss {
			cfg.logRequestError(arn.Route53ResourceRecordSetRType, *rrs.Name, err, logrus.Fields{
				"parent_resource_type": arn.Route53HostedZoneRType,
				"parent_resource_name": rd.HostedZoneID,
			})
		}
		if cfg.IgnoreErrors {
			return nil
		}
		return err
	}

	for _, rrs := range rrss {
		cfg.logRequestSuccess(arn.Route53ResourceRecordSetRType, *rrs.Name, logrus.Fields{
			"parent_resource_type": arn.Route53HostedZoneRType,
			"parent_resource_name": rd.HostedZoneID,
		})
		fmt.Printf("%s %s from HostedZone %s\n", fmtStr, *rrs.Name, rd.HostedZoneID)
	}

	return nil
}

// RequestRoute53ResourceRecordSets requests route53 resource record sets by
// hosted zone names from the AWS API and returns a map of hosted zones to
// resource record sets
func (rd *Route53ResourceRecordSetDeleter) RequestRoute53ResourceRecordSets() ([]*route53.ResourceRecordSet, error) {
	if rd.HostedZoneID == "" {
		return nil, nil
	}

	recordSets := make([]*route53.ResourceRecordSet, 0)

	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId: rd.HostedZoneID.AWSString(),
		MaxItems:     aws.String("100"),
	}

	for {
		ctx := aws.BackgroundContext()
		resp, err := rd.GetClient().ListResourceRecordSetsWithContext(ctx, params)
		if err != nil {
			fmt.Printf("{\"error\": \"%s\"}\n", err)
			return recordSets, err
		}

		for _, rrs := range resp.ResourceRecordSets {
			if !isTypeNSorSOA(aws.StringValue(rrs.Type)) {
				recordSets = append(recordSets, rrs)
			}
		}

		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}

		params.StartRecordIdentifier = resp.NextRecordIdentifier
		params.StartRecordType = resp.NextRecordType
		params.StartRecordName = resp.NextRecordName
	}

	return recordSets, nil
}

// Cannot delete NS/SOA type record sets
func isTypeNSorSOA(t string) bool {
	return t == recordSetNSType || t == recordSetSOAType
}

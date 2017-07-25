package deleter

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/coreos/grafiti/arn"
	"github.com/sirupsen/logrus"
)

const (
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
		params           *route53.DeleteHostedZoneInput
		recordSetDeleter Route53ResourceRecordSetDeleter
	)
	for _, n := range rd.ResourceNames {
		// Delete resource record sets from hosted zone
		recordSetDeleter = Route53ResourceRecordSetDeleter{HostedZoneID: n}
		recordSets, rerr := recordSetDeleter.RequestRoute53ResourceRecordSets()
		if rerr != nil && !cfg.IgnoreErrors {
			return rerr
		}
		recordSetDeleter.ResourceRecordSets = recordSets
		if err := recordSetDeleter.DeleteResources(cfg); err != nil {
			return err
		}

		// Delete any record sets created in a public hosted zone
		if err := recordSetDeleter.deleteRecordSetsFromPublicHostedZones(cfg); err != nil {
			return err
		}

		if cfg.DryRun {
			fmt.Println(drStr, fmtStr, n)
			continue
		}

		// Delete hosted zones
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

// RequestRoute53HostedZones requests resources from the AWS API and returns
// hosted zones by names
func (rd *Route53HostedZoneDeleter) RequestRoute53HostedZones() ([]*route53.HostedZone, error) {
	if len(rd.ResourceNames) == 0 {
		return nil, nil
	}

	// Only way to filter is iteratively (no query parameters)
	want := map[arn.ResourceName]struct{}{}
	for _, id := range rd.ResourceNames {
		if _, ok := want[arn.HostedZonePrefix+id]; !ok {
			want[arn.HostedZonePrefix+id] = struct{}{}
		}
	}

	wantedHZs := make([]*route53.HostedZone, 0)
	hzs, err := rd.RequestAllRoute53HostedZones()
	if err != nil {
		return nil, err
	}

	for _, hz := range hzs {
		if _, ok := want[arn.ToResourceName(hz.Id)]; ok {
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
			logger.Errorln(err)
			return hzs, err
		}

		hzs = append(hzs, resp.HostedZones...)

		if !aws.BoolValue(resp.IsTruncated) {
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

	// Cache hosted zones to avoid requesting all hosted zones multiple times
	cachedPublicHostedZones []*route53.HostedZone
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
	if rd.HostedZoneID == "" || len(rd.ResourceRecordSets) == 0 {
		return nil
	}

	fmtStr := "Deleted Route53 ResourceRecordSet"

	if cfg.DryRun {
		for _, rrs := range rd.ResourceRecordSets {
			fmt.Printf("%s %s %s from HostedZone %s\n", drStr, fmtStr, aws.StringValue(rrs.Name), rd.HostedZoneID)
		}
		return nil
	}

	changes := make([]*route53.Change, 0, len(rd.ResourceRecordSets))
	for _, rrs := range rd.ResourceRecordSets {
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
		for _, rrs := range rd.ResourceRecordSets {
			cfg.logRequestError(arn.Route53ResourceRecordSetRType, aws.StringValue(rrs.Name), err, logrus.Fields{
				"parent_resource_type": arn.Route53HostedZoneRType,
				"parent_resource_name": rd.HostedZoneID,
			})
		}
		if cfg.IgnoreErrors {
			return nil
		}
		return err
	}

	for _, rrs := range rd.ResourceRecordSets {
		nameStr := aws.StringValue(rrs.Name)
		cfg.logRequestSuccess(arn.Route53ResourceRecordSetRType, nameStr, logrus.Fields{
			"parent_resource_type": arn.Route53HostedZoneRType,
			"parent_resource_name": rd.HostedZoneID,
		})
		fmt.Printf("%s %s from HostedZone %s\n", fmtStr, nameStr, rd.HostedZoneID)
	}

	return nil
}

// Delete resource record sets that correspond to a private hosted zone with
// ID 'hzID' from any public hosted zones
func (rd *Route53ResourceRecordSetDeleter) deleteRecordSetsFromPublicHostedZones(cfg *DeleteConfig) error {
	want := createResourceNameMapFromHostedZones(rd.ResourceRecordSets)

	var (
		err error
		hzs []*route53.HostedZone
	)
	hzDel := new(Route53HostedZoneDeleter)
	if len(rd.cachedPublicHostedZones) == 0 {
		if hzs, err = hzDel.RequestAllRoute53HostedZones(); err != nil && !cfg.IgnoreErrors {
			return err
		}
		for _, hz := range hzs {
			if !isPrivateHostedZone(hz) {
				rd.cachedPublicHostedZones = append(rd.cachedPublicHostedZones, hz)
			}
		}
	}

	recordSetDel := new(Route53ResourceRecordSetDeleter)
	var recordSets []*route53.ResourceRecordSet
	for _, hz := range rd.cachedPublicHostedZones {
		recordSetDel.HostedZoneID = arn.SplitHostedZoneID(aws.StringValue(hz.Id))
		if recordSets, err = recordSetDel.RequestRoute53ResourceRecordSets(); err != nil && !cfg.IgnoreErrors {
			return err
		}

		for _, recordSet := range recordSets {
			if _, ok := want[aws.StringValue(recordSet.Name)]; ok {
				recordSetDel.ResourceRecordSets = append(recordSetDel.ResourceRecordSets, recordSet)
			}
		}

		if err = recordSetDel.DeleteResources(cfg); err != nil {
			return err
		}

		recordSetDel.ResourceRecordSets = nil
	}
	rd = nil

	return nil
}

func createResourceNameMapFromHostedZones(recordSets []*route53.ResourceRecordSet) map[string]struct{} {
	want := map[string]struct{}{}
	for _, recordSet := range recordSets {
		setName := aws.StringValue(recordSet.Name)
		if _, ok := want[setName]; !ok {
			want[setName] = struct{}{}
		}
	}
	return want
}

func isPrivateHostedZone(hz *route53.HostedZone) bool {
	return aws.BoolValue(hz.Config.PrivateZone)
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
			logger.Errorln(err)
			return recordSets, err
		}

		for _, rrs := range resp.ResourceRecordSets {
			if !isTypeNSorSOA(aws.StringValue(rrs.Type)) {
				recordSets = append(recordSets, rrs)
			}
		}

		if !aws.BoolValue(resp.IsTruncated) {
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

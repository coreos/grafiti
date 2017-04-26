// Copyright © 2017 grafiti/predator authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	jq "github.com/threatgrid/jqpipe-go"
	"github.com/tidwall/gjson"
)

var inputFile string

func init() {
	RootCmd.AddCommand(parseCmd)
	parseCmd.PersistentFlags().StringVarP(&inputFile, "inputFile", "i", "", "CloudTrail Log input")
}

var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "parse and output resources by reading CloudTrail logs",
	Long:  "Parse a CloudTrail Log and output resources. By default, talks to the configured aws account and reads directly from CloudTrail.",
	RunE:  runParseCommand,
}

func runParseCommand(cmd *cobra.Command, args []string) error {
	if inputFile != "" {
		return parseFromFile(inputFile)
	}
	if err := parseFromCloudTrail(); err != nil {
		return err
	}
	return nil
}

type CloudTrailLogFile struct {
	Events []*cloudtrail.Event `json:"Records"`
}

func parseFromFile(logFileName string) error {
	raw, err := ioutil.ReadFile(logFileName)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	var logFile CloudTrailLogFile
	if err := json.Unmarshal(raw, &logFile); err != nil {
		return err
	}
	printEvents(logFile.Events)

	return nil
}

type NotTaggedFilter struct {
	Type string `json:"type"`
}

func parseFromCloudTrail() error {
	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(viper.GetString("grafiti.az")),
		},
	))
	svc := cloudtrail.New(sess)
	params := &cloudtrail.LookupEventsInput{
		EndTime: aws.Time(time.Now()),
		LookupAttributes: []*cloudtrail.LookupAttribute{
			{
				AttributeKey:   aws.String("ResourceType"),
				AttributeValue: aws.String(viper.GetString("grafiti.resourceType")),
			},
		},
		MaxResults: aws.Int64(50),
		StartTime:  aws.Time(time.Now().Add(time.Duration(viper.GetInt("grafiti.hours")) * time.Hour)),
	}

	for {
		req, resp := svc.LookupEventsRequest(params)
		if err := req.Send(); err != nil {
			return err
		}
		printEvents(resp.Events)
		if resp.NextToken == nil {
			break
		}
		params.NextToken = resp.NextToken
	}

	return nil
}

type OutputWithEvent struct {
	Event           *cloudtrail.Event
	TaggingMetadata *TaggingMetadata
	Tags            map[string]string
}

type Output struct {
	TaggingMetadata *TaggingMetadata
	Tags            map[string]string
}

func filterOutput(output []byte) []byte {
	for _, filter := range viper.GetStringSlice("grafiti.filterPatterns") {
		results, err := jq.Eval(string(output), filter)
		if err != nil {
			return nil
		}
		if len(results) == 0 {
			return nil
		}
		json, _ := results[0].MarshalJSON()
		if string(json) != "true" {
			return nil
		}
	}
	return output
}

func printEvents(events []*cloudtrail.Event) {
	for _, e := range events {
		parsedEvent := gjson.Parse(*e.CloudTrailEvent)
		printEvent(e, parsedEvent)
	}
}

func printEvent(event *cloudtrail.Event, parsedEvent gjson.Result) {
	includeEvent := viper.GetBool("grafiti.includeEvent")
	for _, r := range event.Resources {
		tags := getTags(event)
		output := getOutput(includeEvent, event, tags, &TaggingMetadata{
			ResourceName: *r.ResourceName,
			ResourceType: *r.ResourceType,
			ResourceARN:  ARNForResource(r, parsedEvent),
			CreatorARN:   parsedEvent.Get("userIdentity.arn").Str,
			CreatorName:  parsedEvent.Get("userIdentity.userName").Str,
		})
		resourceJson, err := json.Marshal(output)
		resourceJson = filterOutput(resourceJson)
		if resourceJson == nil {
			continue
		}
		if err != nil {
			fmt.Println(fmt.Sprintf(`{"error": "%s"}`, err))
		}
		fmt.Println(string(resourceJson))
	}
}

func getTags(event *cloudtrail.Event) map[string]string {
	allTags := make(map[string]string)
	for _, tagPattern := range viper.GetStringSlice("grafiti.tagPatterns") {
		results, err := jq.Eval(*event.CloudTrailEvent, tagPattern)
		if err != nil {
			fmt.Println(fmt.Sprintf(`{"error": "%s"}`, err))
			return nil
		}
		for _, r := range results {
			rBytes, err := r.MarshalJSON()
			if err != nil {
				break
			}
			tagMap, ok := gjson.Parse(string(rBytes)).Value().(map[string]interface{})

			if !ok {
				break
			}

			for k, v := range tagMap {
				allTags[k] = v.(string)
			}
		}
	}
	return allTags
}

func getOutput(includeEvent bool, event *cloudtrail.Event, tags map[string]string, taggingMetadata *TaggingMetadata) interface{} {
	if includeEvent {
		return OutputWithEvent{
			Event:           event,
			TaggingMetadata: taggingMetadata,
			Tags:            tags,
		}
	}
	return Output{
		TaggingMetadata: taggingMetadata,
		Tags:            tags,
	}
}

func ServiceNameForResource(resource *cloudtrail.Resource) string {
	switch {
	case strings.HasPrefix(*resource.ResourceType, "AWS::EC2::"):
		return "ec2"
	case strings.HasPrefix(*resource.ResourceType, "AWS::AutoScaling::"):
		return "autoscaling"
	case strings.HasPrefix(*resource.ResourceType, "AWS::ACM::"):
		return "acm"
	case strings.HasPrefix(*resource.ResourceType, "AWS::CloudTrail::"):
		return "cloudtrail"
	case strings.HasPrefix(*resource.ResourceType, "AWS::CodePipeline::"):
		return "codepipeline"
	case strings.HasPrefix(*resource.ResourceType, "AWS::ElasticLoadBalancing::"):
		return "elasticloadbalancing"
	case strings.HasPrefix(*resource.ResourceType, "AWS::IAM::"):
		return "iam"
	case strings.HasPrefix(*resource.ResourceType, "AWS::Redshift::"):
		return "redshift"
	case strings.HasPrefix(*resource.ResourceType, "AWS::RDS::"):
		return "rds"
	case strings.HasPrefix(*resource.ResourceType, "AWS::S3::"):
		return "s3"
	}
	return ""
}

func ARNForResource(resource *cloudtrail.Resource, parsedEvent gjson.Result) string {
	region := parsedEvent.Get("awsRegion").Str
	accountID := parsedEvent.Get("userIdentity.accountId").Str
	ARNPrefix := fmt.Sprintf("arn:aws:%s:%s:%s", ServiceNameForResource(resource), region, accountID)

	switch *resource.ResourceType {
	case "AWS::AutoScaling::AutoScalingGroup":
		//arn:aws:autoscaling:region:account-id:scalingPolicy:policyid:autoScalingGroupName/groupfriendlyname:policyname/policyfriendlyname
		break
	case "AWS::AutoScaling::LaunchConfiguration":
		//arn:aws:autoscaling:region:account-id:launchConfiguration:launchconfigid:launchConfigurationName/launchconfigfriendlyname
		break
	case "AWS::AutoScaling::ScalingPolicy":
		//arn:aws:autoscaling:region:account-id:autoScalingGroup:groupid:autoScalingGroupName/groupfriendlyname
		break
	case "AWS::AutoScaling::ScheduledAction":
		//arn:aws:autoscaling:region:account-id:scheduledUpdateGroupAction:scheduleactionid:autoScalingGroupName/autoscalinggroupfriendlyname:scheduledActionName/scheduledactionfriendlyname
		break
	case "AWS::ACM::Certificate":
		//arn:aws:acm:region:account-id:certificate/certificate-id
		break
	case "AWS::CloudTrail::Trail":
		//arn:aws:cloudtrail:region:account-id:trail/trailname
		break
	case "AWS::CodePipeline::Pipeline":
		//arn:aws:codepipeline:region:account-id:resource-specifier
		break
	case "AWS::EC2::Ami":
		//arn:aws:ec2:region::image/image-id
		imageID := parsedEvent.Get("responseElements.instancesSet.items.0.imageId")
		return fmt.Sprintf("%s::image/%s", ARNPrefix, imageID)
	case "AWS::EC2::BundleTask":
		break
	case "AWS::EC2::ConversionTask":
		break
	case "AWS::EC2::CustomerGateway":
		//arn:aws:ec2:region:account-id:customer-gateway/cgw-id
		break
	case "AWS::EC2::DHCPOptions":
		//arn:aws:ec2:region:account-id:dhcp-options/dhcp-options-id
		break
	case "AWS::EC2::EIP":
		break
	case "AWS::EC2::EIPAssociation":
		break
	case "AWS::EC2::ExportTask":
		break
	case "AWS::EC2::FlowLog":
		break
	case "AWS::EC2::Host":
		// arn:aws:ec2:region:account_id:dedicated-host/host_id
		break
	case "AWS::EC2::ImportTask":
		break
	case "AWS::EC2::Instance":
		// arn:aws:ec2:region:account-id:instance/instance-id
		return fmt.Sprintf("%s:instance/%s", ARNPrefix, *resource.ResourceName)
	case "AWS::EC2::InternetGateway":
		// arn:aws:ec2:region:account-id:internet-gateway/igw-id
		break
	case "AWS::EC2::KeyPair":
		// arn:aws:ec2:region:account-id:key-pair/key-pair-name
		break
	case "AWS::EC2::NatGateway":
		break
	case "AWS::EC2::NetworkAcl":
		// arn:aws:ec2:region:account-id:network-acl/nacl-id
		break
	case "AWS::EC2::NetworkInterface":
		// arn:aws:ec2:region:account-id:network-interface/eni-id
		break
	case "AWS::EC2::NetworkInterfaceAttachment":
		break
	case "AWS::EC2::PlacementGroup":
		//arn:aws:ec2:region:account-id:placement-group/placement-group-name
		break
	case "AWS::EC2::ReservedInstance":
		break
	case "AWS::EC2::ReservedInstancesListing":
		break
	case "AWS::EC2::ReservedInstancesModification":
		break
	case "AWS::EC2::RouteTable":
		//arn:aws:ec2:region:account-id:route-table/route-table-id
		break
	case "AWS::EC2::ScheduledInstance":
		break
	case "AWS::EC2::SecurityGroup":
		//arn:aws:ec2:region:account-id:security-group/security-group-id
		break
	case "AWS::EC2::Snapshot":
		//arn:aws:ec2:region:account-id:snapshot/snapshot-id
		break
	case "AWS::EC2::SpotFleetRequest":
		break
	case "AWS::EC2::SpotInstanceRequest":
		break
	case "AWS::EC2::Subnet":
		//arn:aws:ec2:region:account-id:subnet/subnet-id
		subnetID := parsedEvent.Get("responseElements.subnet.subnetId")
		return fmt.Sprintf("%s:subnet/%s", ARNPrefix, subnetID)
	case "AWS::EC2::SubnetNetworkAclAssociation":
		break
	case "AWS::EC2::SubnetRouteTableAssociation":
		break
	case "AWS::EC2::Volume":
		//arn:aws:ec2:region:account-id:volume/volume-id
		break
	case "AWS::EC2::VPC":
		//arn:aws:ec2:region:account-id:vpc/vpc-id
		vpcID := parsedEvent.Get("responseElements.instancesSet.items.0.vpcId")
		return fmt.Sprintf("%s:vpc/%s", ARNPrefix, vpcID)
	case "AWS::EC2::VPCEndpoint":
		break
	case "AWS::EC2::VPCPeeringConnection":
		//arn:aws:ec2:region:account-id:vpc-peering-connection/vpc-peering-connection-id
		break
	case "AWS::EC2::VPNConnection":
		//arn:aws:ec2:region:account-id:vpn-connection/vpn-id
		break
	case "AWS::EC2::VPNGateway":
		//arn:aws:ec2:region:account-id:vpn-gateway/vgw-id
		break
	case "AWS::ElasticLoadBalancing::LoadBalancer":
		//arn:aws:elasticloadbalancing:region:account-id:loadbalancer/name
		return fmt.Sprintf("%s:loadbalancer/%s", ARNPrefix, *resource.ResourceName)
	case "AWS::IAM::AccessKey":
		break
	case "AWS::IAM::AccountAlias":
		break
	case "AWS::IAM::Group":
		//arn:aws:iam::account-id:group/group-name
		break
	case "AWS::IAM::InstanceProfile":
		//arn:aws:iam::account-id:instance-profile/instance-profile-name
		break
	case "AWS::IAM::MfaDevice":
		//arn:aws:iam::account-id:mfa/virtual-device-name
		break
	case "AWS::IAM::OpenIDConnectProvider":
		//arn:aws:iam::account-id:oidc-provider/provider-name
		break
	case "AWS::IAM::Policy":
		//arn:aws:iam::account-id:policy/policy-name
		break
	case "AWS::IAM::Role":
		//arn:aws:iam::account-id:role/role-name
		break
	case "AWS::IAM::SamlProvider":
		//arn:aws:iam::account-id:saml-provider/provider-name
		break
	case "AWS::IAM::ServerCertificate":
		//arn:aws:iam::account-id:server-certificate/certificate-name
		break
	case "AWS::IAM::SigningCertificate":
		break
	case "AWS::IAM::SshPublicKey":
		break
	case "AWS::IAM::User":
		//arn:aws:iam::account-id:user/user-name
		break
	case "AWS::Redshift::Cluster":
		//arn:aws:redshift:region:account-id:cluster:clustername
		break
	case "AWS::Redshift::ClusterParameterGroup":
		//arn:aws:redshift:region:account-id:parametergroup:parametergroupname
		break
	case "AWS::Redshift::ClusterSecurityGroup":
		//arn:aws:redshift:region:account-id:securitygroup:securitygroupname
		break
	case "AWS::Redshift::ClusterSnapshot":
		//arn:aws:redshift:region:account-id:snapshot:clustername/snapshotname
		break
	case "AWS::Redshift::ClusterSubnetGroup":
		//arn:aws:redshift:region:account-id:subnetgroup:subnetgroupname
		break
	case "AWS::Redshift::EventSubscription":
		break
	case "AWS::Redshift::HsmClientCertificate":
		break
	case "AWS::Redshift::HsmConfiguration":
		break
	case "AWS::RDS::DBCluster":
		//arn:aws:rds:region:account-id:cluster:db-cluster-name
		break
	case "AWS::RDS::DBClusterOptionGroup":
		break
	case "AWS::RDS::DBClusterParameterGroup":
		//arn:aws:rds:region:account-id:cluster-pg:cluster-parameter-group-name
		break
	case "AWS::RDS::DBClusterSnapshot":
		//arn:aws:rds:region:account-id:cluster-snapshot:cluster-snapshot-name
		break
	case "AWS::RDS::DBInstance":
		//arn:aws:rds:region:account-id:db:db-instance-name
		break
	case "AWS::RDS::DBOptionGroup":
		//arn:aws:rds:region:account-id:og:option-group-name
		break
	case "AWS::RDS::DBParameterGroup":
		//arn:aws:rds:region:account-id:pg:parameter-group-name
		break
	case "AWS::RDS::DBSecurityGroup":
		//arn:aws:rds:region:account-id:secgrp:security-group-name
		break
	case "AWS::RDS::DBSnapshot":
		//arn:aws:rds:region:account-id:snapshot:snapshot-name
		break
	case "AWS::RDS::DBSubnetGroup":
		//arn:aws:rds:region:account-id:subgrp:subnet-group-name
		break
	case "AWS::RDS::EventSubscription":
		//arn:aws:rds:region:account-id:es:subscription-name
		break
	case "AWS::RDS::ReservedDBInstance":
		break
	case "AWS::S3::Bucket":
		//arn:aws:s3:::bucket_name
		break
	}
	return ""
}

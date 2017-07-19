# Grafiti

[![Build Status](https://jenkins-tectonic-installer.prod.coreos.systems/job/grafiti/badge/icon)](https://jenkins-tectonic-installer.prod.coreos.systems/job/grafiti)

Grafiti is a tool for parsing, tagging, and deleting AWS resources.

* Using a [CloudTrail](https://aws.amazon.com/cloudtrail/) trail, resource CRUD events can be parsed using `grafiti` for identifying resource information.
* Parsed data can optionally be fed through `grafiti filter --ignore-file <tag-file>`, which filters out all resources tagged with tags in `<tag-file>` from parsed data.
* Parsed data can be fed into `grafiti tag` and tagged using the AWS resource group tagging API.
* Tagged resources are retrieved using the same API during `grafiti delete`, and deleted using resource type-specific service API's.

Each sub-command can be used in a sequential pipe, or individually.

## Motivating Example

We listen to CloudTrail events, and tag created resources with a default expiration of 2 weeks and the ARN of the creating user.

Every day, we can query the resource tagging API for resources that will expire in one week, and the owners can be notified via email/Slack.

Every day, we also query for resources that have expired, and delete them.

# Installation
Ensure you have the following installed:

* [Golang](https://golang.org/dl/) 1.7+
* `jq` (see below)
* The [glide](https://glide.sh/) package manager

Retrieve and install grafiti (the binary will be in `$GOPATH/bin`):
```
go get github.com/coreos/grafiti/cmd/grafiti
```

Alternatively, if `$GOPATH/src/github.com/coreos/grafiti` is already present, simply install it:
```
go install github.com/coreos/grafiti/cmd/grafiti
```
or use the Makefile (requires `make`):
```
make install
```

## `jq` installation
`jq` is a CLI JSON parsing tool that `grafiti` uses internally to evaluate config file expressions, and must be installed before running `grafiti`. This program is quite useful for parsing `grafiti` input/output as well. You can find download instructions [here](https://stedolan.github.io/jq/download/).

# Usage

## Grafiti commands

* `grafiti parse` - Parses CloudTrail data and outputs useful information (to be consumed by `grafiti tag` or `grafiti filter`)
* `grafiti filter` - Filters `grafiti parse` output by removing resources with defined tags (to be consumed by `grafiti tag`)
* `grafiti tag` - Tags resources in AWS based on tagging rules defined in your `config.toml` file
* `grafiti delete` - Deletes resources in AWS based on tags


```
Usage:
  grafiti [flags]
  grafiti [command]

Available Commands:
  delete      Delete resources in AWS
  filter      Filter AWS resources by tag
  help        Help about any command
  parse       Parse and output resources by reading CloudTrail logs
  tag         Tag resources in AWS

Flags:
  -c, --config string   Config file (default is $HOME/.grafiti.toml)
      --debug           Enable debug logging.
      --dry-run         Output changes to stdout instead of AWS.
  -e, --ignore-errors   Continue processing even when there are API errors.

Use "grafiti [command] --help" for more information about a command.
```

## Configure aws credentials

  In order to use `grafiti`, you will need to configure your machine to talk to AWS with a `~/.aws/credentials` [file](http://docs.aws.amazon.com/cli/latest/userguide/cli-config-files.html).

```
 [default]
 aws_access_key_id = AKID1234567890
 aws_secret_access_key = MY-SECRET-KEY
```

 Alternatively, you can set the following [environment variables](http://docs.aws.amazon.com/cli/latest/userguide/cli-environment.html):

```
 AWS_ACCESS_KEY_ID=AKID1234567890
 AWS_SECRET_ACCESS_KEY=MY-SECRET-KEY
```

## Configure Grafiti

Grafiti takes a config file which configures it's basic function.

```toml
resourceTypes = ["AWS::EC2::Instance"]
endHour = 0
startHour = -8
endTimeStamp = "2017-06-14T01:01:01Z"
startTimeStamp = "2017-06-13T01:01:01Z"
region = "us-east-1"
maxNumRequestRetries = 11
includeEvent = false
tagPatterns = [
  "{CreatedBy: .userIdentity.arn}"
]
filterPatterns = [
  ".TaggingMetadata.ResourceType == \"AWS::EC2::Instance\""
]
```

 * `resourceTypes` - Specifies a list of resource types to query for. These can be any values the CloudTrail [API](http://docs.aws.amazon.com/awscloudtrail/latest/userguide/view-cloudtrail-events-supported-resource-types.html), or CloudTrail [log files](http://docs.aws.amazon.com/awscloudtrail/latest/userguide/cloudtrail-supported-services.html) if you're parsing files from a CloudTrail S3 bucket, accept.
 * `endHour`,`startHour` - Specifies the range of hours (beginning at `startHour`, ending at `endHour`) to query events from CloudTrail.
 * `endTimeStamp`,`startTimeStamp` - Specifies the range between two exact times (beginning at `startTimeStamp`, ending at `endTimeStamp`) to query events from CloudTrail. These fields take RFC-3339 (no milliseconds) format.
    * **Note**: Only one of `*Hour`, `*TimeStamp` pairs can be used. An error will be thrown if both are used.
 * `region` - The AWS region to query.
 * `maxNumRequestRetries` = The maximum number of retries the delete request retryer should attempt. Defaults to 8.
 * `includeEvent` - Setting `true` will include the raw CloudEvent in the tagging output (this is useful for finding attributes to filter on).
 * `tagPatterns` - should use `jq` syntax to generate `{tagKey: tagValue}` objects from output from `grafiti parse`. The results will be included in the `Tags` field of the tagging output.
 * `filterPatterns` - will filter output of `grafiti parse` based on `jq` syntax matches.

### Environment variables

In addition to, or in lieu of, a config file, grafiti can be configured with the following environment variables:

 * `AWS_REGION` corresponds to the `region` config file field.
 * `GRF_START_HOUR` corresponds to the `startHour` config file field.
 * `GRF_END_HOUR` corresponds to the `endHour` config file field.
 * `GRF_START_TIMESTAMP` corresponds to the `startTimeStamp` config file field.
 * `GRF_END_TIMESTAMP` corresponds to the `endTimeStamp` config file field.
 * `GRF_INCLUDE_EVENT` corresponds to the `includeEvent` config file field.

If one of the above variables is set, its' data will be used as a default value. That means that the only circumstance in which its' value will be used is when no corresponding config file field is set, and, by extension, no config file is provided. Setting these variables allows you to avoid using a config file.

### A note on AWS resource deletion order
AWS resources have (potentially many) dependencies that must be explicitly detached/removed/deleted before deleting a top-level resource (ex. a VPC). Therefore a deletion order must be enforced. This order is universal for all AWS resources and is not use-case-specific, because deletion actions will only run if a resource with a specific tag, or one of it's dependencies, is detected.

#### Order
1. S3 Bucket
    1. S3 Object
2. Route53 HostedZone
    1. Route53 RecordSet
3. EC2 RouteTableAssociation
6. EC2 Instance
7. AutoScaling Group
8. AutoScaling LaunchConfiguration
9. ElasticLoadBalancer
10. EC2 NAT Gateway
11. ElasticIPAssociation
12. ElasticIP (Allocation)
13. IAM InstanceProfile
    1. IAM Role Association
14. IAM Role
15. IAM User
16. EC2 InternetGateway
    1. EC2 InternetGatewayAttachment
17. EC2 NetworkInterface
18. EC2 NetworkACL
    1. EC2 NetworkACL Entry
19. EC2 VPN Connection
    1. EC2 VPN Connection Route
20. EC2 CustomerGateway
21. EBS Volume
22. EC2 Subnet
23. EC2 RouteTable
    1. EC2 RouteTable Route
24. EC2 SecurityGroup
    1. EC2 SecurityGroup Ingress Rule
    2. EC2 SecurityGroup Egress Rule
25. EC2 VPN Gateway
    1. EC2 VPN Gateway Attachment
26. EC2 VPC
    1. EC2 VPC CIDRBlock

# Examples

## Parsing

Grafiti is designed to take advantage of existing tools like `jq`.

```sh
$ cat config.toml
```
```toml
resourceTypes = ["AWS::EC2::Instance"]
endHour = 0
startHour = -8
region = "us-east-1"
maxNumRequestRetries = 11
includeEvent = false
tagPatterns = [
  "{CreatedBy: .userIdentity.arn}",
]
```
```sh
$ grafiti -c config.toml parse
```
```json
{
  "TaggingMetadata": {
    "ResourceName": "i-05a1ecfab5f74ffac",
    "ResourceType": "AWS::EC2::Instance",
    "ResourceARN": "arn:aws:ec2:us-east-1:0123456789101:instance/i-05a1ecfab5f74ffac",
    "CreatorARN": "arn:aws:iam::0123456789101:user/user123",
    "CreatorName": "user123",
  },
  "Tags": {
  	"CreatedBy": "arn:aws:iam::0123456789101:user/user123"
  }
}
{
  "TaggingMetadata": {
    "ResourceName": "i-081db11d0978f67a2",
    "ResourceType": "AWS::EC2::Instance",
    "ResourceARN": "arn:aws:ec2:us-east-1:0123456789101:instance/i-081db11d0978f67a2",
    "CreatorARN": "arn:aws:iam::0123456789101:user/user123",
    "CreatorName": "user123",
  },
  "Tags": {
	"CreatedBy": "arn:aws:iam::0123456789101:user/user123"
  }
}
```

Grafiti output is designed to be filtered/parsed (filters and tag generators can be embedded in config.toml as well)
```sh
# Print all usernames
grafiti -c config.toml parse | jq 'TaggingMetadata.CreatorName' | sort | uniq

# Filter events by username
grafiti -c config.toml parse | jq 'if .Event.Username != "root" then . else empty end'

# Print only EC2 Instance events
grafiti -c config.toml parse | jq 'if .TaggingMetadata.ResourceType == "AWS::EC2::Instance" then . else empty end'

# These require `includeEvent=true` to be set to ouput the raw event data
# Print "CloudEvent" data (lower level event info, needs to be unescaped) (for linux change -E to -R)
grafiti -c config.toml parse | jq '.Event.CloudTrailEvent' | sed -E 's/\\(.)/\1/g' | sed -e 's/^"//' -e 's/"$//' | jq .

# Parse ARNs from "CloudEvent" data (for linux change -E to -R)
grafiti  -c config.toml parse | jq '.Event.CloudTrailEvent' | sed -E 's/\\(.)/\1/g' | sed -e 's/^"//' -e 's/"$//' | jq '.userIdentity.arn'
```

## Filtering

`grafiti` can filter resources parsed from CloudTrail events by their tag using the `filter` sub-command. Specifically, the `--ignore-file`/`-i` option takes a file with TagFilters corresponding to resource tags; resources with these tags will be removed from`grafiti parse` output, and remaining resources can be filtered into `grafiti tag`.

Filter input file (see `example-tags-input.json`) has the form:

```json
{
	"TagFilters": [
		{
			"Key": "TagKey",
			"Values": ["TagValue"]
		}
	]
}
```

**Note**: Both tag keys and values are _case sensitive_.

TagFilters have the same form as the AWS Resource Group Tagging API `TagFilter`. See the [docs for details](http://docs.aws.amazon.com/resourcegroupstagging/latest/APIReference/API_TagFilter.html). Of note is tag value chaining with AND and OR e.g. `"Values": ["Yes", "OR", "Maybe"]`

Filtering can be done like so:
```bash
grafiti -c config.toml parse | grafiti -c config.toml filter -i example-tags-input.json | grafiti -c tag
```

## Tagging

Tagging input takes the form:

```json
{
  "TaggingMetadata": {
    "ResourceName": "string",
    "ResourceType": "string",
    "ResourceARN": "arn:aws:namespace:region:account-id:resource-info",
    "CreatorARN": "arn:aws:iam::account-id:user/user-name",
    "CreatorName": "string",
  },
  "Tags": {
	"TagName": "TagValue"
  }
}
```
This will apply the tags to the referenced resource.

Once resources have been tagged, you can view them in your AWS Console by resource type, region, and tag.
1. In the main AWS console, open the `Resource Groups` dropdown in the top navigation bar.
2. Select `Tag Editor`.
3. Once navigated to the editor, search for your tags by region, resource type(s), and tag key(s) and value(s).

## Parsing + Tagging

### Tag EC2 instances with CreatedBy, CreatedAt, and TaggedAt
This is a full example of parsing events and generating tags from them.

config.toml
```toml
resourceTypes = ["AWS::EC2::Instance"]
endHour = 0
startHour = -8
region = "us-east-1"
maxNumRequestRetries = 11
includeEvent = false
tagPatterns = [
  "{CreatedBy: .userIdentity.arn}",
  "{CreatedAt: .eventTime}",
  "{TaggedAt: now|todate}",
]
filterPatterns = [
  ".TaggingMetadata.ResourceType == \"AWS::EC2::Instance\"",
]

```

Run:
```sh
grafiti parse -c config.toml | grafiti tag -c config.toml
```

The above command will tag all matching resources with tags that look like:

```json
{
  "CreatedBy": "arn:aws:iam::0123456789101:root",
  "CreatedAt": "20170512T01:02:03Z",
  "TaggedAt": "2017-04-28"
}
```


## Deleting

Grafiti supports deleting resources based on matching Tags.

Delete input file (see `example-tags-input.json`) has the form:

```json
{
	"TagFilters": [
		{
			"Key": "TagKey",
			"Values": ["TagValue"]
		}
	]
}
```

**Note**: Both tag keys and values are _case sensitive_.

As with `grafiti filter`'s input tag file, TagFilters have the same form as AWS TagFilter JSON. See the [docs for details](http://docs.aws.amazon.com/resourcegroupstagging/latest/APIReference/API_TagFilter.html). Of note is tag value chaining with AND and OR e.g. `"Values": ["Yes", "OR", "Maybe"]`

From stdin:
```sh
echo "{\"TagFilters\":[{\"Key\": \"DeleteMe\", \"Values\": [\"Yes\"]}]}" | grafiti -c config.toml delete
```
From file:
```sh
grafiti -c config.toml delete -f example-delete-tags.json
```
### Error handling

Additionally it is **highly recommended** that the `--ignore-errors` flag is used when deleting resources. Many top-level resources have dependencies that, if not first deleted, will cause API errors that interrupt deletion loops. `--ignore-errors` instead handles errors gracefully by continuing deletion loops and printing error messages to stdout in JSON format:

```json
{
  "error": "error message text"
}
```

### Deleting dependencies

By default, `grafiti delete` will not trace relationships between resources and add them to the deletion graph. Passing the `--all-deps` flag will trace these relationships and add all found dependencies to the deletion graph.

For example, if a tagged VPC has a user-created (non-default) subnet that is not tagged, running `grafiti delete` will not delete the subnet, and in all likelihood will not delete the VPC due to dependency issues imposed by AWS.

### Deleted resources report

The `--report` flag will enable `grafiti delete` to aggregate all failed resource deletions and pretty-print them after a run. Log records of failed deletions will be saved as JSON objects in a log file in your current directory. Logging functionality uses the [logrus](https://github.com/sirupsen/logrus) package, which allows you to both create and parse log entries. However, because grafiti log entries are verbose, the logrus log parser might not function as expected. We recommend using `jq` to parse log data.

# Grafiti

Grafiti is a tool for parsing, tagging, and deleting AWS resources.

Using a CloudTrail trail, resource CRUD events can be `grafiti parse`'d for resource identifying
information. This parsed data can be fed into `grafiti tag` and tagged using the AWS resource group tagging
API. Tagged resources are retrieved using the same API during `grafiti delete`, and deleted
using resource type-specific service API's. Each sub-command can be used in a sequential pipe,
or individually.

## Motivating Example

We listen to CloudTrail events, and tag created resources with a default expiration of 2 weeks and the ARN of the creating user.

Every day, we can query the resource tagging API for resources that will expire in one week, and the owners can be notified via email/Slack.

Every day, we also query for resources that have expired, and delete them.

# Installation
Ensure you have [Golang](https://golang.org/dl/) 1.7+ and `jq` (see below) installed and your `GOPATH` set correctly.

Retrieve and install grafiti (the binary will be in `$GOPATH/bin`):
```
go get github.com/coreos/grafiti/cmd/grafiti
```

## `jq` installation
`jq` is a CLI JSON parsing tool that `grafiti` uses internally to evaluate config file expressions, and must
be installed before running `grafiti`. You can find download instructions [here](https://stedolan.github.io/jq/download/).

## Grafiti commands

* `grafiti parse` - Parses CloudTrail data and outputs useful information (to be consumed by `grafiti tag`)
* `grafiti tag` - Tags resources in AWS based on tagging rules defined in your `config.toml` file
* `grafiti delete` - Deletes resources in AWS based on tags


```
Usage:
  grafiti [flags]
  grafiti [command]

Available Commands:
  delete      Delete resources in AWS
  help        Help about any command
  parse       parse and output resources by reading CloudTrail logs
  tag         tag resources in AWS

Flags:
  -c, --config string   Config file (default is $HOME/.grafiti.toml)
      --debug           Enable debug logging.
      --dry-run         Output changes to stdout instead of AWS.
  -e, --ignore-errors   Continue processing even when there are API errors.
```

# Usage

## Configure aws credentials

  In order to use grafiti, you will need to configure your machine to talk to aws with a `~/.aws/credentials` file.

```
 [default]
 aws_access_key_id = AKID1234567890
 aws_secret_access_key = MY-SECRET-KEY
```

 Alternatively, you can set the following environment variables:

```
 AWS_ACCESS_KEY_ID=AKID1234567890
 AWS_SECRET_ACCESS_KEY=MY-SECRET-KEY
```

## Configure Grafiti

Grafiti takes a config file which configures it's basic function.

```toml
[grafiti]
resourceTypes = ["AWS::EC2::Instance"]
hours = -8
region = "us-east-1"
includeEvent = false
tagPatterns = [
  "{CreatedBy: .userIdentity.arn}",
]
filterPatterns = [
  ".TaggingMetadata.ResourceType == \"AWS::EC2::Instance\"",
]
```

 * `resourceTypes` - Specifies a list of resources to query for. These can be any values the CloudTrail API accepts.
 * `hours` - Specifies how far back to query CloudTrail, in hours.
 * `region` - The region to query
 * `includeEvent` - Setting `true` will include the raw CloudEvent in the tagging output (this is useful for finding
attributes to filter on).
 * `tagPatterns` - should use `jq` syntax to generate `{tagKey: tagValue}` objects from output from `grafiti parse`. The
 results will be included in the `Tags` field of the tagging output.
 * `filterPatterns` - will filter output of `grafiti parse` based on `jq` syntax matches.


### A note on AWS resource deletion order
AWS resources have (potentially many) dependencies that must be explicitly detached/removed/deleted before
deleting a top-level resource (ex. a VPC). Therefore a deletion order must be enforced. This order is universal
for all AWS resources and is not use-case-specific, because deletion actions will only run if a resource with a
specific tag, or one of it's dependencies, is detected.

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
19. EC2 CustomerGateway
20. EBS Volume
21. EC2 Subnet
22. EC2 RouteTable
    1. EC2 RouteTable Route
23. EC2 SecurityGroup
    1. EC2 SecurityGroup Ingress Rule
    2. EC2 SecurityGroup Egress Rule
24. EC2 VPC
    1. EC2 VPC CIDRBlock

# Examples

## Parsing

Grafiti is designed to take advantage of existing tools like `jq`.

```sh
$ cat config.toml
```
```toml
[grafiti]
resourceTypes = ["AWS::EC2::Instance"]
hours = -8
region = "us-east-1"
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
[grafiti]
resourceTypes = ["AWS::EC2::Instance"]
hours = -8
region = "us-east-1"
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

Delete input file (see `example-delete-tags.json`) has the form:

```json
{
	"TagFilters": [
		{
			"Key": "DeleteMe",
			"Values": ["Yes"]
		}
	]
}
```
TagFilters have the same form as the AWS TagFilter resources, see the
[docs for details](http://docs.aws.amazon.com/resourcegroupstagging/latest/APIReference/API_TagFilter.html).
Of note is that values can be chained with AND and OR e.g. `"Values": ["Yes", "OR", "Maybe"]`

From stdin:
```sh
echo "{\"TagFilters\":[{\"Key\": \"DeleteMe\", \"Values\": [\"Yes\"]}]}" | grafiti -c config.toml delete
```
From file:
```sh
grafiti -c config.toml delete -f example-delete-tags.json
```
### Error handling

Additionally it is **highly recommended** that the `--ignore-errors` flag is used when deleting resources. Many top-level resources have
dependencies that, if not first deleted, will cause API errors that interrupt deletion loops. `--ignore-errors` instead handles
errors gracefully by continuing deletion loops and printing error messages to stdout in JSON format:

```json
{
  "error": "error message text"
}
```

### Deleting dependencies

By default, `grafiti delete` will not trace relationships between resources and add them to the deletion graph.
Passing the `--all-deps` flag will trace these relationships and add all found dependencies to the deletion graph.

For example, if a tagged VPC has a user-created (non-default) subnet that is not tagged, running `grafiti delete`
will not delete the subnet, and in all likelihood will not delete the VPC due to dependency issues imposed by AWS.

# Grafiti

Grafiti can parse CloudTrail event data and tag resources based on it.

## Motivating Example

We listen to CloudTrail events, and tag created resources with a default expiration of 2 weeks and the ARN of the creating user.

Every day, we can query the resource tagging API for resources that will expire in one week, and the owners can be notified via email/slack.

Every day, we also query for resources that have expired, and delete them.

### Tag EC2 instances with CreatedBy, TaggedAt, and ExpiresAt
This is a full example of parsing events and generating tags from them.

config.toml
```toml
[grafiti]
resourceType = "AWS::EC2::Instance"
hours = -8
az = "us-east-1"
includeEvent = false
tagPatterns = [
  "{CreatedBy: .userIdentity.arn}",
  "{TaggedAt: now|strftime(\"%Y-%m-%d\")}",
  "{ExpiresAt: (now+(60*60*24*14))|strftime(\"%Y-%m-%d\")}" # Expire in 2 weeks
]
filterPatterns = [
  ".TaggingMetadata.ResourceType == \"AWS::EC2::Instance\"",
]
```

Run:
```sh
grafiti parse -c config.toml | grafiti tag -c config.toml
```

This will tag all matching resources with tags that look like:

```json
{
  "CreatedBy": "arn:aws:iam::206170669542:root",
  "ExpiresAt": "2017-05-12",
  "TaggedAt": "2017-04-28"
}
```

# Usage

Make sure you have `jq` installed.

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

## Grafiti commands

* `grafiti parse` - Parses CloudTrail data and outputs useful information (to be consumed by `grafiti tag`)
* `grafiti tag` - Tags resources in AWS based on tagging rules
* `grafiti delete` - Deletes resources in AWS based on tag data


```
Usage:
  grafiti [flags]
  grafiti [command]

Available Commands:
  delete      delete resources in AWS
  help        Help about any command
  parse       parse and output resources by reading CloudTrail logs
  tag         tag resources in AWS

Flags:
  -c, --config string   config file (default is $HOME/.grafiti.toml)
  -d, --debug           enable debug logging
      --dry-run         output changes to stdout instead of AWS
  -e, --ignoreErrors    Continue processing even when there are API errors.
```

## Configure Grafiti

Grafiti takes a config file which configures it's basic function.

```toml
[grafiti]
resourceType = "AWS::EC2::Instance"
hours = -8
az = "us-east-1"
includeEvent = false
tagPatterns = [
  "{CreatedBy: .userIdentity.arn}",
]
filterPatterns = [
  ".TaggingMetadata.ResourceType == \"AWS::EC2::Instance\"",
]
```

 * `resourceType` - Specifies which type of resource to query for. This can be any value CloudTrail APIs accept.
 * `hours` - Specifies how far back to query CloudTrail, in hours.
 * `az` - The availability zone to query
 * `includeEvent` - Setting `true` will include the raw CloudEvent in the tagging output (this is useful for finding
attributes to filter on).
 * `tagPatterns` - should use jq syntax to generate `{tagKey: tagValue}` objects from output from `grafiti parse`. The
 results will be included in the `Tags` field of the tagging output.
 * `filterPatterns` - will filter output of `grafiti parse` based on jq syntax matches.

# Examples

## Parse CloudTrail data

Grafiti is designed to take advantage of existing tools like `jq`.

```sh
$ cat config.toml
```
```toml
[grafiti]
resourceType = "AWS::EC2::Instance"
hours = -8
az = "us-east-1"
includeEvent = false
tagPatterns = [
  "{CreatedBy: .userIdentity.arn}",
]
```
```sh
$ grafiti parse -c ./config.toml
```
```json
{
  "TaggingMetadata": {
    "ResourceName": "i-05a1ecfab5f74ffac",
    "ResourceType": "AWS::EC2::Instance",
    "ResourceARN": "arn:aws:ec2:us-east-1:206170669542:instance/i-05a1ecfab5f74ffac",
    "CreatorARN": "arn:aws:iam::206170669542:user/QuayEphemeralBuilder",
    "CreatorName": "QuayEphemeralBuilder"
  },
  "Tags": {
  	"CreatedBy": "arn:aws:iam::206170669542:user/QuayEphemeralBuilder"
  }
}
{
  "TaggingMetadata": {
    "ResourceName": "i-081db11d0978f67a2",
    "ResourceType": "AWS::EC2::Instance",
    "ResourceARN": "arn:aws:ec2:us-east-1:206170669542:instance/i-081db11d0978f67a2",
    "CreatorARN": "arn:aws:iam::206170669542:user/QuayEphemeralBuilder",
    "CreatorName": "QuayEphemeralBuilder"
  },
  "Tags": {
	"CreatedBy": "arn:aws:iam::206170669542:user/QuayEphemeralBuilder"
  }
}
```

Grafiti output is designed to be filtered/parsed (filters and tag generators can be embedded in config.toml as well)
```sh
# Print all usernames
grafiti parse | jq 'TaggingMetadata.CreatorName' | sort | uniq

# Filter events by username
grafiti parse -c ./config.toml | jq 'if .Event.Username != "root" then . else empty end'

# Print only EC2 Instance events
grafiti parse | jq 'if .TaggingMetadata.ResourceType == "AWS::EC2::Instance" then . else empty end'

# These require `includeEvent=true` to be set to ouput the raw event data
# Print "CloudEvent" data (lower level event info, needs to be unescaped) (for linux change -E to -R)
grafiti parse -c ./config.toml | jq '.Event.CloudTrailEvent' | sed -E 's/\\(.)/\1/g' | sed -e 's/^"//' -e 's/"$//' | jq .

# Parse ARNs from "CloudEvent" data (for linux change -E to -R)
grafiti parse -c ./config.toml | jq '.Event.CloudTrailEvent' | sed -E 's/\\(.)/\1/g' | sed -e 's/^"//' -e 's/"$//' | jq '.userIdentity.arn'
```

## Tag AWS Resources

Tagging input takes the form:

```json
{
  "TaggingMetadata": {
    "ResourceName": "data",
    "ResourceType": "data",
    "ResourceARN": "arn:aws:ec2:us-east-1:206170669542:instance/i-081db11d0978f67a2",
    "CreatorARN": "data",
    "CreatorName": "data"
  },
  "Tags": {
	"TagName": "TagValue"
  }
}
```

This will apply the tags to the referenced resource.


## Parsing + Tagging


### Tag EC2 instances with CreatedBy, CreatedAt, and TaggedAt
This is a full example of parsing events and generating tags from them.

config.toml
```toml
[grafiti]
resourceType = "AWS::EC2::Instance"
hours = -8
az = "us-east-1"
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

## Deleting Resources

Grafiti supports deleting resources based on matching Tags.

Delete input has the form:

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

```sh
echo "{\"TagFilters\":[{\"Key\": \"DeleteMe\", \"Values\": [\"Yes\"]}]}" | ./grafiti --dry-run -c config.toml delete
```
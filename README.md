# Grafiti/Predator

 Grafiti tags your AWS resources.

 Predator hunts your tagged images for sport.

## Configure aws credentials

  In order to user these tools, you will need to configure your machine to talk to aws with a `~/.aws/credentials` file.

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

## Usage

Parse and generate tags with `grafiti parse`. Use that data to tag resources in AWS with `grafiti tag`.

```
Usage:
  grafiti [flags]
  grafiti [command]

Available Commands:
  help        Help about any command
  parse       parse and output resources by reading CloudTrail logs
  tag         tag resources in AWS

Flags:
  -c, --config string   config file (default is $HOME/.grafiti.toml)
  -d, --debug           enable debug logging
      --dry-run         output changes to stdout instead of AWS
```

## Configure Grafiti

```
[grafiti]
resourceType = "AWS::EC2::Instance"
hours = -8
az = "us-east-1"
includeEvent = false
tagPatterns = [
  "{CreatedBy: .userIdentity.arn}",
#  "{CreatedAt: .eventTime}",
#  "{TaggedAt: now|todate}",
]
filterPatterns = [
#  ".TaggingMetadata.ResourceType == \"AWS::EC2::Instance\"",
#  ".TaggingMetadata.ResourceType == \"AWS::ElasticLoadBalancing::LoadBalancer\"",
]
```

Setting `includeEvent = true` will include the raw CloudEvent in the tagging output (this is useful for finding
attributes to filter on).

`tagPatterns` should use jq syntax to generate `{tagKey: tagValue}` objects from output from `grafiti parse`. The
 results will be included in the `Tags` field of the tagging output.

`filterPatterns` will filter output of `grafiti parse` based on jq syntax matches.


## Parse CloudTrail data

Grafiti is designed to take advantage of existing tools like `jq`.

```
$ cat config.toml
[grafiti]
resourceType = "AWS::EC2::Instance"
hours = -8
az = "us-east-1"
includeEvent = false
tagPatterns = [
  "{CreatedBy: .userIdentity.arn}",
]

$ grafiti parse -c ./config.toml

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
...
```

Grafiti output is designed to be filtered/parsed (filters and tag generators can be embedded in config.toml as well)
```
# Print all usernames
grafiti parse | jq '.CreatorName' | sort | uniq

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
    "ResourceType": "data,
    "ResourceARN": "arn:aws:ec2:us-east-1:206170669542:instance/i-081db11d0978f67a2",
    "CreatorARN": "data,
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
```
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
```
grafiti parse -c config.toml | grafiti tag -c config.toml
```
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
```

## Parse CloudTrail data

Grafiti is designed to take advantage of existing tools like `jq`.

```
# Print default tagging data
grafiti parse -c ./config.toml

# Output
{
  "TaggingMetadata": {
    "ResourceName": "i-05a1ecfab5f74ffac",
    "ResourceType": "AWS::EC2::Instance",
    "ResourceARN": "arn:aws:ec2:us-east-1:206170669542:instance/i-05a1ecfab5f74ffac",
    "CreatorARN": "arn:aws:iam::206170669542:user/QuayEphemeralBuilder",
    "CreatorName": "QuayEphemeralBuilder"
  }
}
{
  "TaggingMetadata": {
    "ResourceName": "i-081db11d0978f67a2",
    "ResourceType": "AWS::EC2::Instance",
    "ResourceARN": "arn:aws:ec2:us-east-1:206170669542:instance/i-081db11d0978f67a2",
    "CreatorARN": "arn:aws:iam::206170669542:user/QuayEphemeralBuilder",
    "CreatorName": "QuayEphemeralBuilder"
  }
}
...
```

Grafiti output is designed to be filtered/parsed
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
	"Resource": {
		"ResourceType": "AWS:EC2:Instance",
		"ResourceName": "i-123456"
	},
	"Tags": {
		"TagName": "ValueName",
		"TagName2": "ValueName2"
	}
}
```

This will apply the tags to the referenced resource.


## Parsing + Tagging


### Tag with Username
This is a full example of parsing events and generating tags from them. This matches all EC2::Instance events created by
non-root users and adds a "CreatedBy: <username>" tag to them.

```
grafiti parse -c config.toml | jq 'if .Event.Username != "root" then . else empty end' | jq '{Tags: {CreatedBy: .Event.Username}, Resource: .Event.Resources[]}' | jq 'if .Resource.ResourceType == "AWS::EC2::Instance" then . else empty end' | grafiti -c config.toml tag
```

Output:
```
Tagging EC2 Instance i-0b512b2b72f960a94
Tagging EC2 Instance i-037f3df2ad504f355
Tagging EC2 Instance i-0a383997ce0d97773
Tagging EC2 Instance i-0eb72af0dc46eab32
Tagging EC2 Instance i-0b6f8a5db42547739
Tagging EC2 Instance i-032f2ab01e3a952dc
```
### Tag with Expire time

```
# Time until expiry in seconds
grafiti parse -c config.toml | jq 'if .Event.Username != "root" then . else empty end' | jq '{Tags: {CreatedBy: .Event.Username, Expires: (now+60*60*24)|todate}, Resource: .Event.Resources[]}' | jq 'if .Resource.ResourceType == "AWS::EC2::Instance" then . else empty end'
```

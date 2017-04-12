# Grafiti/Predator

 Grafiti tags your AWS resources.

 Predator hunts your tagged images for sport.

## Configure aws credentials

  In order to user these tools, you will need to configure your machine to talk to aws with a `~/.aws/credentials` file.

```
 [default]
 aws_access_key_id = AKID1234567890
 aws_secret_access_key = MY-SECRET-KEY
 You can learn more about the credentials file from this blog post.
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
# Print usernames
grafiti parse | jq '.Event.Username' | sort | uniq

# Print Username: Resource Pairs
grafiti parse -c ./config.toml | jq '{user: .Event.Username, resource: .Event.Resources[]}'

# Filter events by username
grafiti parse -c ./config.toml | jq 'if .Event.Username != "root" then . else empty end'

# Filter events by username and print matches as Username: Resource pairs
grafiti parse -c ./config.toml | jq 'if .Event.Username != "root" then . else empty end' | jq '{user: .Event.Username, resource: .Event.Resources[]}''

# Get all EC2 instances created by a user and output tagging rules to be consumed by "grafiti tag"
grafiti parse -c ./config.toml | jq 'if .Event.Username == "ant31" then . else empty end' | jq '{Tags: {CreatedBy: .Event.Username}, Resource: .Event.Resources[]}' | jq 'if .Resource.ResourceType == "AWS::EC2::Instance" then . else empty end'

# Print "CloudEvent" data (lower level event info, needs to be unescaped) (for linux change -E to -R)
grafiti parse -c ./config.toml | jq '.Event.CloudTrailEvent' | sed -E 's/\\(.)/\1/g' | sed -e 's/^"//' -e 's/"$//' | jq .

# Parse ARNs from "CloudEvent" data (for linux change -E to -R)
grafiti parse -c ./config.toml | jq '.Event.CloudTrailEvent' | sed -E 's/\\(.)/\1/g' | sed -e 's/^"//' -e 's/"$//' | jq '.userIdentity.arn'
```

## Tag AWS Resources

Tagging input takes the form:


```json
{
	"Resource" {
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

This is a full example of parsing events and generating tags from them. This matches all EC2::Instance events created by
non-root users and adds a "CreatedBy: <username>" tag to them.

```
grafiti parse -c config.toml | jq 'if .Event.Username != "root" then . else empty end' | jq '{Tags: {CreatedBy: .Event.Username}, Resource: .Event.Resources[]}' | jq 'if .Resource.ResourceType == "AWS::EC2::Instance" then . else empty end' | grafiti -c config.toml tag
```

Output:
```

```
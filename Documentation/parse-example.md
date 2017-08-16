# Parsing

These examples demonstrate usage of the `parse` sub-command.

Grafiti is designed to take advantage of existing tools like `jq`.

```sh
$ cat config.toml
```
```toml
resourceTypes = ["AWS::EC2::Instance"]
endHour = 0
startHour = -8
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

Grafiti output is designed to be filtered/parsed. Filters and tag generators can be embedded in config.toml as well.

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

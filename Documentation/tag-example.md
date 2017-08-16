# Tagging

These examples demonstrate usage of the `tag` sub-command.

Tagging input has the form:

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

Parsing and tagging are often done together, as `tag` input naturally takes `parse` output.

### Tag EC2 instances with CreatedBy, CreatedAt, and TaggedAt

This is a full example of parsing events and generating tags from them.

config.toml
```toml
resourceTypes = ["AWS::EC2::Instance"]
endHour = 0
startHour = -8
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

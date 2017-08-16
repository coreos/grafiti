# Deleting

These examples demonstrate usage of the `delete` sub-command.

Grafiti supports deleting resources based on matching Tags. Delete input file (see [`example-tags-input.json`][file-example-tags-input]) has the form:

```json
{
	"TagFilters": [
		{
			"Key": "TagKey",
			"Values": ["TagValue1", "TagValue2"]
		}
	]
}
```

**Note**: Both tag keys and values are _case sensitive_.

As with `grafiti filter`'s input tag file, TagFilters have the same form as AWS TagFilter JSON. See the [AWS docs for details](aws-docs-rgta-tagfilter). Of note is tag value chaining with AND and OR e.g. `"Values": ["Yes", "OR", "Maybe"]`

From stdin:
```sh
echo "{\"TagFilters\":[{\"Key\": \"DeleteMe\", \"Values\": [\"Yes\"]}]}" | grafiti -c config.toml delete
```
From file:
```sh
grafiti -c config.toml delete -f example-delete-tags.json
```

[aws-docs-rgta-tagfilter]: http://docs.aws.amazon.com/resourcegroupstagging/latest/APIReference/API_TagFilter.html

[file-example-tags-input]: ../example-tags-input.json

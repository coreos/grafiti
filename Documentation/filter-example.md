# Filtering

These examples demonstrate usage of the `filter` sub-command.

Grafiti can filter resources parsed from CloudTrail events by their tag using the `filter` sub-command. Specifically, the `--ignore-file`/`-i` option takes a file with TagFilters corresponding to resource tags; resources with these tags will be removed from`grafiti parse` output, and remaining resources can be filtered into `grafiti tag`.

Filter input file (see [`example-tags-input.json`][file-example-tags-input]) has the form:

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

TagFilters have the same form as the AWS Resource Group Tagging API `TagFilter`. See the [AWS docs for details][aws-docs-rgta-tagfilter]. Of note is tag value chaining with AND and OR e.g. `"Values": ["Yes", "OR", "Maybe"]`

Filtering can be done like so:
```bash
grafiti -c config.toml parse | \
grafiti -c config.toml filter -i example-tags-input.json | \
grafiti -c config.toml tag
```

[aws-docs-rgta-tagfilter]: http://docs.aws.amazon.com/resourcegroupstagging/latest/APIReference/API_TagFilter.html

[file-example-tags-input]: ../example-tags-input.json

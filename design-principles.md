# Design Principles

There are exceptions to these principles, but these are general guidelines the project strives to adhere to.

## General

- grafiti should be a tool that can:
    - Parse AWS resource data from either the CloudTrail API or CloudTrail log files and evaluate tag key and value expressions from that data.
        - See the Parsing section for an explanation of key and value expressions.
    - Filter AWS resource data, in the format of parsed CloudTrail data, by tag keys and/or values.
    - Tag AWS resources that support tagging with user-defined tags.
        - See the [contribution](CONTRIBUTING.md) guide for information regarding the projects' policy on supported resource types.
    - Delete AWS resources tagged with user-defined tags.
        - See the [contribution](CONTRIBUTING.md) guide for information regarding the projects' policy on supported resource types.
- Each of the 4 actions listed above map to 4 subcommands listed below.
    - Subcommands are meant to be chained together using \*nix `pipe` capabilities.
- Feature additions should be developed with consideration to the parse [-> filter] -> tag -> delete pipeline.
- Feature flags should be avoided to keep the grafiti learning curve as low as possible.
- Creating AWS resources is not within the scope of this project.

## Parsing

- `grafiti parse` should only accept either:
    - CloudTrail API response data
    - CloudTrail S3 bucket log file data
- Parsing should append tag key and value expressions, specified in a configuration file, post-evaluation to identifying resource information in JSON format, which will be used by:
    - `grafiti filter` to filter AWS resources by their tag keys and/or values
    - `grafiti tag` to tag AWS resources
- Tag key and value expressions will be evaluated using some JSON parser (see [jq](https://stedolan.github.io/jq/)), as values for these fields should be extractable from CloudTrail data if desired.
- Parsed data should be filterable using `resourceTypes` and `filterPatterns` fields defined in a configuration file. See the [README](README.md) file for field specifications.

## Filtering

- `grafiti filter` should only accept data in the format of `grafiti parse` output.
- Filtering should involve querying the AWS API to get resource tag information.
- Filter tags should be of the format described in the Resource Group Tagging API (RGTA) [TagFilter](http://docs.aws.amazon.com/resourcegroupstagging/latest/APIReference/API_TagFilter.html) docs, demonstrated in an example [tag input file](example-tags-input.json).

## Tagging

- `grafiti tag` should only accept data in the format of `grafiti parse`, and by proxy `grafiti filter`, output.
- Tags applied by `grafiti parse` using `tagPatterns` should be applied to AWS resources using the appropriate AWS API.
    - All resource types that are supported by the RGTA will be tagged using the RGTA.
    - Resource types not supported by the RGTA will be tagged using their respective AWS API's.

## Deleting

- `grafiti delete` should only accept a file containing tags of the format described in the Resource Group Tagging API (RGTA) [TagFilter](http://docs.aws.amazon.com/resourcegroupstagging/latest/APIReference/API_TagFilter.html) docs, demonstrated in an example [tag input file](example-tags-input.json).
- Deleting should have the option to delete AWS resources in one of the following ways:
    - An AWS resource that is tagged
    - An AWS resource that is tagged and all dependent resources that must be explicitly deleted before the parent resource is deleted.
- Deleting should offer useful and informative reporting of what is being deleted, and what has failed to delete.

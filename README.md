# Grafiti

[![Build Status](jenkins-grafiti-icon)](jenkins-grafiti-build-job)

[jenkins-grafiti-build-job]: https://jenkins-tectonic-installer.prod.coreos.systems/job/grafiti
[jenkins-grafiti-icon]: https://jenkins-tectonic-installer.prod.coreos.systems/job/grafiti/badge/icon

Grafiti is a tool for parsing, tagging, and deleting AWS resources.

* Using a [CloudTrail][aws-docs-cloudtrail] trail, resource CRUD events can be parsed using `grafiti` for identifying resource information.
* Parsed data can optionally be fed through `grafiti filter --ignore-file <tag-file>`, which filters out all resources tagged with tags in `<tag-file>` from parsed data.
* Parsed data can be fed into `grafiti tag` and tagged using the AWS resource group tagging API.
* Tagged resources are retrieved using the same API during `grafiti delete`, and deleted using resource type-specific service API's.

Each sub-command can be used in a sequential pipe, or individually.

## Motivating Example

We listen to CloudTrail events, and tag created resources with a default expiration of 2 weeks and the ARN of the creating user.

Every day, we can query the resource tagging API for resources that will expire in one week, and the owners can be notified via email/Slack.

Every day, we also query for resources that have expired, and delete them.

# Installation
Ensure you have the following installed:

* [Golang][golang-website] 1.7+
* `jq` (see below)
* The [glide][glide-website] package manager

Retrieve and install grafiti (the binary will be in `$GOPATH/bin`):
```
go get -u github.com/coreos/grafiti/cmd/grafiti
```

If `$GOPATH/src/github.com/coreos/grafiti` is already present, simply install grafiti:
```
go install github.com/coreos/grafiti/cmd/grafiti
```
or use the Makefile (requires `make`):
```
make install
```

## `jq` installation
`jq` is a CLI JSON parsing tool that `grafiti` uses internally to evaluate config file expressions, and must be installed before running `grafiti`. This program is quite useful for parsing `grafiti` input/output as well. You can find download instructions on the [`jq` website][jq-website].

# Usage

## Grafiti commands

* `grafiti parse` - Parses CloudTrail data and outputs useful information (to be consumed by `grafiti tag` or `grafiti filter`)
* `grafiti filter` - Filters `grafiti parse` output by removing resources with defined tags (to be consumed by `grafiti tag`)
* `grafiti tag` - Tags resources in AWS based on tagging rules defined in your `config.toml` file
* `grafiti delete` - Deletes resources in AWS based on tags


```
Usage:
  grafiti [flags]
  grafiti [command]

Available Commands:
  delete      Delete resources in AWS by tag.
  filter      Filter AWS resources by tag.
  help        Help about any command
  parse       Parse resource data from CloudTrail logs.
  tag         Tag resources in AWS.

Flags:
  -c, --config string   Config file (default: $HOME/.grafiti.toml).
      --debug           Enable debug logging.
      --dry-run         Output changes to stdout instead of AWS.
  -h, --help            help for grafiti
  -e, --ignore-errors   Continue processing even when there are API errors.

Use "grafiti [command] --help" for more information about a command.
```

## Configure AWS

You will need to [configure your machine][aws-docs-configure] to talk to AWS prior to running grafiti; configuring both credentials and [AWS region][aws-docs-configure-region] is required.

### Credentials

There are several ways to configure your AWS credentials for the [Go SDK][aws-docs-configure-credentials]. Grafiti supports all methods because it uses the Go SDK and does not implement its own credential handling logic.

## Configure Grafiti

Grafiti takes a config file which configures it's basic function.

```toml
resourceTypes = ["AWS::EC2::Instance"]
endHour = 0
startHour = -8
endTimeStamp = "2017-06-14T01:01:01Z"
startTimeStamp = "2017-06-13T01:01:01Z"
maxNumRequestRetries = 11
includeEvent = false
tagPatterns = [
  "{CreatedBy: .userIdentity.arn}"
]
filterPatterns = [
  ".TaggingMetadata.ResourceType == \"AWS::EC2::Instance\""
]
logDir = "/var/log"
```

 * `resourceTypes` - Specifies a list of resource types to query for. These can be any values the CloudTrail [API][aws-docs-cloudtrail-supp-res-api], or CloudTrail [log files][aws-docs-cloudtrail-supp-res-log] if you're parsing files from a CloudTrail S3 bucket, accept.
 * `endHour`,`startHour` - Specifies the range of hours (beginning at `startHour`, ending at `endHour`) to query events from CloudTrail.
 * `endTimeStamp`,`startTimeStamp` - Specifies the range between two exact times (beginning at `startTimeStamp`, ending at `endTimeStamp`) to query events from CloudTrail. These fields take RFC-3339 (no milliseconds) format.
    * **Note**: Only one of `*Hour`, `*TimeStamp` pairs can be used. An error will be thrown if both are used.
 * `maxNumRequestRetries` = The maximum number of retries the delete request retryer should attempt. Defaults to 8.
 * `includeEvent` - Setting `true` will include the raw CloudEvent in the tagging output (this is useful for finding attributes to filter on).
 * `tagPatterns` - should use `jq` syntax to generate `{tagKey: tagValue}` objects from output from `grafiti parse`. The results will be included in the `Tags` field of the tagging output.
 * `filterPatterns` - will filter output of `grafiti parse` based on `jq` syntax matches.
 * `logDir` - By default, grafiti logs to stderr. If this field is present in your config, grafiti writes logs to a file in this directory. Log files have the format: 'grafiti-yyyymmdd_HHMMSS.log'.

### Environment variables

Grafiti can be configured with the following environment variables in addition to, or in lieu of, a config file:

 * `GRF_START_HOUR` corresponds to the `startHour` config file field.
 * `GRF_END_HOUR` corresponds to the `endHour` config file field.
 * `GRF_START_TIMESTAMP` corresponds to the `startTimeStamp` config file field.
 * `GRF_END_TIMESTAMP` corresponds to the `endTimeStamp` config file field.
 * `GRF_INCLUDE_EVENT` corresponds to the `includeEvent` config file field.
 * `GRF_MAX_NUM_RETRIES` corresponds to the `maxNumRequestRetries` config file field.

If one of the above variables is set, its' data will be used as the corresponding config value and override that config file field if set. Setting environment variables allows you to avoid using a config file in certain cases; some config file fields are complex, ex. `tagPatterns` and `filterPatterns`, and cannot be succinctly encoded by environment variables. See [this pull request][grafiti-pr-env-var] for the reasoning behind this hierarchy.

## Further documentation

A note on resource [deletion order][file-deletion-order].

Examples of grafiti in action:
  * [Parsing][file-parse-example] resource data.
  * [Filtering][file-filter-example] resource data between parse and tag stages.
  * [Tagging][file-tag-example] resources in AWS.
  * [Deleting][file-delete-example] resources in AWS.

Kubernetes:
  * How to run grafiti as a [Kubernetes CronJob][file-kube-cronjob].

Usage notes and tips:
  * [Error handling][file-usage-notes-error-handle] configuration.
  * Using the [`--all-deps` flag][file-usage-notes-all-deps] to delete child dependencies.
  * Generating a [report][file-usage-notes-report].
  * [Logging][file-usage-notes-logging] configuration.

[aws-docs-cloudtrail]: https://aws.amazon.com/cloudtrail/
[aws-docs-cloudtrail-supp-res-api]: http://docs.aws.amazon.com/awscloudtrail/latest/userguide/view-cloudtrail-events-supported-resource-types.html
[aws-docs-cloudtrail-supp-res-log]: http://docs.aws.amazon.com/awscloudtrail/latest/userguide/cloudtrail-supported-services.html
[aws-docs-configure]: http://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html
[aws-docs-configure-credentials]: http://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials
[aws-docs-configure-region]: http://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-the-region

[file-delete-example]: Documentation/delete-example.md
[file-deletion-order]: Documentation/deletion-order.md
[file-filter-example]: Documentation/filter-example.md
[file-kube-cronjob]: Documentation/kubernetes-cronjob.md
[file-parse-example]: Documentation/parse-example.md
[file-tag-example]: Documentation/tag-example.md
[file-usage-notes-all-deps]: Documentation/usage-notes-and-tips.md#deleting-dependencies
[file-usage-notes-error-handle]: Documentation/usage-notes-and-tips.md#error-handling
[file-usage-notes-logging]: Documentation/usage-notes-and-tips.md#logging
[file-usage-notes-report]: Documentation/usage-notes-and-tips.md#deleted-resources-report

[golang-website]: https://golang.org/dl/

[glide-website]: https://glide.sh/

[grafiti-pr-env-var]: https://github.com/coreos/grafiti/pull/110

[jq-website]: https://stedolan.github.io/jq/download/

# Usage notes and tips

Grafiti is highly configurable, and with complex configuration often comes confusion. This document discusses points that the authors and users of grafiti think enhance the user experience.

## Error handling

We **highly recommend** using the `--ignore-errors` flag when deleting resources. Many top-level resources have dependencies that, if not first deleted, will cause API errors that interrupt deletion loops. `--ignore-errors` instead handles errors gracefully by continuing deletion loops and printing error messages to stdout in JSON format:

```json
{
  "level": "error",
  "msg": "error message",
  "time": "2017-08-01T123:59:59-07:00"
}
```

## Deleting dependencies

By default, `grafiti delete` will not trace relationships between resources and add them to the deletion graph. Passing the `--all-deps` flag will trace these relationships and add all found dependencies to the deletion graph.

For example, if a tagged VPC has a user-created (non-default) subnet that is not tagged, running `grafiti delete` will not delete the subnet, and in all likelihood will not delete the VPC due to dependency issues imposed by AWS.

## Deleted resources report

The `--report` flag will enable `grafiti delete` to aggregate all failed resource deletions and pretty-print them after a run. Log records of failed deletions will be saved as JSON objects in a log file in your current directory. Logging functionality uses the [logrus][logrus-repo] package, which allows you to both create and parse log entries. However, because grafiti log entries are verbose, the logrus log parser might not function as expected. We recommend using `jq` to parse log data.

## Logging

Grafiti supports two forms of logging: to a file or stderr. Logs are sent to stderr by default, and to a log file if the `logDir` config field (`GRF_LOG_DIR` environment variable) is not empty. In the latter case, grafiti log files of the format `grafiti-yyyymmdd_HHMMSS.log` are created by each `grafiti` execution.

[logrus-repo]: https://github.com/sirupsen/logrus

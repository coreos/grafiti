# Grafiti Roadmap

This document tracks desired grafiti features and their timelines.

## Grafiti 0.2

The 0.2 grafiti release includes better error reporting and parallelism features.

Goals:

* Comprehensive error reporting and logging.
    * **Status:** WIP, PR: [#111](https://github.com/coreos/grafiti/pull/111)
    * **Issues:** [#107](https://github.com/coreos/grafiti/issues/107)
    * **Timeline:** 2017/8/18
* Rate limiting on all API calls in order to support parallelism.
    * **Status:** WIP
    * **Issues:** [#7](https://github.com/coreos/grafiti/issues/7)
    * **Timeline:** 2017/8/18

## Beyond 0.2

All features required for the 1.0 release are documented below. These features will be constantly updated as new user stories surface.

* Slack integration to notify users of dangling resources and deletions.
* Parallelism in requests.

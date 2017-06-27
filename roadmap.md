# Grafiti Roadmap

This document tracks desired `grafiti` features and their timelines.

## Grafiti 0.1

The 0.1 release signifies that a minimal but production-grade version of `grafiti` is ready.

Goals:

* Handle AWS rate limiting/failures with exponential backoff retryer.
    * **Status:** PR [#65](https://github.com/coreos/grafiti/pull/65)
    * **Issues:** [#5](https://github.com/coreos/grafiti/issues/5)
    * **Timeline:** 2017/6/30
* Should be runnable as a daemon or CronJob on a Kubernetes cluster.
    * **Status:** WIP
    * **Issues:** [#6](https://github.com/coreos/grafiti/issues/6), [#33](https://github.com/coreos/grafiti/issues/33)
    * **Timeline:** 2017/6/30
* Parse CloudTrail events from log files directly.
    * **Status:** PR [#71](https://github.com/coreos/grafiti/pull/65)
    * **Issues:** [#47](https://github.com/coreos/grafiti/issues/47)
    * **Timeline:** 2017/06/29

## Beyond 0.1

Everything leading up to a 1.0 release will be documented below.

* Slack integration to notify users of dangling resources and deletions.
* Parallelism in requests.

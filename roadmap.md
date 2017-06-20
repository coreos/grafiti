# Grafiti Roadmap

This document tracks desired `grafiti` features and their timelines.

## Grafiti 0.1

The 0.1 release signifies that a minimal but production-grade version of `grafiti` is ready.

Goals:

* Reliably delete all resources in a single run.
    * **Status:** occasional failures involving EC2 Internet Gateways, otherwise working as expected.
    * **Issues:** none
    * **Timeline:** 2017/6/16
* Logging support for all errors, special logging for deletion failures such that log file can be parsed and deletions attempted repeatedly.
    * **Status:** [PR open](https://github.com/coreos/grafiti/pull/38) for deletion failure logging.
    * **Issues:** [#11](https://github.com/coreos/grafiti/issues/11)
    * **Timeline:** 2017/6/21
* Handle AWS rate limiting/failures with exponential backoff retryer.
    * **Status:** WIP
    * **Issues:** [#5](https://github.com/coreos/grafiti/issues/5)
    * **Timeline:** 2017/6/23
* Should be runnable as a daemon or CronJob on a Kubernetes cluster.
    * **Status:** WIP
    * **Issues:** [#6](https://github.com/coreos/grafiti/issues/6), [#33](https://github.com/coreos/grafiti/issues/33)
    * **Timeline:** 2017/6/23

## Beyond 0.1

Everything leading up to a 1.0 release will be documented below.

* Slack integration to notify users of dangling resources and deletions.

<img src="./logo/landscaper.svg" width="221">


# Landscaper

[![CI Build status](https://concourse.ci.gardener.cloud/api/v1/teams/landscaper/pipelines/landscaper-master/jobs/master-head-update-job/badge)](https://concourse.ci.gardener.cloud/teams/landscaper/pipelines/landscaper-master/jobs/master-head-update-job)
[![Go Report Card](https://goreportcard.com/badge/github.com/landscaper/landscaper)](https://goreportcard.com/report/github.com/landscaper/landscaper)
[![reuse compliant](https://reuse.software/badge/reuse-compliant.svg)](https://reuse.software/)

<!-- Motivation -->
_Landscaper_ provides the means to describe, install and maintain cloud-native landscapes. It allows
you to express an order of building blocks, connect output with input data and ultimately, bring your landscape to live.

What does a 'landscape' consist of? In this context it refers not only to application bundles but also includes
infrastructure components in public, private and hybrid environments. 

While tools like Terraform, Helm or native Kubernetes resources work well in their specific problem space, it has been a
manual task to connect them so far. Landscaper solves this specific problem and offers a fully-automated installation
flow. To do so, it translates blueprints of components into actionable items and employs well-known tools like Helm or
Terraform to deploy them. In turn the produced output can be used as input and trigger for a subsequent step -
regardless of the tools used underneath. Since implemented as a set of Kubernetes operators, Landscaper uses the concept
of reconciliation to enforce a desired state, which also allows for updates to be rolled out smoothly.
<!-- end -->

> **_NOTE:_** **The Landscaper now also supports [OCM (Open Component Model)](https://ocm.software/) Component
> Descriptors [Version 3](https://ocm.software/docs/component-descriptors/version-3/), additionally to [Version
> 2](https://ocm.software/docs/component-descriptors/version-2/).**

**Work in progress... partial and unfinished documentation ahead!**

### Start Reading
- The documentation can be found [here](docs/README.md) or you jump directly to the [Guided Tour](docs/guided-tour).
- A list of available deployers is maintained [here](docs/deployer).
- A glossary can be found [here](docs/concepts/Glossary.md)
- Installation instructions can be found [here](docs/installation/install-landscaper-controller.md)

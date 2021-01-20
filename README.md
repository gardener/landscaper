<img src="./logo/landscaper.svg" width="221">


# Landscaper

[![CI Build status](https://concourse.ci.gardener.cloud/api/v1/teams/gardener/pipelines/gardener-master/jobs/master-head-update-job/badge)](https://concourse.ci.gardener.cloud/teams/gardener/pipelines/landscaper-master/jobs/master-head-update-job)
[![Go Report Card](https://goreportcard.com/badge/github.com/gardener/landscaper)](https://goreportcard.com/report/github.com/gardener/landscaper)
[![reuse compliant](https://reuse.software/badge/reuse-compliant.svg)](https://reuse.software/)

**:warning: Landscaper is currently in an alpha state, expect the api to change at any time.**

**Work in progress... partial and unfinished documentation ahead!**

The _Landscaper_ is an [operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) to install, update & manage a Gardener deployment.

The documentation can be found [here](docs/README.md).<br>
A list of available deployers is maintained [here](docs/deployer).

<!-- Motivation -->

what is it
what does it
why is it better


Landscaper is a set of Kubernetes operators that manage the lifecycle of cloud-native landscapes comprised of multiple software deployments and infrastructure setups.
Dedicated deployers for Helm, Terraform or other deployment methods take blueprints that describe the desired state of the landscape, ultimately bring it to life and keep it up-to-date.


Today's cloud-native landscapes consist of not only application bundles but also multiple infrastructure components in public, private and hybrid environments.
Most tooling focuses on specific aspects of these installations like deploying a helm chart, however, there is no uniform way to describe a landscape and act upon it.

In this talk we aim to introduce a blueprint description for different infrastructure and software components, and how they fit together. An operator takes these to create a well-defined installation order and connect their input and output data flows.
The result is a comprehensive description of a landscape by means of deployable items which are picked up by dedicated operators. These operators utilize helm, terraform or any other kind of deployment method to bring the desired landscape setup to life.
Using the pull-principle for our deployer operators, we can manage components across public, private and hybrid environment with a single Landscaper.

Gardener, the Kubernetes Botanist, consists of many different services, programs and infrastructure components.
As a result, we encountered the challenge of installing and keeping our landscapes up to date, in order to also give our community an easy way to get started.
Our components are deployed with different tools like Terraform, Helm or native Kubernetes resources, and many more.
All of these tools work well in their specific problem space but we struggled to connect them into a fully-automated installation flow.
This is when we started to work on a tool agnostic landscape description format which allowed us to operate versioned components.

Knowing that we are not the only ones facing these issues, we would like to share our current approach with the community.


<!-- end -->
### Quick Start

The Landscaper is a kubernetes controller that can be easily installed via helm using the helm chart in [charts/landscaper](charts/landscaper).

```
helm install -n ls-system landscaper ./charts/landscaper
```

We also build oci charts so the landscaper can also be installed with a specific version with:
```
export HELM_EXPERIMENTAL_OCI=1
export LS_VERSION="0.1.0"
helm chart pull eu.gcr.io/gardener-project/landscaper/charts/landscaper-controller:$LS_VERSION
helm chart export eu.gcr.io/gardener-project/landscaper/charts/landscaper-controller:$LS_VERSION ./charts
helm install ./charts
```

The chart can be configured via the values file:
```yaml
image:
  tag: image version # .e.g. 0.2.0; check the latest releases in the github releases

landscaper:
  registryConfig:
    allowPlainHttpRegistries: false
    secrets: # contains optional oci secrets
      default: {
        "auths": {
          "hostname": {
            "auth": "my auth"
          }
        }
      }

  # deploy with integrated deployers for quick start
  deployers:
  - container
  - helm
  - manifest
```

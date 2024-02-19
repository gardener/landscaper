---
title: Prerequisites
sidebar_position: 1
---

# Guided Tour

## Prerequisites and Basic Definitions

- For all examples, you need a [running Landscaper instance](../installation/install-landscaper-controller.md).

- A convenient tool we will often use in the following examples is the [Landscaper
  CLI](https://github.com/gardener/landscapercli). 

- During the following exercises, you might need to change files, provided with the examples. For this, you should
  simply clone [this repository](https://github.com/gardener/landscaper) and do the required changes on your local files. You could also fork the repo and work on your fork.

- In all examples, 3 Kubernetes clusters are involved:

  - the **Landscaper Host Cluster**, on which the Landscaper runs
  - the **target cluster**, on which the deployments will be done
  - the **Landscaper Resource Cluster**, on which the various custom resources are stored. These custom resources are
    watched by the Landscaper, and define which deployments should happen on which target cluster.

  It is possible that some or all of these clusters coincide, e.g. in the most simplistic approach, you have only one
  cluster. Such a "one-cluster-setup" is the easiest way to start working with the Landscaper.

> **_NOTE:_** The Landscaper now also supports [OCM (Open Component Model)](https://ocm.software/) Component
> Descriptors [Version 3](https://ocm.software/docs/component-descriptors/version-3/), additionally to [Version
> 2](https://ocm.software/docs/component-descriptors/version-2/).  
> Since we try our best to avoid disruptions, this functionality is currently behind a feature switch. For detailed 
> information on how to enable this for your own landscaper instance, set the corresponding flag in the configuration to
> `true`, as shown [here](https://github.com/gardener/landscaper/blob/master/docs/installation/install-landscaper-controller.md#configuration-through-valuesyaml).
> If you just want to follow along the tour, you should have the switch enabled!

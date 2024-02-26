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

## How to follow the Tour

The Guided Tour consists of several chapters, some of which build on each other. In order to get the most out of it, 
you should be following the Guided Tour in this sequence. Another important point is, that this documentation
is not optimal with respect to getting an overview about the used example resources in the Guided Tour. Therefore,
it might be better to read it in the [original github repository](https://github.com/gardener/landscaper/tree/master/docs/guided-tour).

### A Hello World Example

[1. Hello World Example](./hello-world)

### Basics

[2. Upgrading the Hello World Example](./basics/upgrade)

[3. Manifest Deployer Example](./basics/manifest-deployer)

[4. Multiple Deployments in One Installation](./basics/multiple-deployitems)

### Recovering from Errors

[5. Handling an Immediate Error](./error-handling/immediate-error)

[6. Handling a Timeout Error](./error-handling/timeout-error)

[7. Handling a Delete Error](./error-handling/delete-error)

You can find a list of error messages and corresponding solutions [here](./error-handling/problem_analysis.md).

### Blueprints and Components

[8. An Installation with an Externally Stored Blueprint](./blueprints/external-blueprint)

[9. Helm Chart Resources in the Component Descriptor](./blueprints/helm-chart-resource)

[10. Echo Server Example](./blueprints/echo-server)

### Imports and Exports

[11. Import Parameters](./import-export/import-parameters)

[12. Import Data Mappings](./import-export/import-data-mappings)

[13. Export Parameters](./import-export/export-parameters)

### Templating

[14. Templating: Accessing Component Descriptors ](./templating/components)

## Target Maps

[15. Target Maps: Multiple Deploy Items](./target-maps/01-multiple-deploy-items)

[16. Target Maps: Target Map References](./target-maps/02-targetmap-ref)

[17. Target Maps: Multiple Subinstallations](./target-maps/03-multiple-subinst)

[18. Target Maps: Target Map on Subinstallation Level](./target-maps/04-forward-map)

[19. Target Maps: Other Target Map Examples](./target-maps/05-other-examples)

## Optimization

[20. Optimization Hints ](../usage/Optimization.md)

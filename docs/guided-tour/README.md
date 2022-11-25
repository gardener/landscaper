# Guided Tour

This document contains a guided tour presenting the different Landscaper features by examples.

## Prerequisites and Basic Definitions

For all examples you need a [running Landscaper instance](../gettingstarted/install-landscaper-controller.md).

A convenient tool that we will often use in the examples is the 
[Landscaper CLI](https://github.com/gardener/landscapercli). 

In all examples there are 3 kubernetes clusters involved:

- the **Landscaper Host Cluster**, on which the Landscaper runs;
- the **target cluster**, on which the Helm chart shall be deployed.
- the **Landscaper Resource Cluster**, on which the custom resources are stored that are watched by the Landscaper.
  These custom resources define what should be deployed on which target cluster.

It is possible that some or all of these clusters coincide, e.g. in the most simplistic approach you have only one
cluster. This is the easiest setup when you start working with the Landscaper.

## A Hello World Example

[1. Hello World Example](./hello-world)

## Basics

[2. Upgrading the Hello World Example](./basics/upgrade)

[3. Manifest Deployer Example](./basics/manifest-deployer)

[4. Multiple Deployments in One Installation](./basics/multiple-deployitems)

## Recovering from Errors

[5. Handling an Immediate Error](./error-handling/immediate-error)

[6. Handling a Timeout Error](./error-handling/timeout-error)

[7. Handling a Delete Error](./error-handling/delete-error)

## Blueprints and Components

[8. An Installation with an Externally Stored Blueprint](./blueprints/external-blueprint)

[9. Helm Chart Resources in the Component Descriptor](./blueprints/helm-chart-resource)


<!--
Observed generation, jobID, jobIDFinished
Delete without uninstall
Deploying a blueprint to multiple targets/target list
Pull secrets for helm chart repo (with and without secret ref)
Pull secret in context to access a protected oci registry
Timeouts
Import, export
Subinstallations
deploy executions in files
images listed in a component descriptor
additional files in blueprint, e.g. for config data

Make use of temp files in the scripts that upload a component descriptor
-->

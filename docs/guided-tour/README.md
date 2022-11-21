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

[Hello World Example](./hello-world)

## Basics

[Upgrading the Hello World Example](./basics/upgrade)

[Manifest Deployer Example](./basics/manifest-deployer)

## Recovering from Errors

[Handling an Immediate Error](./error-handling/immediate-error)

[Handling a Timeout Error](./error-handling/timeout-error)

[Handling a Delete Error](./error-handling/delete-error)

## Blueprints and Components

[An Installation with an Externally Stored Blueprint](./blueprints/external-blueprint)


<!--
Delete without uninstall
Observed generation, jobID, jobIDFinished
Deploying a blueprint to multiple targets/target list
Pull secrets for helm chart repo (with and without secret ref)
context to access oci registry
timeouts
-->

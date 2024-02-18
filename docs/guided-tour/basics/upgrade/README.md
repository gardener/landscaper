---
title: Upgrading Hello World
sidebar_position: 1
---

# Upgrading the Hello World Example

In this example, we start by deploying the hello-world Helm chart in its original version `1.0.0`. 
Afterwards, we will upgrade the Installation so that it deploys the newer version `1.0.1` of the chart.

For prerequisites see [here](../../README.md).

You can find the new Helm chart [here](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/basics/upgrade/chart/hello-world). It replaces the ConfigMap of the original chart version by
a Secret. The new chart version can be found in our
[public registry](https://eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:1.0.1).


## Procedure

First, we deploy the original hello-world helm chart:

1. Add the kubeconfig of your target cluster to your [target.yaml](installation/target.yaml) at the specified location.

2. On the Landscaper resource cluster, create a namespace `example` and apply your [target.yaml](installation/target.yaml) and [installation.yaml](installation/installation.yaml):
   
   ```shell
   kubectl create ns example
   kubectl apply -f <path to target.yaml>
   kubectl apply -f <path to installation-1.0.0.yaml>
   ```

3. Wait until the Installation reaches phase `Succeeded` and check that the ConfigMap of the Helm chart is available in the target cluster.

4. Upgrade the Installation by applying [installation-1.0.1.yaml](installation/installation-1.0.1.yaml). This Installation references the newer version `1.0.1` of the hello-world Helm chart, which simply deploys a Secret instead of a ConfigMap.

   ```shell
   kubectl apply -f <path to installation-1.0.1.yaml>
   ```

Note that the upgraded Installation has the annotation `landscaper.gardener.cloud/operation: reconcile`. Without this annotation, Landscaper will not start processing the Installation.

5. Wait until the Installation is again in phase `Succeeded`. The ConfigMap that was deployed by the old chart version should no longer exist. Instead, there should be a Secret deployed by the new chart version:

   ```shell
   kubectl get secret -n example hello-world
   ```

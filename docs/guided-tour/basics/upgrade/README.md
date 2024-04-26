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

1. In the [settings](commands/settings) file, adjust the variables `RESOURCE_CLUSTER_KUBECONFIG_PATH` 
   and `TARGET_CLUSTER_KUBECONFIG_PATH`.

2. On the Landscaper resource cluster, create a namespace `cu-example`.

3. Run script [commands/deploy-k8s-resources.sh](commands/deploy-k8s-resources.sh).
   It templates a [target.yaml.tpl](installation/target.yaml.tpl) and an [installation.yaml.tpl](installation/installation.yaml.tpl)
   and applies both on the resource cluster.

4. Wait until the Installation reaches phase `Succeeded` and check that the ConfigMap of the Helm chart is available in the target cluster.

5. Run script [commands/upgrade-installation.sh](commands/upgrade-installation.sh).
   It applies [installation-upg.yaml.tpl](installation/installation-upg.yaml.tpl). This upgraded Installation 
   references the newer version `1.0.1` of the hello-world Helm chart, which simply deploys a Secret instead of a ConfigMap.

Note that the upgraded Installation has the annotation `landscaper.gardener.cloud/operation: reconcile`. 
Without this annotation, Landscaper will not start processing the Installation.

6. Wait until the Installation is again in phase `Succeeded`. The ConfigMap that was deployed by the old chart version should no longer exist. Instead, there should be a Secret deployed by the new chart version:

   ```shell
   kubectl get secret -n example hello-world
   ```

## Cleanup

You can remove the Installation with the
[delete-installation script](commands/delete-installation.sh).
When the Installation is gone, you can delete the Target with the
[delete-other-k8s-resources script](commands/delete-other-k8s-resources.sh).

---
title: Multi-Chart Installations
sidebar_position: 3
---

# Deploying Multiple Helm Charts with One Installation

It is possible to combine multiple deployments in one Installation. We will demonstrate this with the blueprint in [this Installation](./installation/installation.yaml.tpl). It contains two DeployItems. Each of them deploys the hello-world Helm chart, but to two different clusters. Therefore, the Installation 
imports two Target objects: [target-1.yaml](./installation/target-1.yaml) and 
[target-2.yaml](./installation/target-2.yaml).

For prerequisites, see [here](../../README.md).


## Dependencies Between DeployItems

Suppose the second deployment must not take place until the first one has been successfully completed.
You can achieve this by defining a dependency between the two DeployItems. In the current example, we have defined that the second DeployItem depends on the first, i.e. the DeployItem with name `deploy-item-2` contains a [`dependsOn` section](../../../usage/Blueprints.md#deployitems) as follows:

```yaml
dependsOn:
   - deploy-item-1
```

As a consequence, DeployItem `deploy-item-1` will be processed first, and DeployItem `deploy-item-2` afterward.

Each DeployItem can depend on several other DeployItems. 

> Note: Always make sure that the dependency graph does not contain cycles!

## Procedure

1. In the [settings](commands/settings) file, adjust the variables `RESOURCE_CLUSTER_KUBECONFIG_PATH`,
   `TARGET_CLUSTER_KUBECONFIG_PATH_1`, and `TARGET_CLUSTER_KUBECONFIG_PATH_2`.

2. On the Landscaper resource cluster, create a namespace `cu-example`.

3. Run script [commands/deploy-k8s-resources.sh](commands/deploy-k8s-resources.sh).
   It creates two targets based on the template [target.yaml.tpl](installation/target.yaml.tpl) and the [installation.yaml.tpl](installation/installation.yaml.tpl).

4. Wait until the Installation is in phase `Succeeded`. Check if both target clusters contain the hello-world Helm chart deployment. 
   If you are fast enough, you should be able to observe that this happens one after the other in the two target clusters.

## Cleanup

You can remove the Installation with the
[delete-installation script](commands/delete-installation.sh).
When the Installation is gone, you can delete the Targets with the
[delete-other-k8s-resources script](commands/delete-other-k8s-resources.sh).

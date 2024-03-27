---
title: Multi-Chart Installations
sidebar_position: 3
---

# Deploying Multiple Helm Charts with One Installation

It is possible to combine multiple deployments in one Installation. We will demonstrate this with the blueprint in [this Installation](./installation/installation.yaml). It contains two DeployItems. Each of them deploys the hello-world Helm chart, but to two different clusters. Therefore, the Installation 
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

1. Insert the kubeconfigs of your target clusters into [target-1.yaml](./installation/target-1.yaml) and [target-2.yaml](./installation/target-2.yaml).

1. On the Landscaper resource cluster, create namespace `example` and apply the two targets and the Installation [installation.yaml](./installation/installation.yaml):
   
   ```shell
   kubectl create ns example
   kubectl apply -f <path to target-1.yaml>
   kubectl apply -f <path to target-2.yaml>
   kubectl apply -f <path to installation.yaml>
   ```
2. Check if both target clusters contain the hello-world Helm chart deployment. If you are fast enough, you should be able to observe that this always happens first in the cluster specified in [target-1.yaml](./installation/target-1.yaml), and only after this installation has succeeded, deployment is started in the cluster specified in [target-2.yaml](./installation/target-2.yaml).

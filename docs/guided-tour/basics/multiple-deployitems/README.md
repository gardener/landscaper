# Deploying Multiple Helm Charts with One Installation

For prerequisites see [here](../../README.md#prerequisites-and-basic-definitions).

It is possible to combine multiple deployments in one Installation. 
We demonstrate this with the blueprint in this [Installation](./installation/installation.yaml). It contains two
DeployItems. Each of them deploys the hello-world Helm chart, but to different clusters. Therefore, the Installation 
imports two Target objects [target-1.yaml](./installation/target-1.yaml) and 
[target-2.yaml](./installation/target-2.yaml).


## Dependencies Between DeployItems

Suppose the second deployment must not take place until the first one has been successfully completed.
You can achieve this by defining a dependency between the DeployItems. In the current example we have defined that
the second DeployItem depends on the first, i.e. the DeployItem with name `deploy-item-2` contains a 
[`dependsOn` section](../../../usage/Blueprints.md#deployitems) as follows:

```yaml
dependsOn:
   - deploy-item-1
```

As a consequence, DeployItem `deploy-item-1` will be executed first, and DeployItem `deploy-item-2` afterwards.

Each DeployItem can depend on several others. But you have to take care that the dependency graph contains no cycles.

## Procedure

1. Insert the kubeconfigs of your target clusters in the files [target-1.yaml](./installation/target-1.yaml) 
   and [target-2.yaml](./installation/target-2.yaml).

2. On the Landscaper resource cluster, create namespace `example` and apply the two targets and the Installation 
   [installation.yaml](./installation/installation.yaml):
   
   ```shell
   kubectl create ns example
   kubectl apply -f <path to target-1.yaml>
   kubectl apply -f <path to target-2.yaml>
   kubectl apply -f <path to installation.yaml>
   ```

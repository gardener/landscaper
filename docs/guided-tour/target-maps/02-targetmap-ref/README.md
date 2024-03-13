---
title: Deploying to Multiple Clusters using one Subinstallation
sidebar_position: 2
---
# Deploying to Multiple Clusters using a Subinstallation

In this example, we show again how target maps can be used to deploy an artefact to a variable number of target clusters. 

For prerequisites, see [here](../../README.md#prerequisites-and-basic-definitions) and check out the [first target map example](../01-multiple-deploy-items/README.md) 

This example is a modification of the first target map example. It achieves the same result, i.e. deploying 
config maps with specific data to a variable number of target clusters, but it uses a Subinstallation for that.
This prevents the problem, that when removing a target from the import target map, the target itself is only allowed
to be removed, when it is not required anymore for e.g. removing k8s resources from the target cluster.

## Example description

This example presents again a component version, which gets as input a map of targets and a data object containing
configuration data for every input target. The import data are forwarded to a Subinstallation.  
For every input target, the Subinstallation creates a DeployItem which deploys a config map with the right data.

The example component version is stored 
[here](https://eu.gcr.io/gardener-project/landscaper/examples/component-descriptors/github.com/gardener/guided-tour/targetmaps/guided-tour-targetmap-ref). 
If you want to upload the component version to another registry, you can just adapt the [settings](https://github.com/gardener/landscaper/blob/master/docs/guided-tour/target-maps/02-targetmap-ref/component/commands/settings) 
file and execute the component version build and upload script [here](https://github.com/gardener/landscaper/blob/master/docs/guided-tour/target-maps/02-targetmap-ref/component/commands/component.sh).

The component version itself is specified here:
  - [component configuration](component/components.yaml)
  - [blueprints](https://github.com/gardener/landscaper/blob/master/docs/guided-tour/target-maps/02-targetmap-ref/component/blueprint) 

## Installing the example

The procedure to install example is as follows:

1. On the Landscaper resource cluster create 
  - namespace `cu-example`

2. On the Landscaper resource cluster, in namespace `cu-example` create
  - different targets `cluster-blue`, `cluster-green`, `cluster-red`, `cluster-yellow`, `cluster-orange`. 
    Usually these targets contain access data to different target clusters but for simplicity we use the same target 
    cluster for all of them.
  - a Context `landscaper-examples` and a root Installation `targetmap-ref`
  - a DataObject `config` containing the data for the different config maps which will be deployed on the different
    target clusters.

The templates for these resources can be found [here](https://github.com/gardener/landscaper/blob/master/docs/guided-tour/target-maps/02-targetmap-ref/component/installation) and can be deployed with 
this [script](https://github.com/gardener/landscaper/blob/master/docs/guided-tour/target-maps/02-targetmap-ref/component/commands/deploy-k8s-resources.sh). Adapt the [settings](https://github.com/gardener/landscaper/blob/master/docs/guided-tour/target-maps/02-targetmap-ref/component/commands/settings) file
such that the entry `TARGET_CLUSTER_KUBECONFIG_PATH` points to the kubeconfig of the target cluster to which the
config maps should be deployed.

Now you can see successful installations on your Landscaper resource cluster:

```bash
kubectl get inst -n cu-example 
NAME                     PHASE       EXECUTION                AGE
targetmap-ref            Succeeded                            20m
targetmapref-sub-pcz2g   Succeeded   targetmapref-sub-pcz2g   20m
```

Let's have a deeper look into the resources of the example. The root Installation `targetmap-ref` contains as import 
a target map importing three of the five deployed targets.  

```yaml
  imports:
  
    targets:
    - name: clusters
      targetMap:
        blue: cluster-blue
        green: cluster-green
        red: cluster-red
        
    data:
    - dataRef: config
      name: config
      
    ...
```

Furthermore, the root Installation imports the DataObject `config` ([see](component/installation/dataobject.yaml.tpl)) 
which contains the configuration for the different config maps. 

The root installation references this [blueprint](component/blueprint/root/blueprint.yaml) which imports the 
`config` as well as the target map. This is all similar to the first target map example. The difference is
that the blueprint creates a [Subinstallation](component/blueprint/root/subinst.yaml) instead of DeployItems. 
This Subinstallation just forwards the import data of the root Installation. For the imported target map it uses a 
so-called target map reference:

```yaml
imports:
  targets:
    - name: clusters                     # name of the target map reference
      targetMapRef: rootclusters         # target map name in the blueprint
```

The Subinstallation uses another [blueprint](component/blueprint/sub/blueprint.yaml) which creates a DeployItem for 
every target provided by the target map in its [deployExecution](component/blueprint/sub/deploy-execution.yaml). 
Therefore, it iterates again over the target map with the following expression:

```yaml
{{ range $key, $target := .imports.clusters }}
```

The rest is quite similar as for the [first target map example](../01-multiple-deploy-items/README.md).

The DeployItems on the Landscaper resource cluster looks as follows:

```bash
kubectl get di -n cu-example                                                  
NAME                                    TYPE                                            PHASE       EXPORTREF   AGE
targetmapref-sub-pcz2g-di-blue-7jw4t    landscaper.gardener.cloud/kubernetes-manifest   Succeeded               59m
targetmapref-sub-pcz2g-di-green-tttpx   landscaper.gardener.cloud/kubernetes-manifest   Succeeded               59m
targetmapref-sub-pcz2g-di-red-ck4w5     landscaper.gardener.cloud/kubernetes-manifest   Succeeded               55m
```

## Note about Subinstallation and DeployItem names

It is important to note that the names of Subinstallations and DeployItems are created using the keys of the targets
in the target maps. This is important because it gives these objects a name which connects them to a particular target. 
If you remove one target of the input data of the root Installation, the right Subinstallation/DeployItem can be deleted
using the right target. Iterating over the target map and creating these names using some loop counter instead, might 
change their name if one of the import targets is removed and the counter of all succeeding targets is reduced by one.

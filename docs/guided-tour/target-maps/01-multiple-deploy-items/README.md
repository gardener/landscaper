---
title: Deploying to Multiple Clusters
sidebar_position: 1
---

# Deploying to Multiple Clusters

In this example, we show how target maps can be used to deploy an artefact to a variable number of target clusters. 

For prerequisites, see [here](../../README.md#prerequisites-and-basic-definitions).

## Description

This example presents a component, which gets as input a map of targets and a data object containing
configuration data for every input target. For every input target a DeployItem is created which deploys
a config map on the target cluster specified by one of the input targets. The data of the config map contains the
corresponding data provided as another import.

The example component is stored 
[here](https://eu.gcr.io/gardener-project/landscaper/examples/component-descriptors/github.com/gardener/guided-tour/targetmaps/guided-tour-multiple-deploy-items). 
If you want to upload the component to another registry, you can just adapt the [settings](https://github.com/gardener/landscaper/blob/master/docs/guided-tour/target-maps/01-multiple-deploy-items/component/commands/settings) 
file and execute the component build and upload script [here](https://github.com/gardener/landscaper/blob/master/docs/guided-tour/target-maps/01-multiple-deploy-items/component/commands/component.sh).

The component itself is specified here:
  - [component configuration](component/components.yaml)
  - [blueprints](https://github.com/gardener/landscaper/blob/master/docs/guided-tour/target-maps/01-multiple-deploy-items/component/blueprint) 

## Installing the example

The procedure to install example is as follows:

1. On the Landscaper resource cluster create 
  - namespace `cu-example`

2. On the Landscaper resource cluster, in namespace `cu-example` create
  - different targets `cluster-blue`, `cluster-green`, `cluster-red`, `cluster-yellow`, `cluster-orange`. 
    Usually these targets contain access data to different target clusters but for simplicity we use the same target 
    cluster for all of them.
  - a Context `landscaper-examples` and a root Installation `multiple-items`
  - a DataObject `config` containing the data for the different config maps which will be deployed on the different
    target clusters.

The templates for these resources can be found [here](component/installation) and could be deployed with 
this [script](component/commands/deploy-k8s-resources.sh). Adapt the [settings](component/commands/settings) file
such that the entry `TARGET_CLUSTER_KUBECONFIG_PATH` points to the kubeconfig of the target cluster to which the
config maps should be deployed.

Now you should see a successful installation on your Landscaper resource cluster:

```
kubectl get inst -n cu-example multiple-items     
          
NAME             PHASE       EXECUTION        AGE
multiple-items   Succeeded   multiple-items   14s
```

Let's have a deeper look into the resources of the example. The root Installation `multiple-items` contains as import 
a target map importing three of the five deployed targets. An entry of the target map consists of a logical name (blue, green etc.)
and the name of the k8s target object (cluster-blue, cluster-green etc.) 

```
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

Furthermore, the root Installation imports the DataObject `config` which contains the configuration for the different 
config maps which should be deployed to the different target clusters which looks as follows (see also 
[here](component/installation/dataobject.yaml.tpl)):

```yaml
...
data:
  blue:
    color: blue
    cpu: 100m
    memory: 100Mi
  green:
    color: green
    cpu: 120m
    memory: 120Mi
...

```

The root installation references this [blueprint](component/blueprint/blueprint.yaml) which imports the `config` as well
as the target map. The Blueprint creates a DeployItem for every target provided by the target map in its
[deployExecution](component/blueprint/deploy-execution.yaml). Therefore, it iterates over the target map with the 
following expression:

```
{{ range $key, $target := .imports.clusters }}
```

Thereby, the variable `$key` is set on the logical names of the target from the Installation (red, green, blue). 
This allows 

- to give the DeployItems a stable name `name: di-{{ $key }}` which is important if you change the input targets later, 
  to logically identify which DeployItems were removed and added.
- address the corresponding data in the `config` with `{{- index $config $key | toYaml | nindent 14 }}` which should
  be provided to the config maps on the target clusters. 

On the Landscaper resource cluster you see the three DeployItems each with the corresponding color in its name:

```
kubectl get di -n cu-example                     
NAME                            TYPE                                            PHASE       EXPORTREF   AGE
multiple-items-di-blue-jx5qc    landscaper.gardener.cloud/kubernetes-manifest   Succeeded               2d22h
multiple-items-di-green-nwj4v   landscaper.gardener.cloud/kubernetes-manifest   Succeeded               2d22h
multiple-items-di-red-8zn7r     landscaper.gardener.cloud/kubernetes-manifest   Succeeded               2d22h
```

On the target cluster you see the deployed config maps with the corresponding color in their names:

```
kubectl get cm -n example                                                          
NAME                     DATA   AGE
compose-map-exec-blue    1      21s
compose-map-exec-green   1      21s
compose-map-exec-red     1      21s
```

Every config map contains the right data from the `config` object.

## Updating the Root Installation

Assume you want to update your deployments such that instead of the config maps `compose-map-exec-blue`,
`compose-map-exec-green` and `compose-map-exec-red`, the config maps `compose-map-exec-blue`,
`compose-map-exec-red`, `compose-map-exec-yellow` and `compose-map-exec-orange` are deployed. Therefore, you
just have to adapt the import target map of you root installation as follows:

```
  imports:
  
    targets:
    - name: clusters
      targetMap:
        blue: cluster-blue
        red: cluster-red
        yellow: cluster-yellow
        orange: cluster-orange
        
    data:
    - dataRef: config
      name: config
      
    ...
```

And you get your intended new set of config maps, whereby the config map `compose-map-exec-green` is deleted:

```
kubectl get cm -n example                                                          
NAME                     DATA   AGE
compose-map-exec-blue    1      21s
compose-map-exec-red     1      21s
compose-map-exec-yellow  1      21s
compose-map-exec-orange  1      21s
```

An important point in this example is the following. When you are removing a target from the target map of the root 
Installation, the target itself is not allowed to be removed as long as the corresponding DeployItem is not 
successfully deleted. This is because the target is still used for accessing the target cluster for the deletion of 
the installed k8s resources. This is only a problem if you create the DeployItems in the top level blueprint. 
If you are using Subinstallations to create the DeployItems, this problem does not occur, because the targets are 
copied into an internal context for the Subinstallations and only deleted after the deletion of the corresponding
Subinstallations. There are examples in this guided tour presenting such a solution.

## Removal of the test data

You can remove the Installation, the context and targets by first executing [this](component/commands/delete-inst.sh) 
script to delete the Installation and then [this](component/commands/delete-rest.sh) to remove the rest.

# Multiple Subinstallations Example + Others

In this example, we show again how target maps can be used to deploy an artefact to a variable number of target clusters. 

For prerequisites, see [here](../../README.md#prerequisites-and-basic-definitions) and check out the [first](../01-multiple-deploy-items/README.md) 
and [second target map example](../02-targetmap-ref).

This example is a modification of the first and second target map example. It achieves the same result, i.e. deploying 
config maps with specific data to a variable number of target clusters, but it uses several Subinstallations for that.

## Example description

This example presents again a component, which gets as input a map of targets and a data object containing
configuration data for every input target. The import data are forwarded to a Subinstallation.  
For every input target, the Subinstallation creates a DeployItem which deploys a config map with the right data.

The example component is stored 
[here](eu.gcr.io/gardener-project/landscaper/examples/component-descriptors/github.com/gardener/guided-tour/targetmaps/guided-tour-multiple-subinst). 
If you want to upload the component to another registry, you can just adapt the [settings](component/commands/settings) 
file and execute the component build and upload script [here](component/commands/component.sh).

The component itself is specified here:
  - [component configuration](component/components.yaml)
  - [blueprints](component/blueprint) 

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

The templates for these resources can be found [here](component/installation) and can be deployed with 
this [script](component/commands/deploy-k8s-resources.sh). Adapt the [settings](component/commands/settings) file
such that the entry `TARGET_CLUSTER_KUBECONFIG_PATH` points to the kubeconfig of the target cluster to which the
config maps should be deployed.

Now you should see successful installations on your Landscaper resource cluster:

```
kubectl get inst -n cu-example 
NAME                               PHASE       EXECUTION                          AGE
multiple-subinst                   Completing                                      7s
multiple-subinst-sub-blue-9nsp9    Succeeded    multiple-subinst-sub-blue-9nsp9    6s
multiple-subinst-sub-green-wh74m   Succeeded    multiple-subinst-sub-green-wh74m   6s
multiple-subinst-sub-red-tvrrm     Succeeded    multiple-subinst-sub-red-tvrrm     6s
```

Let's have a deeper look into the resources of the example. The root Installation `multiple-subinst` is quite similar
to the first and second target map example and contains as import a target map importing three of the five deployed 
targets as well as configuration data.

The root installation references this [blueprint](component/blueprint/root/blueprint.yaml) which creates a Subinstallation 
for every target. This is done [here](component/blueprint/root/subinst-execution.yaml), where the following expression 
loops over all input targets:

```
{{ range $key, $target := .imports.rootclusters }}
```

Every Subinstallation gets as import one target from the target map with the following expression:

```
    imports:
      targets:
        - name: cluster
          target: rootclusters[{{ $key }}]
```

In the import data mapping part, the key is forwarded to the [blueprint](component/blueprint/sub/blueprint.yaml) of 
the Subinstallations such that the DeployItems can get a stable name, which is important for a later deletion if
particular targets are removed. Furthermore, the correct entry from the data is extracted and provided:

```
    importDataMappings:
      instanceName: {{ $key }}
      config:
        {{- $config | toYaml | nindent 8 }}
```

Every Subinstallation creates one DeployItem to deploy one config map.

You can see successful DeployItems on your Landscaper resource cluster with:

```
kubectl get di -n cu-example                                                  
NAME                                              TYPE                                            PHASE          AGE
multiple-subinst-sub-blue-9nsp9-di-blue-rhf4p     landscaper.gardener.cloud/kubernetes-manifest   Succeeded      10s
multiple-subinst-sub-green-wh74m-di-green-gt475   landscaper.gardener.cloud/kubernetes-manifest   Succeeded      10s
multiple-subinst-sub-red-tvrrm-di-red-8wvf2       landscaper.gardener.cloud/kubernetes-manifest   Succeeded      10s
```
---
title: Data Flow Between Subinstallations
sidebar_position: 2
---

# Data Flow Between Subinstallations

In the previous examples, blueprints defined deploy items. In complex scenarios however, a blueprint can also consist of 
subinstallations. Each subinstallation has itself a blueprint, which can again consist of subinstallations or define deploy items.

The blueprint in this example has three subinstallations. All three subinstallations use the same blueprint, which 
creates a ConfigMap. Therefore, the scenario comprises the following resources:
- the [installation](./installation/installation.yaml.tpl) &mdash; to distinguish it from the subinstallations, we call it the "root" installation.
- the [blueprint of the root installation](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/subinstallations/export-import/blueprints/root) &mdash; 
  it defines its subinstallations:
  - [subinstallation-1](./blueprints/root/subinstallation-1.yaml),
  - [subinstallation-2](./blueprints/root/subinstallation-2.yaml),
  - [subinstallation-3](./blueprints/root/subinstallation-3.yaml).  
  
  At runtime, the Landscaper will create actual Installation resources for them.
- the common [blueprint of the subinstallations](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/subinstallations/export-import/blueprints/sub).


### The Data Flow 

The blueprint of the subinstallations imports the name of the Configmap that it creates (import parameter `configmap-name-in`)
and exports a slightly modified name (export parameter `configmap-name-out`) which serves as import for the next subinstallation.
The blueprint of the root installation imports the name of the first ConfigMap (import parameter `configmap-name-base`), 
and exports the list of all configmap names (export parameter `configmapNames`).

The following diagram shows the data flow. Installations are displayed in yellow, blueprint in blue, 
and DataObjects which contain the export and import values in grey (The parameter for the Target is omitted.) 

![export-import](./images/export-import.png)

The order in which subinstallations are processed is implicitely derived from the exports and imports. 
In our case the subinstallations are processed in a chain, because the second imports an export of the first, and
the third imports an export of the second.


### Analogy with Functions and Function Calls

There is an analogy in which blueprints correspond to functions, and Installations to function calls. In this analogy,
our example would correspond to the following code, in which a function `rootBlueprint` calls three times a 
function `subBlueprint`:

```go
// root installation = call of function "rootBlueprint"
rootBlueprint(cluster, "example-configmap")

// root blueprint
func rootBlueprint(cluster Target, configmapNameBase string) (configmapNames []string) {

    // 1st subinstallation = 1st call of function "subBlueprint"
    configmapName2 := subBlueprint(cluster, configmapNameBase)

    // 2nd subinstallation = 2nd call of function "subBlueprint"
    configmapName3 := subBlueprint(cluster, configmapName2)

    // 3rd subinstallation = 3rd call of function "subBlueprint"
    _ = subBlueprint(cluster, configmapName3)
	
	return []string {configmapNameBase, configmapName2, configmapName3}
} 

// sub blueprint
func subBlueprint(cluster Target, configmapNameIn string) (configmapNameOut string) {
	...
}
```


## Procedure

1. On the target cluster, create a namespace `example`. It is the namespace into which we will deploy the ConfigMaps.

2. On the Landscaper resource cluster, create a namespace `cu-example`.

3. On the Landscaper resource cluster, in namespace `cu-example`, create a Target `my-cluster` containing a
   kubeconfig for the target cluster, a Context `landscaper-examples`, a DataObject `do-configmap-name-base`, 
   and an Installation `export-import`. There are templates for these resources in the directory
   [installation](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/subinstallations/export-import/installation).
   To apply them:
    - adapt the [settings](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/subinstallations/export-import/commands/settings) file
      such that the entry `RESOURCE_CLUSTER_KUBECONFIG_PATH` points to the kubeconfig of the resource cluster,
      and the entry `TARGET_CLUSTER_KUBECONFIG_PATH` points to the kubeconfig of the target cluster,
    - run the [deploy-k8s-resources script](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/subinstallations/export-import/commands/deploy-k8s-resources.sh),
      which will template and apply the Target, Context, DataObject, and Installation.


As a result, the three subinstallations have deployed three ConfigMaps on the target cluster in namespace `example`:

```shell
$ kubectl get configmaps -n example

NAME                    DATA   AGE
example-configmap       1      41s
example-configmap-x     1      36s
example-configmap-x-x   1      31s
```

The export parameter of the root installation is written to the DataObject `do-configmap-names` in namespace `cu-example`
on the resource cluster. The export execution of the root blueprint defines the value of this export parameter, namely 
the list of the three ConfigMap names.

```shell
$ kubectl get dataobject -n cu-example do-configmap-names -o yaml

apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DataObject
metadata:
  name: do-configmap-names
  namespace: cu-example
  ...
data:
- example-configmap
- example-configmap-x
- example-configmap-x-x
```


## Cleanup

You can remove the Installation with the
[delete-installation script](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/subinstallations/export-import/commands/delete-installation.sh).
When the Installation is gone, you can delete the Context and Target with the
[delete-other-k8s-resources script](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/subinstallations/export-import/commands/delete-other-k8s-resources.sh).



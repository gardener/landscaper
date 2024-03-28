---
title: Import Parameters
sidebar_position: 1
---

# Import Parameters

This example is a modification of [Blueprint and Helm Chart Resources in a Component Version](../../components/helm-chart). 
We add import parameters to the blueprint to make some of its settings configurable: the Helm release name and namespace, 
as well as the text of the echo server. The `Installation` will provide values for these parameters which it reads
from `DataObject` custom resources. 

In some sense, you can view a blueprint as a function that executes a deployment,
and the `Installation` as a call of this function providing values for its parameters.


## Declaring Import Parameters

The [blueprint](./blueprint/blueprint.yaml) of the present example declares three import parameters:
the target parameter `cluster`, and the data parameters `release` and `values`. 

```yaml
imports:
- name: cluster
  type: target
  targetType: landscaper.gardener.cloud/kubernetes-cluster

- name: release
  type: data
  schema:
    type: object

- name: values
  type: data
  schema:
    type: object
```

In general, we distinguish parameters of type `target`, `targetMap`, and `data`.

To define target parameters more detailed, they have a `targetType`. However, currently there is only one supported 
target type, namely `landscaper.gardener.cloud/kubernetes-cluster`. A blueprint can import more than one target. 
For example, the inline blueprint [here](../../basics/multiple-deployitems/installation/installation.yaml)
has two target imports. A parameter of type `targetMap` allows to import several targets whose number is not 
specified by the blueprint. There is a separate chapter on target maps.

To define data parameters more precisely, they have a `schema`, i.e. a json schema that describes the structure 
of the data.

For more details, see [Import Definitions](../../../usage/Blueprints.md#import-definitions)


## Using Import Parameters

A blueprint can use its import parameters in the templating of `DeployItems`. 
The value of an import parameter can be accessed by `.imports.<parameter name>`.
If a parameter is of type `object`, you can access a field by appending the path to the field, for example:

```yaml
name: {{ .imports.release.name }}
namespace: {{ .imports.release.namespace }}
```

For more details, see [Rendering](../../../usage/Blueprints.md#rendering)


## Binding Values to Import Parameters

We have stored the values for the two import parameters in `DataObject` custom resources
[dataobject-release.yaml.tpl](./installation/dataobject-release.yaml.tpl) and 
[dataobject-values.yaml.tpl](./installation/dataobject-values.yaml.tpl).

The `imports` section of the `Installation` connects each import parameters of the blueprint 
with a corresponding `DataObject` or `Target`.  

```yaml
imports:
  targets:
    - name:    <name of the import parameter of the blueprint>
      target:  <name of the Target custom resource containing the kubeconfig of the target cluster>
  data:
    - name:    <name of the import parameter of the blueprint>
      dataRef: <name of a DataObject containing the parameter value>
```

The `DataObjects` and `Targets` must belong to the same namespace as the `Installation`. Note that it is also possible to store 
parameter values in `ConfigMaps` or `Secrets`. For more details, see [Imports](../../../usage/Installations.md#imports).


## Procedure

The procedure to install the helm chart with Landscaper is as follows:

1. On the Landscaper resource cluster, create a namespace `cu-example`.

2. On the Landscaper resource cluster, in namespace `cu-example`, create a Target, a Context, two DataObjects, and an
   Installation. There are templates for these resources in the directory
   [installation](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/import-export/import-parameters/installation).
   To apply them:
   - adapt the [settings](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/import-export/import-parameters/commands/settings) file
     such that the entry `TARGET_CLUSTER_KUBECONFIG_PATH` points to the kubeconfig of the target cluster,
   - run the [deploy-k8s-resources script](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/import-export/import-parameters/commands/deploy-k8s-resources.sh),
     which will template and apply the Target, Context, DataObjects, and Installation.

3. To try out the echo server, first define a port forwarding on the target cluster:

   ```shell
   kubectl port-forward -n example-2 service/echo 8080:80
   ```

   Then open `localhost:8080` in a browser. The response should be "Hello, Landscaper!", which is the text defined
   in the [dataobject-values.yaml.tpl](./installation/dataobject-values.yaml.tpl).



## Cleanup

You can remove the Installation with the
[delete-installation script](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/components/helm-chart/commands/delete-installation.sh).
When the Installation is gone, you can delete the Target, Context, and DataObjects with the
[delete-other-k8s-resources script](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/components/helm-chart/commands/delete-other-k8s-resources.sh).



## References

[Import Definitions](../../../usage/Blueprints.md#import-definitions)

[Imports](../../../usage/Installations.md#imports)

[Rendering](../../../usage/Blueprints.md#rendering)

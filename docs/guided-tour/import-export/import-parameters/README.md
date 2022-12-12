# Import Parameters

This example is a modification of the [Echo Server Example](../../blueprints/echo-server). 
We add import parameters to the blueprint to make some of its settings configurable: the Helm release name and namespace, 
as well as the text of the echo server. The `Installation` will provide values for these parameters which it reads
from `DataObject` custom resources. 

In some sense, you can view a blueprint as a function that executes a deployment,
and the `Installation` as a call of this function providing values for its parameters.


## Declaring Import Parameters

The [blueprint](./blueprint/blueprint.yaml) of the present example declares three import parameters:
the target parameter `cluster`, and the data parameters `release` and `text`. 

```yaml
imports:
- name: cluster
  type: target
  targetType: landscaper.gardener.cloud/kubernetes-cluster

- name: release
  type: data
  schema:
    type: object

- name: text
  type: data
  schema:
    type: string
```

In general, we distinguish parameters of type `target`, `targetList`, and `data`.

To define target parameters more detailed, they have a `targetType`. However, currently there is only
one supported target type, namely `landscaper.gardener.cloud/kubernetes-cluster`. A blueprint can import more than one
target. For example, the inline blueprint
[here](../../basics/multiple-deployitems/installation/installation.yaml)
has two target imports. A parameter of type `targetList` allows to import a list of targets whose length is not 
specified by the blueprint.

To define data parameters more precisely, they have a `schema`, i.e. a json schema that describes the structure 
of the data.

For more details, see [Import Definitions](../../../usage/Blueprints.md#import-definitions)


## Using Import Parameters

A blueprint can use its import parameters in the templating of `DeployItems`. 
The value of an import parameter can be accessed by `.imports.<parameter name>`, for example:

```yaml
text: {{ .imports.text }}
```

If a parameter is of type `object`, you can access a field by appending the path to the field, for example:

```yaml
name: {{ .imports.release.name }}
namespace: {{ .imports.release.namespace }}
```

For more details, see [Rendering](../../../usage/Blueprints.md#rendering)

## Binding Values to Import Parameters

We have stored the values for the two import parameters in `DataObject` custom resources
[dataobject-release.yaml](./installation/dataobject-release.yaml) and 
[dataobject-text.yaml](./installation/dataobject-text.yaml).

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

1. Add the kubeconfig of your target cluster to your [target.yaml](installation/target.yaml).

2. On the Landscaper resource cluster, create namespace `example` and apply
   the [context.yaml](./installation/context.yaml),
   the [dataobject-release.yaml](./installation/dataobject-release.yaml),
   the [dataobject-text.yaml](./installation/dataobject-text.yaml),
   the [target.yaml](installation/target.yaml), and the [installation.yaml](installation/installation.yaml):

   ```shell
   kubectl create ns example
   kubectl apply -f <path to context.yaml>
   kubectl apply -f <path to dataobject-release.yaml>
   kubectl apply -f <path to dataobject-text.yaml>
   kubectl apply -f <path to target.yaml>
   kubectl apply -f <path to installation.yaml>
   ```

3. To try out the echo server, first define a port forwarding on the target cluster:

   ```shell
   kubectl port-forward -n example-2 service/echo 8080:80
   ```

   Then open `localhost:8080` in a browser.

   The response should be "Hello, Landscaper!", which is the text defined
   in the [dataobject-text.yaml](./installation/dataobject-text.yaml).


## References

[Import Definitions](../../../usage/Blueprints.md#import-definitions)

[Imports](../../../usage/Installations.md#imports)

[Rendering](../../../usage/Blueprints.md#rendering)

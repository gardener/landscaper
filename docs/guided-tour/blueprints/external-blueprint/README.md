# An Installation with an Externally Stored Blueprint

For prerequisites, see [here](../../README.md#prerequisites-and-basic-definitions).

In the following example, we will demonstrate how deployment procedures can be made reusable, such that they can be used in several Installations.

The installations in the previous examples had two main parts: the import of a Target and a Blueprint. The Target defines on which cluster something should be deployed. The Blueprint defines the general deployment procedure. It is this part that we want to make reusable. 

For example, if we want to deploy the same Helm chart on several clusters, we would create a Target and an Installation for each cluster. All these Installations would reference the same blueprint, instead of containing it inline. This becomes possible if we store the Blueprint at a referencable location, e.g. an OCI registry.

## The Example Blueprint

You can find the blueprint for the current example [here](./blueprint). Note that the blueprint is a directory, and not just the [blueprint/blueprint.yaml](./blueprint/blueprint.yaml) file. In future examples the blueprint directory will contain further files.

We have uploaded the blueprint
[here](https://eu.gcr.io/gardener-project/landscaper/examples/blueprints/guided-tour/external-blueprint) into an OCI registry, from where the Landscaper can access it. You can find the commands which we have used to upload the blueprint in this script: [commands/push-blueprint.sh](./commands/push-blueprint.sh).


## The Component Descriptor

An Installation references its blueprint indirectly via a so-called [component descriptor](../../../concepts/Glossary.md#_component-descriptor_). In general, a component descriptor is used to describe all required resources for the deployment of a component. The [component descriptor for the current example](component-descriptor) contains only one resource, namely the 
blueprint. As you might have noticed when clicking on the link to the component descriptor for the current example, there currently exist two different versions - v2 and v3. While v3 theoretically has more capabilities than v2, they are functionally identical for the scope of these examples. For more information, refer to https://ocm.software/.   
We will describe component descriptors more detailed in later examples.

We have uploaded the component descriptor for our example [here](https://eu.gcr.io/gardener-project/landscaper/examples/component-descriptors/github.com/gardener/landscaper-examples/guided-tour/external-blueprint) into an OCI registry, from where the Landscaper can access it. You can find the command which we have used to upload the component descriptor in this script: 
[commands/push-component-descriptor.sh](./commands/push-component-descriptor.sh).


## Referencing the Blueprint in the Installation

The [Installation](./installation/installation.yaml) references the component descriptor and blueprint as follows:  

```yaml
context: landscaper-examples

componentDescriptor:
 ref:
   componentName: github.com/gardener/landscaper-examples/guided-tour/external-blueprint
   version: 1.0.0

blueprint:
 ref:
   resourceName: blueprint
```

- The field `context` contains the name of a custom resource of kind [Context](../../../usage/Context.md) in the same namespace as the Installation on the Landscaper resource cluster. [Our Context resource](./installation/context.yaml) contains the information in which registry the component descriptor and blueprint are stored.

- The fields `componentDescriptor.ref.componentName` and `componentDescriptor.ref.version` are then used to locate the component descriptor in that registry.

- The [component descriptor](component-descriptor) contains a list of resources, each of which has a name. Field `blueprint.ref.resourceName` in the Installation specifies the name of the blueprint resource in the component descriptor. 


## Procedure

The procedure to deploy the helm chart with the Landscaper is:

1. Insert the kubeconfig of your target cluster into file [target.yaml](installation/target.yaml).

2. On the Landscaper resource cluster, create namespace `example` and apply the [context.yaml](./installation/context.yaml), the [target.yaml](installation/target.yaml), and the [installation.yaml](installation/installation.yaml):

   ```shell
   kubectl create ns example
   kubectl apply -f <path to context.yaml>
   kubectl apply -f <path to target.yaml>
   kubectl apply -f <path to installation.yaml>
   ```


## References 

[Blueprints](../../../usage/Blueprints.md)

[Context](../../../usage/Context.md)

[Accessing Blueprints](../../../usage/AccessingBlueprints.md)

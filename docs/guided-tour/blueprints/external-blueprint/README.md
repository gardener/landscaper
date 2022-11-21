# An Installation with an Externally Stored Blueprint

For prerequisites see [here](../README.md#prerequisites-and-basic-definitions).

In this example we demonstrate how deployment procedures can be made reusable, such that they can be reused in 
several Installations.

The installations in the previous examples had two main parts: the import of a Target and a blueprint.
The Target defines on which specific cluster something should be deployed. 
The blueprint defines the general deployment procedure. It is this part that we want to make reusable.
For example, if we want to deploy the same Helm chart on several clusters, we would create a Target and an Installation
for each cluster. All these Installations would only reference the blueprint instead of containing it inline.
This is only possible if we store the blueprint at a referencable location, e.g. an OCI registry.


## The Example Blueprint

You can find the blueprint for the current example [here](./blueprint). 
Note that the blueprint is a directory, and not just the [blueprint/blueprint.yaml](./blueprint/blueprint.yaml) file.
In future examples the blueprint directory will contain further files.

We have uploaded the blueprint
[here](https://eu.gcr.io/gardener-project/landscaper/examples/blueprints/external-blueprint)
into an OCI registry, from where the Landscaper can access it.
You can find the command that we have used to upload the blueprint in the script 
[commands/push-blueprint.sh](./commands/push-blueprint.sh).


## The Component Descriptor

An Installation references its blueprint indirectly via a so-called 
[component descriptor](../../../concepts/Glossary.md#_component-descriptor_).
In general, we use the component descriptor to collect all required resources for a deployment of a component.
The [component descriptor for the current example](./component-descriptor.yaml) contains only one resource, namely the 
blueprint. We will describe component descriptors more detailed in later examples.

We have uploaded the component descriptor for our example
[here](https://eu.gcr.io/gardener-project/landscaper/examples/component-descriptors/github.com/gardener/landscaper-examples/guided-tour/external-blueprint)
into an OCI registry, from where the Landscaper can access it.
You can find the command that we have used to upload the component descriptor in the script
[commands/push-blueprint.sh](./commands/create-and-push-component.sh).


## Referencing the Blueprint in the Installation

The [Installation](./installation/installation.yaml) references the component descriptor and blueprint as follows:  

```yaml
componentDescriptor:
 ref:
   componentName: github.com/gardener/landscaper-examples/guided-tour/external-blueprint
   version: 1.0.0
   repositoryContext:
     baseUrl: eu.gcr.io/gardener-project/landscaper/examples
     type: ociRegistry

blueprint:
 ref:
   resourceName: blueprint
```

- The field `componentDescriptor.ref.repository` specifies the registry in which the component descriptor is stored.
- The fields `componentDescriptor.ref.componentName` and `componentDescriptor.ref.version` are then used to locate the 
component descriptor in that registry. 
- The component descriptor contains a list of resources, each of which has a name.
Field `blueprint.ref.resourceName` in the Installation specifies the name of the blueprint resource in the component 
descriptor. 


## Procedure

The procedure is the same as before:

1. Insert in file [target.yaml](installation/target.yaml) the kubeconfig of your target cluster.

2. On the Landscaper resource cluster, create namespace `example` and apply
   the [target.yaml](installation/target.yaml) and the [installation.yaml](installation/installation.yaml):

   ```shell
   kubectl create ns example
   kubectl apply -f <path to target.yaml>
   kubectl apply -f <path to installation.yaml>
   ```

Storing the blueprint externally has the advantage that we can reuse it. If we want to deploy the same
Helm chart to a second cluster, we create a second Target and a second Installation referencing the same component and
blueprint.


## References 

[Blueprints](../../../usage/Blueprints.md)

[Accessing Blueprints](../../../usage/AccessingBlueprints.md)

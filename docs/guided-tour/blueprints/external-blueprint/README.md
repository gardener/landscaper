---
title: External Blueprint
sidebar_position: 1
---

# An Installation with an Externally Stored Blueprint

In this example, we will demonstrate how deployment procedures can be made reusable, such that they can be used
in several Installations.

For prerequisites, see [here](../../README.md).

The installations in the previous examples had two main parts: the import of a Target and a Blueprint. The Target
defines on which cluster something should be deployed. The Blueprint defines the general deployment procedure. It is
this part that we want to make reusable. 

For example, if we want to deploy the same Helm chart on several clusters, we would create a Target and an Installation
for each cluster. All these Installations would reference the same blueprint, instead of containing it inline. This
becomes possible if we store the Blueprint at a referencable location, e.g. an OCI registry.

## The Example Blueprint

You can find the blueprint for the current example [here](https://github.com/gardener/landscaper/blob/master/docs/guided-tour/blueprints/external-blueprint/blueprint). Note that the blueprint is a directory, and not just the [blueprint/blueprint.yaml](./blueprint/blueprint.yaml) file. In future examples the blueprint directory will contain further files.

We have uploaded the blueprint
[here](https://eu.gcr.io/gardener-project/landscaper/examples/blueprints/guided-tour/external-blueprint) into an OCI
registry, from where the Landscaper can access it. You can find the commands, which we have used to upload the blueprint
in this script: [commands/blueprint.sh](./commands/blueprint.sh).


## Components

> **_NOTE:_** **To follow along the following section, be sure to set the `useOCM: true` feature switch in the 
> values.yaml, as shown [here](https://github.com/gardener/landscaper/blob/master/docs/installation/install-landscaper-controller.md#configuration-through-valuesyaml).**

To consume the blueprint from such a referencable location, it has to be contained in a component.

A component is a concept introduced by the [Open Component Model](https://github.com/open-component-model/ocm). In
short, technically, a component consists of a *component-descriptor* and a number of *blobs* 
(= arbitrary binary objects).  
The file system representation of our component used for this example is shown [here](./components/component). 
The *component-descriptor* describes the overall component. As you can see in the *component-descriptor*, the component
only contains a single resource - the *blueprint*. And we reference that blueprint at an external location, the *OCI
registry*.  

Alternatively, instead of referencing the blueprint at an external location, we could have embedded it in the
component as a so-called local blob. Then, the *blob* directory in the file system representation would not have been 
empty. It would have contained the blueprint (typically as a tar archive), as demonstrated 
[here](./components/local-blob-component).

The most convenient way to generate such components is through corresponding *component configuration files*. The
*component configuration file* used to create the component for this example is shown 
[here](./config-files/components.yaml): 

```yaml
components:
  - name: github.com/gardener/landscaper-examples/guided-tour/external-blueprint
    version: 2.0.0
    provider:
      name: internal
    resources:
      - name: blueprint
        type: landscaper.gardener.cloud/blueprint
        version: 1.0.0
        access:
          type: ociArtifact
          imageReference: eu.gcr.io/gardener-project/landscaper/examples/blueprints/guided-tour/external-blueprint:1.0.0
```

If you were to prefer to embed the blueprint in the component as a local blob, instead of an `access:...`, you would have to
specify an `input:...` as demonstrated [here](./config-files/local-blob-components.yaml). The commands used to create the actual component based on the *component configuration file* and to upload this
component to an OCI registry can be found [here](./commands/component.sh).

>**Tip:** If you need the same resource in multiple components, instead of copy-pasting, you might want to use 
> [*resource configuration files*](https://ocm.software/docs/guides/getting-started-with-ocm/#using-a-resources-file).

## Referencing the Blueprint in the Installation

The [Installation](./installation/installation.yaml) references the component and blueprint as follows:  

```yaml 
context: landscaper-examples

componentDescriptor:
  ref: 
    componentName: github.com/gardener/landscaper-examples/guided-tour/external-blueprint 
    version: 2.0.0

blueprint: 
  ref: 
    resourceName: blueprint 
```

- The field `context` contains the name of a custom resource of kind [Context](../../../usage/Context.md) in the same
  namespace as the Installation on the Landscaper resource cluster. [Our Context resource](./installation/context.yaml)
  contains the information in which registry the component descriptor and blueprint are stored.

- The fields `componentDescriptor.ref.componentName` and `componentDescriptor.ref.version` are then used to locate the
  component descriptor in that registry.

- The [component descriptor](./component-archive/v2-external/component-descriptor.yaml) contains a list of resources,
  each of which has a name. Field `blueprint.ref.resourceName` in the Installation specifies the name of the blueprint
  resource in the component descriptor. Thereby, it is completely transparent for the installation whether the component
  references the blueprint as an external resource or embeds it as a local blob. 


## Procedure

The procedure to deploy the helm chart with the Landscaper is:

1. Insert the kubeconfig of your target cluster into file [target.yaml](installation/target.yaml).

2. On the Landscaper resource cluster, create namespace `example` and apply the
[context.yaml](./installation/context.yaml), the [target.yaml](installation/target.yaml), and the
[installation.yaml](installation/installation.yaml):

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

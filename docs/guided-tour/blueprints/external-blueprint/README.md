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

## Component Versions

To consume the blueprint from such a referencable location, it has to be contained in a component version. A component 
version is a concept introduced by the [Open Component Model](https://github.com/open-component-model/ocm). In short, technically, 
a component version consists of a *component-descriptor* and a number of *blobs* (= arbitrary binary objects).  

The most convenient way to generate such component versions is through corresponding *component configuration files*. 
The *component configuration file* used to create the component version for this example is shown
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

The commands used to create the actual component version based on the *component configuration file* and to upload this
component version to an OCI registry can be found [here](./commands/component.sh). The first command creates
a file system representation of the component version and the second uploads this to the registry:

```
ocm add components --create --file ../tour-ctf ../config-files/components.yaml
ocm transfer ctf --enforce ../tour-ctf eu.gcr.io/gardener-project/landscaper/examples
```

We have uploaded the component version into this [location](https://console.cloud.google.com/gcr/images/gardener-project/eu/landscaper/examples/component-descriptors/github.com/gardener/landscaper-examples/guided-tour/external-blueprint) to our public OCI registry. You could download
the component version with the following command which more or less provides you a file system representation
of the example component version:

```
ocm download component eu.gcr.io/gardener-project/landscaper/examples//github.com/gardener/landscaper-examples/guided-tour/external-blueprint:2.0.0 -O <your location on the file system>
```

[Here](components/component) you see the downloaded component version. The perhaps most important part is the 
[component descriptor](components/component/component-descriptor.yaml) which mainly contains the adapted data from the 
configuration file. In its `resource` section, you find all specified artefacts including their access data. In our
example only the reference to the blueprint is contained here.

Alternatively, instead of referencing the blueprint at an external location, we can embed it in the
component version as a so-called local blob. Therefore, in the *component configuration file* instead of an `access:...`, 
you would have to specify an `input:...` entry, as demonstrated [here](./config-files/local-blob-components.yaml).
The  configured path must reference the location of the blueprint directory on the file system. The creation 
and uploading of the component version is the same as before. If you download this component version as before
you get [this](components/local-blob-component) file system representation. Now the component descriptor references
the blueprint as a local blob. This local blob is contained in the file system representation of the component
version in the [blobs folder](components/local-blob-component/blobs).

To embed a blueprint into a component version as a local blob is usually the preferred approach.

For more details about component versions check out the [OCM Getting Started Guide](https://ocm.software/docs/guides/getting-started-with-ocm/). In the OCM documentation
you also find more about the advantages of using components to describe your artefacts like transportability, signing, 
scanning etc.

## Referencing the Blueprint (and Component Version) in the Installation

The [Installation](./installation/installation.yaml) references the component version and blueprint as follows:  

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
  contains the information in which registry the component version with the component descriptor is stored. 

- The fields `componentDescriptor.ref.componentName` and `componentDescriptor.ref.version` are then used to locate the
  component descriptor in that registry.

- The [component descriptor](components/component/component-descriptor.yaml) contains a list of resources,
  each of which has a name. Field `blueprint.ref.resourceName` in the Installation specifies the name of the blueprint
  resource in the component descriptor. Thereby, it is completely transparent for the installation whether the 
  component version references the blueprint as an external resource or embeds it as a local blob. 

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

# An Installation with an Externally Stored Blueprint

For prerequisites, see [here](../../README.md#prerequisites-and-basic-definitions).

In the following example, we will demonstrate how deployment procedures can be made reusable, such that they can be used
in several Installations.

The installations in the previous examples had two main parts: the import of a Target and a Blueprint. The Target
defines on which cluster something should be deployed. The Blueprint defines the general deployment procedure. It is
this part that we want to make reusable. 

For example, if we want to deploy the same Helm chart on several clusters, we would create a Target and an Installation
for each cluster. All these Installations would reference the same blueprint, instead of containing it inline. This
becomes possible if we store the Blueprint at a referencable location, e.g. an OCI registry.

## The Example Blueprint

You can find the blueprint for the current example [here](./blueprint). Note that the blueprint is a directory, and not
just the [blueprint/blueprint.yaml](./blueprint/blueprint.yaml) file. In future examples the blueprint directory will
contain further files.

We have uploaded the blueprint
[here](https://eu.gcr.io/gardener-project/landscaper/examples/blueprints/guided-tour/external-blueprint) into an OCI
registry, from where the Landscaper can access it. You can find the commands which we have used to upload the blueprint
in this script: [commands/push-blueprint.sh](./commands/push-blueprint.sh).


## Components and Component Descriptors

An Installation may reference its blueprints via so-called
[component-descriptors](../../../concepts/Glossary.md#_component-descriptor_).  A component descriptor describes a
component, or rather, a specific component version. In general, a component version is a container for all required
resources for the deployment of a specific version of an application or software system. In this example, the
application is the hello-world application deployable with the landscaper through an external blueprint. Thereby, the
external blueprint is the only resource required for the deployment.  A component version may either contain a resource
through referencing it at an external location (such as an oci registry) or through embedding it as a local blob.

#### Component Version with External Resource A file system representation of a component version containing the
resource through an external reference is shown [here](./component-archive/v2-external).  The corresponding
component-descriptor describing that component version is stored as a top-level file. The component descriptor contains
only a single _resource_ with the `name: blueprint`. This resource has an _access_ of `type: ociArtifact` that contains
a reference to the previously uploaded image.  Besides the component-descriptor, there is a directory called blobs at
the top-level. This is where the local blobs of a embedded resource would be located. Since the blueprint is the only
resource of this component version and it is contained through an external reference, this directory is empty.

#### Component Version with Local Resource A file system representation of a component version containing the resource
as a local blob is shown [here](./component-archive/v2-local).  Again, the corresponding component-descriptor describing
that component version is stored as a top-level file. Exactly as before, the component descriptor contains only a single
_resource_ with the `name: blueprint`. But now, this resource has an _access_ of `type: localBlob`. Instead of an
`imageReference`, this _access_ of `type: localBlob` has a `localReference`. This is the sha256 hash value of the
blueprint. If you open the blob directory here, you will see that it contains a file with exactly that name.
Furthermore, this _access_ has a field `mediaType`, which provides information about the format in which the blueprint
is stored, here that it is an archived and compressed (tar+gzip). The _access_ of `type: ociArtifact` did not need to
provide this information since the `ociArtifact` format is determined through the oci standard and the format of the
contents of the oci artifact is described within the artifact itself.  
  
These file system representations of component versions can then be uploaded to an oci registry.  

The commands used to create and upload the above component versions can be found
[here](./commands/upload-component-version.sh). To follow this example, you do not have to do this yourself, we have
uploaded a corresponding component version
[here](https://eu.gcr.io/gardener-project/landscaper/examples/component-descriptors/github.com/gardener/landscaper-examples/guided-tour/external-blueprint).
If you want to inspect the uploaded component version (e.g. to find out whether we uploaded the one with the external
resource or with the local resource), you can do so using the following command:  ``` ocm download componentversion
eu.gcr.io/gardener-project/landscaper/examples//github.com/gardener/landscaper-examples/guided-tour/external-blueprint
-O component-archive ``` For more information about components and related concepts, refer to the [documentation of the
ocm project](https://ocm.software/).


## Referencing the Blueprint in the Installation

The [Installation](./installation/installation.yaml) references the component descriptor and blueprint as follows:  

```yaml context: landscaper-examples

componentDescriptor: ref: componentName: github.com/gardener/landscaper-examples/guided-tour/external-blueprint version:
1.0.0

blueprint: ref: resourceName: blueprint ```

- The field `context` contains the name of a custom resource of kind [Context](../../../usage/Context.md) in the same
  namespace as the Installation on the Landscaper resource cluster. [Our Context resource](./installation/context.yaml)
  contains the information in which registry the component descriptor and blueprint are stored.

- The fields `componentDescriptor.ref.componentName` and `componentDescriptor.ref.version` are then used to locate the
  component descriptor in that registry.

- The [component descriptor](./component-archive/v2-external/component-descriptor.yaml) contains a list of resources,
  each of which has a name. Field `blueprint.ref.resourceName` in the Installation specifies the name of the blueprint
  resource in the component descriptor. Thereby, it is completely transparent for the installation whether the component
  version references the blueprint as an external resource or embeds it as a local blob. 


## Procedure

The procedure to deploy the helm chart with the Landscaper is:

1. Insert the kubeconfig of your target cluster into file [target.yaml](installation/target.yaml).

2. On the Landscaper resource cluster, create namespace `example` and apply the
[context.yaml](./installation/context.yaml), the [target.yaml](installation/target.yaml), and the
[installation.yaml](installation/installation.yaml):

   ```shell kubectl create ns example kubectl apply -f <path to context.yaml> kubectl apply -f <path to target.yaml>
   kubectl apply -f <path to installation.yaml> ```


## References 

[Blueprints](../../../usage/Blueprints.md)

[Context](../../../usage/Context.md)

[Accessing Blueprints](../../../usage/AccessingBlueprints.md)

---
title: Helm Chart Component
sidebar_position: 1
---

# Blueprint and Helm Chart Resources in a Component Version

In this section we create a **component version**, and show how to use it for a deployment of a Helm chart with the 
Landscaper.

For prerequisites, see [here](../../README.md).



## The Open Component Model

A **component version** is a concept introduced by the [Open Component Model (OCM)](https://ocm.software). In short, a 
component version consists of a number of **resources** and a **component descriptor**. Our example component version 
will have three resources: a blueprint, a Helm chart, and a Docker image.
The component descriptor lists these resources in a standard format, and is stored in an **OCM repository**. In our case, 
an OCI registry serves as OCM repository. 
The resources are divided into **local** and **external** resources. The local resources are stored together with the
component descriptor in the OCM repository. For example, we will store blueprints as local resources. 
External resources are stored elsewhere, for example a Docker image in a Docker registry.

This Guided Tour cannot go into all the details about that model, so you might want to read about the core concepts,
the benefits and the available tools on the official website under [https://ocm.software](https://ocm.software).
A good entry point is the [OCM Getting Started Guide](https://ocm.software/docs/guides/getting-started-with-ocm/).

The advantages of the OCM approach are:

- The blueprint is stored at a referencable location so that it can be reused by several Installations.
- The component descriptor contains a complete list of all resources involved in a deployment. Otherwise, you would have
  to search them spread somewhere in templates of charts and deploy items.
- The standardized format of component descriptors can be used by other tools to perform lifecycle management activities like 
  transport, signing/verification, scanning.



## The Component Version

To give an example, we build a component version for an echo server. It has the following resources:

- A blueprint, which you can find [here](https://github.com/gardener/landscaper/blob/master/docs/guided-tour/components/helm-chart/blueprint), 
  and which will be stored as local resource together with the component descriptor.
- A Helm chart, which you can find [here](https://github.com/gardener/landscaper/blob/master/docs/guided-tour/components/helm-chart/chart/echo-server). 
  We have uploaded it [in an OCI registry](https://eu.gcr.io/gardener-project/landscaper/examples/charts/guided-tour/echo-server).
  It will be an external resource of our component version.
- The Docker image [hashicorp/http-echo](https://hub.docker.com/r/hashicorp/http-echo). It will be another external resource.

### Uploading the Component Version

The most convenient way to generate a component version is through a **component constructor** file as described in
[Getting Started with OCM - All in One](https://ocm.software/docs/guides/getting-started-with-ocm/#all-in-one).
The component constructor file specifies the component name and version, and the list of resources.
The component constructor file for this example is [here](./commands/components.yaml):

```yaml
components:
  - name: github.com/gardener/landscaper-examples/guided-tour/helm-chart   # component name
    version: 1.0.0                                                         # component version
    provider:
      name: internal
    resources:
      - name: blueprint
        type: landscaper.gardener.cloud/blueprint
        input:
          type: dir
          path: ../blueprint
          compress: true
          mediaType: application/vnd.gardener.landscaper.blueprint.v1+tar+gzip
      - name: echo-server-chart
        type: helmChart
        version: 1.0.0
        access:
          type: ociArtifact
          imageReference: eu.gcr.io/gardener-project/landscaper/examples/charts/guided-tour/echo-server:1.0.0
      - name: echo-server-image
        type: ociImage
        version: v0.2.3
        access:
          type: ociArtifact
          imageReference: hashicorp/http-echo:0.2.3
```

The resource entries of the component constructor file either have an `input` or an `access` section.

- Resource entries with an `access` section will become external resources of the component version. 
  In our case, the chart and image are both OCI artifacts whose access information is an OCI image reference.
  There exist other access types, which are specified [here](https://ocm.software/docs/guides/input_and_access/#access-types).
  See for example section [Helm Chart in a Helm Chart Repository](#helm-chart-in-a-helm-chart-repository) below.

- Resource entries with an `input` section become local resources (normally, it depends on the type).
  The blueprint entry has an `input` section of type `dir`, whose `path` references the blueprint directory on the file system.
  During the creation of the component version, the content of the blueprint will be taken from there and uploaded 
  as local resource together with the component descriptor into the OCM repository.
  Again, there exist other input types, which are specified [here](https://ocm.software/docs/guides/input_and_access/#input-types).

The script [components.sh](./commands/component.sh) creates the actual component version based on the component constructor file.
The script uses commands of the [OCM Command Line Client](https://ocm.software/docs/cli/). It proceeds in two steps:
- First, the command [ocm add componentversions](https://ocm.software/docs/cli/add/componentversions/) 
  creates a file system representation of the component version, a so-called common transport file (CTF).
  The component constructor file determines what will be included in the CTF.
- Second, the command [ocm transfer ctf](https://ocm.software/docs/cli/transfer/commontransportarchive/) uploads the CTF
  to the OCM repository.

### Downloading the Component Version

We have uploaded the component version into this 
[location](https://console.cloud.google.com/gcr/images/gardener-project/eu/landscaper/examples/component-descriptors/github.com/gardener/landscaper-examples/guided-tour/helm-chart) 
in our public OCI registry. You can download the component version from there with the following command
[ocm download componentversions](https://ocm.software/docs/cli/download/componentversions/), which more or less provides 
you its file system representation:

```
ocm download componentversions eu.gcr.io/gardener-project/landscaper/examples//github.com/gardener/landscaper-examples/guided-tour/helm-chart:1.0.0 -O <your location on the file system>
```

[Here](https://github.com/gardener/landscaper/blob/master/docs/guided-tour/components/component) you can see the 
downloaded component version. The most important part is the [component descriptor](component/component-descriptor.yaml) 
which mainly contains the adapted data from the component constructor file. Besides the component descriptor, there is a 
directory `blobs` which contains the local resources of the component version. In our case there is only one blob, 
namely the blueprint as tar file.



## The Installation

Our Installation does not contain the blueprint inline as in previous examples. Rather it references the blueprint 
in our component version. This is done as follows:

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
spec:
  context: landscaper-examples
  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper-examples/guided-tour/components/helm-chart
      version: 1.0.0
  blueprint:
    ref:
      resourceName: blueprint
```

- The field `context` contains the name of a custom resource of kind [Context](../../../usage/Context.md). This Context
  must exist in the same namespace as the Installation on the Landscaper resource cluster. The Context contains the 
  information in which OCM repository the component descriptor is stored:

    ```yaml
    apiVersion: landscaper.gardener.cloud/v1alpha1
    kind: Context
    repositoryContext:
      type: ociRegistry
      baseUrl: eu.gcr.io/gardener-project/landscaper/examples
    ```

- The fields `componentDescriptor.ref.componentName` and `componentDescriptor.ref.version` are then used to locate the
  component descriptor in that OCM repository.

- The component descriptor contains a list of resources,
  each of which has a name. Field `blueprint.ref.resourceName` in the Installation specifies the name of the blueprint
  resource in the component descriptor.



## The Blueprint

You can find the blueprint for the current example 
[here](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/components/helm-chart/blueprint). 
The blueprint is a directory. It must contain a `blueprint.yaml` file. It may contain further files and subdirectories
like the `deploy-execution.yaml` file in our case.

### Referencing the Helm Chart

Our blueprint defines a Helm deploy item. In the template for the deploy item, the Helm chart must be specified. 
The preferred way is to reference the corresponding resource in the component version. 
The entry in the component descriptor looks as follows:

```yaml
resources:
  - name: echo-server-chart
    type: helmChart
    version: 1.0.0
    access:
      type: ociArtifact
      imageReference: eu.gcr.io/gardener-project/landscaper/examples/charts/guided-tour/echo-server:1.0.0
```

In the [deploy item template](./blueprint/deploy-execution.yaml), we reference this resource of the component version 
by its name `echo-server-chart`:
   
```yaml
chart:
  resourceRef: {{ getResourceKey `cd://resources/echo-server-chart` }}
```

This snippet shows a Go Templating function `getResourceKey` with a single input argument `cd://resources/echo-server-chart`.
The result of the Go Templating of this expression is the access data to the helm chart as specified in the component
descriptor.
As this function uses ocm to fetch the corresponding resource, you can even switch the storage technology
(more often referred to as access type in the context of ocm) - thus, e.g. store the helm chart in a helm repository 
instead of an oci registry - without having to adjust the blueprint (You will of course have to adjust the access in the
corresponding component version).

>**_NOTE:_**
> Theoretically, it is possible that the name of a *component reference* or a *resource* is not sufficient to uniquely
> identify them within a component version. The [ocm specification](https://github.com/open-component-model/ocm-spec/blob/main/doc/01-model/03-elements-sub.md#identifiers)
> defines that the identity of references as well as resources may optionally also contain a *version* and an
> *extraId entity* (see following [component configuration file](https://ocm.software/docs/guides/getting-started-with-ocm/#all-in-one)
> as an [example](./assets/components.yaml)).
>
> In such cases, the resource cannot be specified with a path expression. Instead, it has to be specified as defined in
> the [guidelines of the ocm specification](https://github.com/open-component-model/ocm-spec/blob/main/doc/05-guidelines/03-references.md).
> The corresponding part of the deploy execution looks like this:
>
> ```yaml
> chart:
>   resourceRef: {{ getResourceKey `
>     resource:
>       name: ocmcli
>       version: v0.5.0
>       extraIdentity:
>         architecture: amd64
>         os: linux
> `}}
> ```
>
> Here, you have to be careful with the indentation, as the input string has to be valid yaml. You could also specify
> this as JSON.
>
>```yaml
> chart:
>   resourceRef: {{ getResourceKey `{"resource":{"name":"ocmcli","version":"v0.5.0","extraIdentity":{"architecture":"amd64","os":"linux"}}}` }}
>```



### Referencing the Docker Image

The [echo-server helm chart](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/components/helm-chart/chart/echo-server) 
in this example consists of a Deployment and a Service. The Deployment uses a Docker image. 
Instead of a hard-coded image reference in the [deployment.yaml](./chart/echo-server/templates/deployment.yaml), 
we maintain the image reference in the component version. In detail, the connection is the following:

- The component descriptor contains a resource with name `echo-server-image` and a reference to the actual image:

  ```yaml
  resources:
    - name: echo-server-image
      type: ociImage
      version: v0.2.3
      access:
        type: ociArtifact
        imageReference: hashicorp/http-echo:0.2.3
  ```

- The blueprint contains a [template for a DeployItem](./blueprint/deploy-execution.yaml). Part of this is a
  section `values` for the Helm values. During the templating, we read the entry `echo-server-image` of the
  component descriptor, extract the field `access.imageReference`, and write it into the section with Helm values:

  ```yaml
  values:
    {{ $imageResource := getResource .cd "name" "echo-server-image" }}
    image: {{ $imageResource.access.imageReference }}
  ```

  After the templating, the resulting `DeployItem` contains the image reference in its `values` section:

  ```yaml
  values:
    image: hashicorp/http-echo:0.2.3
  ```

- Finally, the [deployment.yaml](./chart/echo-server/templates/deployment.yaml) template of the chart takes the image from the
  Helm values:

  ```yaml
  containers:
    - image: {{ .Values.image }}
  ```

> **_NOTE:_** Since Kubernetes does not support OCM, we need the *oci reference* of the container image, here.
> Consequently, to actually use this component version with landscaper, the container image that has to be deployed in
> a pod cannot be embedded into the component as a local blob (though it make sense to do so, as an intermediate step
> during transport of the component).



## Procedure

The procedure to install the echo server is as follows:

1. On the target cluster, create a namespace `example`. It is the namespace into which we will deploy the echo server.

2. On the Landscaper resource cluster, create a namespace `cu-example`. 

3. On the Landscaper resource cluster, in namespace `cu-example`, create a Target `my-cluster` containing a 
kubeconfig for the target cluster, a Context `landscaper-examples`, and an Installation `echo-server`. 
There are templates for these resources in the directory
[installation](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/components/helm-chart/installation).
To apply them:
   - adapt the [settings](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/components/helm-chart/commands/settings) file
     such that the entry `TARGET_CLUSTER_KUBECONFIG_PATH` points to the kubeconfig of the target cluster,
   - run the [deploy-k8s-resources script](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/components/helm-chart/commands/deploy-k8s-resources.sh), 
     which will template and apply the Target, Context, and Installation.

4. To try out the echo server, first define a port forwarding on the target cluster:

   ```shell
   kubectl port-forward -n example service/echo-server 8080:80
   ```

   Then open `localhost:8080` in a browser. The response should be "hello world", which is the text defined in the 
   [values.yaml](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/components/helm-chart/chart/echo-server/values.yaml)
   of the chart.



## Cleanup

You can remove the Installation with the 
[delete-installation script](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/components/helm-chart/commands/delete-installation.sh).
When the Installation is gone, you can delete the Context and Target with the
[delete-other-k8s-resources script](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/components/helm-chart/commands/delete-other-k8s-resources.sh).



## Helm Chart in a Helm Chart Repository

This section describes a modification of the above example.

Suppose the Helm chart is stored in a Helm chart repository instead of an OCI registry. In this case, you just need to 
modify the component constructor file before you upload the component version. In the entry of the Helm chart resource,
replace the `access` section using the access type `helm` as described in
[OCM Input and Access Types](https://ocm.software/docs/guides/input_and_access/#helm-1):

```yaml
resources:
  - name: echo-server-chart
    type: helmChart
    version: 1.0.0
    access:
      type: helm
      helmChart: echo-server:1.0.0
      helmRepository: https://example.helm.repo.com/landscaper
```



## References

[Context](../../../usage/Context.md)  
[Open Component Model (OCM)](https://ocm.software)  
[OCM: All in One](https://ocm.software/docs/guides/getting-started-with-ocm/#all-in-one)  
[OCM: Input and Access Types](https://ocm.software/docs/guides/input_and_access)  

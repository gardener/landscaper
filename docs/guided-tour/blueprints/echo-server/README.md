---
title: Deploying an Echo-Server as OCM Component
sidebar_position: 3
---

# Echo Server Example

In this example, we use the Landscaper to deploy an echo server.

For prerequisites, see [here](../../README.md).

The example uses the following resources:

- a blueprint, which you can find [here](https://github.com/gardener/landscaper/blob/master/docs/guided-tour/blueprints/echo-server/blueprint/blueprint.yaml), and 
  uploaded [in an OCI registry](https://eu.gcr.io/gardener-project/landscaper/examples/blueprints/guided-tour/echo-server),
- a Helm chart which you can also find [here](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/blueprints/echo-server/chart/echo-server) and
  uploaded [in an OCI registry](https://eu.gcr.io/gardener-project/landscaper/examples/charts/guided-tour/echo-server),
- the Docker image [hashicorp/http-echo](https://hub.docker.com/r/hashicorp/http-echo) as an external resource.

All of these resources are bundled in a component version. The component versions's configuration file is shown 
[here](./config-files/components.yaml).

Describing required resources in a standard format (for which we use the "Open Component Model") has several advantages.
This Guided Tour can not go into all the details about that model, so you might want to read about the core concepts, 
the benefits and the available tools on the official website under [https://ocm.software](https://ocm.software).

Without a consistent description for your component and its technical resources, you would have to search images spread 
somewhere in charts, perhaps even mixed with some templating. Moreover, such standardized components can be used by 
other tools to perform other lifecycle management activities like consistent transports into other environments or to do
signing/verification of software components.

## OCI Image Resource in the Component

The [echo-server helm chart](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/blueprints/echo-server/chart/echo-server) in this example consists of a `Deployment` and a `Service`.
The `Deployment` uses a container image. However, instead of a hard-coded image reference in
the [deployment.yaml](./chart/echo-server/templates/deployment.yaml), we rather maintain the image reference in the
component version. In detail, the connection is the following:

- The [component version](./component-archive/v2-external/component-descriptor.yaml) contains a resource with name 
  `echo-server-image` and a reference to the actual image:
 
  ```yaml
  resources:
  - name: echo-server-image
    type: ociImage
    version: v0.2.3
    access:
      type: ociArtifact
      imageReference: hashicorp/http-echo:0.2.3
  ```

- The [blueprint](./blueprint/blueprint.yaml) contains a template for a `DeployItem`. Part of this is a 
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

The procedure to install the helm chart with Landscaper is as follows:

1. Add the kubeconfig of your target cluster to your [target.yaml](installation/target.yaml).

2. On the Landscaper resource cluster, create namespace `example` and apply
   the [context.yaml](./installation/context.yaml),
   the [target.yaml](installation/target.yaml), and the [installation.yaml](installation/installation.yaml):

   ```shell
   kubectl create ns example
   kubectl apply -f <path to context.yaml>
   kubectl apply -f <path to target.yaml>
   kubectl apply -f <path to installation.yaml>
   ```

3. To try out the echo server, first define a port forwarding on the target cluster:

   ```shell
   kubectl port-forward -n example service/echo-server 8080:80
   ```

   Then open `localhost:8080` in a browser.  
   
   The response should be "hello world", which is the text defined
   in the [values.yaml](./chart/echo-server/values.yaml) of the chart.

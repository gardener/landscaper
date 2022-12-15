# Echo Server Example

In this example, we use the Landscaper to deploy an echo server.

For prerequisites, see [here](../../README.md#prerequisites-and-basic-definitions).

The example uses the following resources:

- a blueprint, which you can find [locally](./blueprint), and 
  uploaded [in an OCI registry](https://eu.gcr.io/gardener-project/landscaper/examples/blueprints/guided-tour/echo-server),
- a Helm chart which you can also find [locally](./blueprint) and
  uploaded [in an OCI registry](https://eu.gcr.io/gardener-project/landscaper/examples/charts/guided-tour/echo-server),
- the Docker image [hashicorp/http-echo](https://hub.docker.com/r/hashicorp/http-echo) as an external resource.

All these resources are listed in the [component descriptor](./component-descriptor.yaml). 
It is an advantage to have them all in one place, in a standard format. 
Otherwise, you would have to search images spread somewhere in charts, perhaps even mixed with some templating.
Moreover, the component descriptor can be used by other tools for example for transport or signing. 


## OCI Image Resource in the Component Descriptor

The [echo-server Helm chart](./chart/echo-server) in this example consists of a `Deployment` and a `Service`.
The `Deployment` uses a container image. However, instead of a hard-coded image reference in
the [deployment.yaml](./chart/echo-server/templates/deployment.yaml), we rather maintain the image reference in the
component descriptor. In detail, the connection is the following:

- The [component descriptor](./component-descriptor.yaml) contains a resource with name `echo-server-image` and a
  reference to the actual image:
 
  ```yaml
  name: echo-server-image
  type: ociImage
  version: v0.2.3
  relation: external
  access:
    type: ociRegistry
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

- Finally, the [deployment.yaml](./chart/echo-server/templates/deployment.yaml) of the chart takes the image from the 
  Helm values:

  ```yaml
  containers:
    - image: {{ .Values.image }}
  ```


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

# An Installation with an Externally Stored Blueprint

The Installation in this example does not contain its blueprint inline as in the previous examples, rather the
blueprint is stored separately together with a 
[component descriptor](../../../concepts/Glossary.md#_component-descriptor_). The component descriptor contains the 
list of all resources that are required for the deployment. In our example, there are two resources, namely the 
blueprint and the hello-world Helm chart.

We have uploaded the component descriptor and the blueprint together
[here](https://console.cloud.google.com/gcr/images/gardener-project/eu/landscaper/examples/component-descriptors/github.com/gardener/landscaper-examples/guided-tour/blueprints/simple?project=gardener-project)
into an OCI registry, from where the Landscaper can access them.


## How it is all connected

### Referencing the Blueprint in the Installation

The [Installation](./installation/installation.yaml) references the component and blueprint as follows:  

```yaml
componentDescriptor:
 ref:
   componentName: github.com/gardener/landscaper-examples/guided-tour/blueprints/simple
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


### Referencing the Helm Chart

The [deploy execution](./blueprint/deploy-execution.yaml) could have referenced the Helm chart directly like this:

```yaml
chart:
  ref: eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:1.0.0
```

However, it is recommended to collect all resources in the component descriptor. Therefore, in the resource list of
the [component descriptor](./component-descriptor.yaml), we created an entry with name `hello-world-chart` 
which contains the address of the Helm chart. 

The [deploy execution](./blueprint/deploy-execution.yaml) takes the Helm chart address from there.
Note that the deploy execution is a Go template. Landscaper provides a function `getResource` which is used here to get
the resource with name `hello-world-chart` from the component descriptor:

```yaml
chart:
  {{ $chartResource := getResource .cd "name" "hello-world-chart" }}
  ref: {{ $chartResource.access.imageReference }}
```


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

[Templating](../../../usage/Templating.md)
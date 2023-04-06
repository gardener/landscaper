# Templating: Accessing Component Descriptors 

For prerequisites, see [here](../../README.md#prerequisites-and-basic-definitions).

This example demonstrates the templating of DeployItems. In particular, we show how you can access component descriptors
during the templating.


## References Between Component Descriptors

Component descriptors can reference other component descriptors. In this example we consider three component descriptors, 
which we name as follows:
- the [root component descriptor](./component-root/component-descriptor.yaml),
- the [core component descriptor](./component-core/component-descriptor.yaml),
- the [extension component descriptor](./component-extension/component-descriptor.yaml).  

The root component descriptor references the other two in its section `component.componentReferences`:

```yaml
component:
  ...
  componentReferences:
    - componentName: github.com/gardener/landscaper-examples/guided-tour/templating-components-core
      name: core
      version: 1.0.0
    - componentName: github.com/gardener/landscaper-examples/guided-tour/templating-components-extension
      name: extension
      version: 1.0.0
```

We have uploaded these three component descriptors into an OCI registry, so that the Landscaper can read them from there
([root](https://eu.gcr.io/gardener-project/landscaper/examples/component-descriptors/github.com/gardener/landscaper-examples/guided-tour/templating-components-root), 
[core](https://eu.gcr.io/gardener-project/landscaper/examples/component-descriptors/github.com/gardener/landscaper-examples/guided-tour/templating-components-core), 
[extension](https://eu.gcr.io/gardener-project/landscaper/examples/component-descriptors/github.com/gardener/landscaper-examples/guided-tour/templating-components-extension)).


## The Blueprint

The [blueprint](./blueprint) of the present example belongs to the root component. Part of the blueprint is a 
[deploy execution](./blueprint/deploy-execution.yaml). The deploy execution is a [Go Template][2], 
which is used to generate a DeployItem. 
The template can be filled with values from a certain data structure. The following fields in this data structure 
provide access to the involved component descriptors:

- **cd** : the component descriptor of the Installation. In our case, this is the root component descriptor.  
  Let's consider an example, how to this field can be used. The expression below evaluates to the component name. 
  That is because field `cd` contains the complete 
  component descriptor, and inside it, the component name is located at the path `component.name`.
  ```yaml
  {{ .cd.component.name }}
  ```

- **components** : a list of component descriptors. It contains the component descriptor of the 
  Installation, and all further component descriptors which can be reached from this one by (transitively) following
  component references. In our case, the list contains the three component descriptors from above.
  To give an example, a list with the names of the involved components can be obtained as follows:
  ```yaml
  componentNames:
  {{ range $index, $comp := .components }}
    - {{ $comp.component.name }}  
  {{ end }}
  ```

Let's discuss the  [deploy execution](./blueprint/deploy-execution.yaml) of our blueprint.

- First, it loops over all components and collects all their resources in a list: `$resources`.  
- In a second step, it selects the resources with certain labels. Resources with label `landscaper.gardener.cloud/guided-tour/type`
are added to a dictionary `$typedResources`, and resources with label `landscaper.gardener.cloud/guided-tour/auxiliary` are added to
a dictionary `$auxiliaryResources`.  
- Finally, these "typed" and "auxiliary" resources are inserted at different places in a ConfigMap manifest, which will
be deployed by the manifest deployer.  

Note that you can use certain [sprig template functions][3] like `list`, `append`, `dict`, etc.

For more details, see [Templating][1].


## Procedure

The procedure to deploy the helm chart with the Landscaper is:

1. Insert the kubeconfig of your target cluster into file [target.yaml](installation/target.yaml).

2. On the Landscaper resource cluster, create namespace `example` and apply the [context.yaml](./installation/context.yaml), 
   the [target.yaml](installation/target.yaml), and the [installation.yaml](installation/installation.yaml):

   ```shell
   kubectl create ns example
   kubectl apply -f <path to context.yaml>
   kubectl apply -f <path to target.yaml>
   kubectl apply -f <path to installation.yaml>
   ```

## Cleanup

To clean up, delete the Installation from the Landscaper resource cluster:

```shell
kubectl delete inst -n example templating-components
```


## References 

[Templating][1]  
[Go Template][2]  
[Sprig template functions][3]

[1]: ../../../usage/Templating.md  
[2]: https://pkg.go.dev/text/template  
[3]: http://masterminds.github.io/sprig/
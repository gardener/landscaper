# Templating Example

For prerequisites, see [here](../../README.md#prerequisites-and-basic-definitions).

This example demonstrates the templating of DeployItems. In particular, we show how you can access component descriptors
during the templating.


## Blueprint and Component Descriptors

You can find the blueprint for the current example [here](./blueprint). 

We have uploaded the blueprint and the component descriptors into an OCI registry, so that the Landscaper can read them from there:
- [blueprint](https://eu.gcr.io/gardener-project/landscaper/examples/blueprints/guided-tour/external-blueprint)
- [component github.com/gardener/landscaper-examples/guided-tour/templating-components-root](https://eu.gcr.io/gardener-project/landscaper/examples/component-descriptors/github.com/gardener/landscaper-examples/guided-tour/templating-components-root)
- [component github.com/gardener/landscaper-examples/guided-tour/templating-components-core](https://eu.gcr.io/gardener-project/landscaper/examples/component-descriptors/github.com/gardener/landscaper-examples/guided-tour/templating-components-core)
- [component github.com/gardener/landscaper-examples/guided-tour/templating-components-extension](https://eu.gcr.io/gardener-project/landscaper/examples/component-descriptors/github.com/gardener/landscaper-examples/guided-tour/templating-components-extension)


## Template

The file [blueprint/deploy-execution.yaml](./blueprint/deploy-execution.yaml) contains a [Go Template][2] which is 
used to generate a DeployItem. 

The template can be filled with values from a certain data structure. The following fields in this data structure 
provide access to the involved component descriptors:

- **cd** : the component descriptor of the Installation.  
  For example, the expression below evaluates to the component name. That is because field `cd` contains the complete 
  component descriptor, and inside it, the component name is located at the path `component.name`.
  ```yaml
  {{ .cd.component.name }}
  ```

- **components** : the list of all referenced component descriptors. A component descriptor can reference others. For
  example the [root component descriptor](./component-root/component-descriptor.yaml) of this example 
  references two other component descriptors in its section `component.componentReferences`. 
  The field `components` contains the component descriptor of the Installation, and all further component descriptors
  which can be reached from this one by (transitively) following component references.  
  For example, a list with the names of the involved components can be obtained as follows:
  ```yaml
  componentNames:
  {{ range $index, $comp := .components }}
    - {{ $comp.component.name }}  
  {{ end }}
  ```


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

[1]: ../../../usage/Templating.md  
[2]: https://pkg.go.dev/text/template  
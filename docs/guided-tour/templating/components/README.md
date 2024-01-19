# Templating: Accessing Component Descriptors 

For prerequisites, see [here](../../README.md#prerequisites-and-basic-definitions).

This example demonstrates the templating of DeployItems. In particular, we show how you can access component descriptors
during the templating.


## References Between Component Descriptors

Component descriptors can reference other component descriptors. In this example we consider three component descriptors, 
which we name as follows:
- the [root component descriptor](./component-root/v2-external/component-descriptor.yaml),
- the [core component descriptor](./component-core/v2/component-descriptor.yaml),
- the [extension component descriptor](./component-extension/v2/component-descriptor.yaml).  

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

The [blueprint](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/templating/components/blueprint) of the present example belongs to the root component. Part of the blueprint is a 
[deploy execution](./blueprint/deploy-execution.yaml). The deploy execution is a [Go Template][2], 
which is used to generate a DeployItem. 
The template can be filled with values from a certain data structure. The following fields in this data structure 
provide access to the involved component descriptors:

> **_NOTE:_** If you are using Component
> Descriptors [Version 3](https://ocm.software/docs/component-descriptors/version-3/) instead of 
> [Version 2](https://ocm.software/docs/component-descriptors/version-2/), the data structure of the 
> component descriptors themselves is slightly different from what is described below (e.g. a component's name is under 
> `metadata.name` instead of `component.name`).  
> Per default, the component descriptor version a blueprint is templating against is the version of the component 
> descriptor referenced in the installation.  
> Since a blueprint could be used in different installations with different component descriptor versions, it is also
> possible to specify the component descriptor version (v2 or ocm.software/v3alpha1) to template against in the
> blueprint itself. So you may decide that you want to template against v2 even though v3alpha1 is the component 
> descriptor version provided in the installation (or vice versa).   Therefore, you may simply add the following 
> annotation to the blueprint:
> 
> ```yaml
> apiVersion: landscaper.gardener.cloud/v1alpha1
> kind: Blueprint
> jsonSchema: "https://json-schema.org/draft/2019-09/schema"
> annotations:
>   ocmSchemaVersion: v2 #or ocm.software/v3alpha1
> ...
> ```

- **cd** : the component descriptor of the Installation. In our case, this is the root component descriptor.  
  Let's consider an example, how this field can be used. The expression below evaluates to the component name. 
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
  {{ range $index, $comp := .components.components }}
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

The resources that we have collected from the component descriptors look for example like this:

```yaml
- access:
    imageReference: eu.gcr.io/gardener-project/landscaper/examples/images/image-a:1.0.0
    type: ociRegistry
  labels:
    - name: landscaper.gardener.cloud/guided-tour/type
      value: type-a
  name: image-a
  relation: external
  type: ociImage
  version: 1.0.0
```

This is not yet the desired result format. Therefore, we use a template `formateResource` to transform the resources. 
The template extracts the field `.access.imageReference` from a resource, splits the string value in 
three parts, and produces the following result: 

```yaml
registry: eu.gcr.io
repository: gardener-project/landscaper/examples/images/image-a
tag: 1.0.0
```

We can pass only one argument to a template. However, our template `formateResource` needs two inputs, a `resource` and
an `indent`. To solve this, we put both values in a dictionary `$args` and pass this dictionary to template:

```yaml
{{- $args := dict "resource" $resource.access.imageReference "indent" 20 }}
{{- template "formatResource" $args }}
```

Note that you can use certain [sprig template functions][3] like `list`, `append`, `dict` etc.

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

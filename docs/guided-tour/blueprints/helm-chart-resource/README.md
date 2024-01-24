# Helm Chart Resources in the Component Descriptor

For prerequisites see [here](../../README.md#prerequisites-and-basic-definitions).

The blueprint of the previous examples reference the Helm chart directly like this:

```yaml
chart:
  ref: eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:1.0.0
```

However, it is recommended to collect all resources of an application in its component version and reference these
resources in the blueprint only by their name in the component descriptor. This has the
advantages that you have exactly one location, listing all resources required to deploy a component and if you just
want to update the version of a resource, you only need to update its component version. No change of
the blueprint is required.

Therefore, we will now modify the previous example as follows:

- We extend the resources contained in our component version by a Helm chart.
- We also modify the blueprint such that it takes the address of the Helm chart from the new entry of the component 
  descriptor.


## The Component Version

You can find the file system representations of the extended component versions [here](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/blueprints/helm-chart-resource/component-archive/v2-external), and in the OCI registry [here](https://eu.gcr.io/gardener-project/landscaper/examples/component-descriptors/github.com/gardener/landscaper-examples/guided-tour/helm-chart-resource).

The list of resources of the component descriptor within the component version has two entries now, one for the 
blueprint and one for the Helm chart. The entry for the Helm chart has the name `hello-world-chart` and contains the 
address in the OCI registry.

## Referencing the Helm Chart
The deploy execution in the [blueprint](./blueprint/blueprint.yaml) can be modified so that it references a specific
resource from the component with a *resource key*.
> **_NOTE:_** For now, this resource key is merely a base64 encoded global resource identity (= component name, 
> component version and the resource identity, which consists at least of the resource name). This information
> might be useful for debugging purposes.  
> **But since this will likely change in the future, for all intends and purposes BUT debugging, you should view the 
> *resource key* as an opaque key!**

#### Go Templating with Path Expression
The `getResourceKey` function can parse a component descriptor path expression of the following form:
`cd://<keyword>/<value>/<keyword>/<value>/...`  
with the **keywords** `componentReferences` and `resources`.

In the most simple case, where the [component](./component-archive/v2-external/component-descriptor.yaml) referenced in 
the [installation](./installation/installation.yaml) directly contains the helm chart resource itself (as it is the case
here), this results in the following path:

```yaml
chart:
  resourceRef: {{ getResourceKey `cd://resources/hello-world-chart` }}
```

As this function uses [ocm](https://ocm.software/) to fetch the corresponding resource, you can even switch the
storage technology (more often referred to as access type in the context of ocm) - thus, e.g. store the helm chart in a
helm repository instead of an oci registry - without having to adjust the blueprint (You will of course have to adjust
the access in the corresponding component version). It even allows you to store your helm chart as a local blob as part
of the component!

> **_DEPRECATED_:** The following explains how this use case has typically been covered before the `getResourceKey()` 
> templating function has been introduced.

The deploy execution in the [blueprint](./blueprint/blueprint.yaml) has been modified so that it takes the Helm chart 
address from the component descriptor:

```yaml
chart:
  {{ $chartResource := getResource .cd "name" "hello-world-chart" }}
  ref: {{ $chartResource.access.imageReference }}
```

Note that the deploy execution is a Go template. Landscaper provides a function `getResource` which is used here to get
the resource entry with name `hello-world-chart` from the component descriptor. The variable `chartResource` has then
the resource entry as value, and the expression `ref: {{ $chartResource.access.imageReference }}` evaluates to the same
value as before.


## Procedure

The procedure to install the helm chart with Landscaper is the same as before:

1. Insert in file [target.yaml](installation/target.yaml) the kubeconfig of your target cluster.

2. On the Landscaper resource cluster, create namespace `example` and apply
   the [context.yaml](./installation/context.yaml),
   the [target.yaml](installation/target.yaml), and the [installation.yaml](installation/installation.yaml):

   ```shell
   kubectl create ns example
   kubectl apply -f <path to context.yaml>
   kubectl apply -f <path to target.yaml>
   kubectl apply -f <path to installation.yaml>
   ```

## Remark on Referencing Helm Charts
Theoretically, it is possible that the name of a *component reference* or a *resource* is not sufficient to uniquely
identify them within a component. The [ocm specification](https://github.com/open-component-model/ocm-spec/blob/main/doc/01-model/03-elements-sub.md#identifiers)  
defines that the identity of references as well as resources may optionally also contain a *version* and an 
*extraIdentity* (see following [component configuration file](https://ocm.software/docs/guides/getting-started-with-ocm/#all-in-one) 
as an [example](./assets/components.yaml)).

In such cases, the resource cannot be specified with a path expression. Instead, it has to be specified as defined in 
the [guidelines of the ocm specification](https://github.com/open-component-model/ocm-spec/blob/main/doc/05-guidelines/03-references.md).
The corresponding part of the deploy execution looks like this:

```yaml
chart:
  resourceRef: {{ getResourceKey `
    resource:
      name: ocmcli
      version: v0.5.0
      extraIdentity:
        architecture: amd64
        os: linux
`}}
```

> **_NOTE:_** Here, you have to be careful with the indentation, as the input string has to be valid yaml.

## Remark on Charts Stored in a Helm Chart Repository
> **_DEPRECATED_:** This use case can be covered with the `getResourceKey()` templating function in a more convenient 
> way.

Let us mention a variation of this example. Above, we have stored our Helm chart in an OCI registry as described in
[Use OCI-based registries](https://helm.sh/docs/topics/registries/).
Of course, you can also store your Helm chart in a Helm chart repository. In that case, the entry in the component 
descriptor would have the following format:

```yaml
  resources:
    - name: example-chart
      type: helm.io/chart
      version: 1.0.0
      relation: external
      access:
        type: helmChartRepository
        mediaType: application/octet-stream
        helmChartRepoUrl: <helm chart repo url>
        helmChartName:    <helm chart name>
        helmChartVersion: <helm chart version>
```

The `chart` section in the deploy execution of the blueprint, would then look like this: 

```yaml
  chart:
    helmChartRepo:
      {{- $chartResource := getResource .cd "name" "example-chart" }}
      helmChartRepoUrl: {{ $chartResource.access.helmChartRepoUrl }}
      helmChartName:    {{ $chartResource.access.helmChartName }}
      helmChartVersion: {{ $chartResource.access.helmChartVersion }}
```

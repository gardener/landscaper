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

> **_NOTE:_** Adding resources other than the blueprints as local blobs to the component version is not
> supported by the landscaper, yet.  
> This is because the deployers currently fetch the image based on the
> access information templated into the deploy item (as shown below) without knowledge of a corresponding
> component version. This information is not sufficient to resolve local blobs.


## Referencing the Helm Chart

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


## Remark on Charts Stored in a Helm Chart Repository

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

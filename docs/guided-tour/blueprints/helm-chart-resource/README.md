# Helm Chart Resources in the Component Descriptor

For prerequisites see [here](../README.md#prerequisites-and-basic-definitions).

The blueprint of the previous examples reference the Helm chart directly like this:

```yaml
chart:
  ref: eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:1.0.0
```

However, it is recommended to collect all resources of a component in its component descriptor. 
Therefore, we will now modify the previous example as follows:

- We extend the resource list of the component descriptor by an entry for the Helm chart.
- We also modify the blueprint such that it takes the address of the Helm chart from the new entry of the component 
  descriptor.


## The Component Descriptor

You can find the extended component descriptor [here](./component-descriptor.yaml), and in the OCI registry
[here](https://eu.gcr.io/gardener-project/landscaper/examples/component-descriptors/github.com/gardener/landscaper-examples/guided-tour/helm-chart-resource).

Its list of resources has now two entries, one for the blueprint and one for the Helm chart.
The entry for the Helm chart has the name `hello-world-chart` and contains the address in the OCI registry.


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
resource entry as value, and the expression `ref: {{ $chartResource.access.imageReference }}` evaluates to the same
value as before.


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

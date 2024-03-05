---
title: Helm Chart Component
sidebar_position: 2
---

# Helm Chart Resources in the Component Version

Let's look at an example in which a Helm Chart is referenced from a component version.

For prerequisites see [here](../../README.md).

## Referencing the Helm Chart without a Component Version

The blueprint of the previous examples reference the Helm chart directly like this:

```yaml
chart:
  ref: eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:1.0.0
```

Using this oci reference, the landscaper is able to fetch the helm chart and deploy it correspondingly. This approach 
has the disadvantage, that the location of the chart is hidden in the blueprint which is an unknown resource type
of OCM. Therefore, if OCM component versions are used for transport, scanning etc. such hidden artefact could not be 
identified as part of the component version and are just skipped. A better approach is to specify such artefacts in 
the component version itself such that they are known on the component level and just reference the artefacts in 
the blueprint via their entries in the component descriptor of the component version. We will show this in the 
following with the helm chart artefact.

## Referencing the Helm Chart with a Component

In the [previous section](../external-blueprint/README.md), the concept of component versions was introduced, as an alternative
means to reference the blueprints in the installation instead of having to write the blueprints directly inline into the
installation (as it was done in the [first several examples](../../hello-world/installation/installation.yaml)). 

The same technique can be used to reference helm charts from within a blueprint, as well. To do
this, we have to:
- Extend the resources contained in our [component version](../external-blueprint/config-files/components.yaml) by a *helm 
chart*. 
- Modify the [blueprint](../external-blueprint/blueprint/blueprint.yaml) so that it references the *helm chart* resource 
  in the component version instead of directly referencing the oci image location.

The *component configuration file* for the component version with the extended resources is shown 
[here](./config-files/components.yaml):

```yaml
components:
  - name: github.com/gardener/landscaper-examples/guided-tour/helm-chart-resource
    version: 2.0.0
    provider:
      name: internal
    resources:
      - name: blueprint
        type: landscaper.gardener.cloud/blueprint
        version: 1.0.0
        access:
          type: ociArtifact
          # notice that this has to be a reference to the updated blueprint
          imageReference: eu.gcr.io/gardener-project/landscaper/examples/blueprints/guided-tour/helm-chart-resource:1.0.0
      - name: helm-chart
        type: helmChart
        version: 1.0.0
        access:
          type: ociArtifact
          imageReference: eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:1.0.0
```

If your helm chart is stored in a *helm chart repository* instead of an oci registry, the *component configuration file* 
for the component with the extended resources would look like this:

```yaml
components:
  - name: github.com/gardener/landscaper-examples/guided-tour/helm-chart-resource
    version: 2.0.0
    provider:
      name: internal
    resources:
      - name: blueprint
        type: landscaper.gardener.cloud/blueprint
        version: 1.0.0
        access:
          type: ociArtifact
          # notice that this has to be a reference to the updated blueprint
          imageReference: eu.gcr.io/gardener-project/landscaper/examples/blueprints/guided-tour/helm-chart-resource:1.0.0
      - name: helm-chart
        type: helmChart
        version: 1.0.0
        access:
          type: helm
          helmChart: hello-world:1.0.0
          helmRepository: https://example.helm.repo.com/landscaper
```

Again, if you prefer to embed the blueprint and the helm chart in the component as a local blob, instead of an 
`access:...`, you would have to specify an `input:...` as demonstrated [here](./config-files/local-blob-components.yaml). 
The commands used to create the actual component based on the *component configuration file* and to upload this
component to an OCI registry can be found [here](./commands/component.sh).

The updated blueprint is shown [here](./blueprint/blueprint.yaml), with the updated chart reference looking like this:

```yaml
chart:
  resourceRef: {{ getResourceKey `cd://resources/helm-chart` }}
```

This snippet shows a Go Templating function `getResourceKey` with a single input argument `cd://resources/helm-chart`.
The input argument specifies the resource with the name `helm-chart` of the component descriptor of the component version. 
The result of the Go Templating of this expression is the access data to the helm chart as specified in the component 
descriptor.

As this function uses [ocm](https://ocm.software/) to fetch the corresponding resource, you can even switch the
storage technology (more often referred to as access type in the context of ocm) - thus, e.g. store the helm chart in a
helm repository instead of an oci registry - without having to adjust the blueprint (You will of course have to adjust
the access in the corresponding component version).

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


# Installation and configuration of Landscaper

This document describes the installation of Landscaper.

Landscaper is a Kubernetes controller that reconciles installations of _Components_ and handles the interaction with _Deployers_ through _DeployItems_.

## Installation

Landscaper can easily be installed via [Helm](https://helm.sh/) using the Helm chart in [charts/landscaper](charts/landscaper).

```
helm install -n ls-system landscaper ./charts/landscaper
```

We are also building OCI charts so a specific version of Landscaper can be installed with:

```
export HELM_EXPERIMENTAL_OCI=1
export LS_VERSION="0.1.0"
helm chart pull eu.gcr.io/gardener-project/landscaper/charts/landscaper-controller:$VERSION
helm chart export eu.gcr.io/gardener-project/landscaper/charts/landscaper-controller:$VERSION ./charts
helm install ./charts
```

Landscaper's Helm chart can be configured with a values file.

In case of an OCI registry that is not exposed via https, the `allowPlainHttpRegistries` flag can be used.

The landscaper does offload all deployment specific functionality like `helm` to deployers.
For a very simple setup, internal deployers (`helm`, `manifest` and `container`) can be served by the landscaper.

## Configuration through `values.yaml`

The following snippet shows a sample `values.yaml` file that is used to parameterize the Helm chart:

```yaml
image:
  tag: image version # .e.g. 0.0.0-dev-8bf4b8150f96fed8868618c56787b81fa4e095e6

landscaper:
  registryConfig:
    allowPlainHttpRegistries: false
    secrets: # contains optional oci secrets
      default: {
        "auths": {
          "hostname": {
            "auth": "my auth"
          }
        }
      }
  
  # deploy with integrated deployers for quick start
  deployers: 
  - container
  - helm
  - mock
  - manifest 
```

### Landscaper image and tag

If Landscaper is installed with a local copy of the Helm chart, the field `image.tag` has to be defined to specifiy which container image for Landscaper should be used.

If Landscaper is installed through an OCI chart, the image tag will automatically be matched with the given version.

### Private registry access

Blueprints and component descriptors must reside in one or more OCI registries. During an installation process, Landscaper attempts to pull them from there. 

If component descriptors or blueprints are stored in a non-public OCI registry, the registry's secrets can be provided in the `landscaper.registrySecrets` section by providing a map of `<config-name>: <docker auth>`. Different sets of secrets can be provided for blueprints and component descriptors by the `landscaper.registrySecrets.blueprint` and `landscaper.registrySecrets.components` fields respectively.

The value to provide to `<docker auth>` must be a Docker auth config as plain JSON (not as a string). Refer to Kubernetes' [pull-secret documentation](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#log-in-to-docker) for a comprehensive guide.

### Internal and external deployers

Landscaper offloads all deployment specific logic (e.g. `helm`) to external deployers that are deployed to a target cluster.

For a very simple setup, internal deployers (`helm`, `manifest` and `container`) can be served by Landscaper.

:warning: Using internal deployers is meant for development and debugging and **should not be used in production**.

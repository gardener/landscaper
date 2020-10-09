# Install and Configure the Landscaper

This document describes the installation of the landscaper.
The landscaper is a kubernetes controller that reconciles installations and handles the interaction with the deployers through deploy items.

## Installation

The Landscaper can be easily installed via helm using the helm chart in [charts/landscaper](charts/landscaper).

```
helm install -n ls-system landscaper ./charts/landscaper
```

We also build oci charts so the landscaper can also be installed with a specific version with:
```
export HELM_EXPERIMENTAL_OCI=1
export LS_VERSION="0.1.0"
helm chart pull eu.gcr.io/gardener-project/landscaper/charts/landscaper-controller:$VERSION
helm chart export eu.gcr.io/gardener-project/landscaper/charts/landscaper-controller:$VERSION ./charts
helm install ./charts
```

The chart can be configured via the following values file.

The image tag will be automatically matched with the given version, if the oci helm chart is used.
If the first installation mechancism is used, the `image.tag` has to be defined.

If component descriptors or blueprints are stored in a non public oci registry, 
the oci secrets can be provided using a map of `<config-name>: <docker auth>`.
The docker auth config should be a docker auth config. 
See the kubernetes pull secret documentation for a comprehensive guide https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#log-in-to-docker.

The landscaper does offload all deployment specific functionality like `helm` to deployers.
For a very simple setup, internal deployers (`helm`, `manifest` and `container`) can be served by the landscaper.

:warning: this should not be used in production.

```yaml
image:
  tag: image version # .e.g. 0.0.0-dev-8bf4b8150f96fed8868618c56787b81fa4e095e6

landscaper:
  registrySecrets: # contains optional oci secrets
    blueprints:
      default: {
        "auths": {
          "hostname": {
            "auth": "my auth"
          }
        }
      }
    components:
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
```

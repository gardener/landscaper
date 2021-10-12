# Installation and configuration of Landscaper

This document describes the installation of Landscaper.

Landscaper is a Kubernetes controller that reconciles installations of _Components_ and handles the interaction with _Deployers_ through _DeployItems_.

## Installation

Landscaper can easily be installed via [Helm](https://helm.sh/) using the Helm chart in [charts/landscaper](../../charts/landscaper).

:warning: Attention: There were some major changes to `helm`'s OCI features with version `v3.7.0`. This guide uses the new version. If you want to use a version of `helm` older than `v3.7.0`, make sure you have `export HELM_EXPERIMENTAL_OCI=1` set and use `helm chart push`, `helm chart pull`, and `helm chart save` instead of `helm push`, `helm pull`, and `helm package` respectively.

```
kubectl create namespace ls-system
helm install -n ls-system landscaper ./charts/landscaper
```

We are also building OCI charts so a specific version of Landscaper can be installed with:

```
export LS_VERSION="v0.13.0" # use the latest available version
helm pull eu.gcr.io/gardener-project/landscaper/charts/landscaper-controller:$LS_VERSION
helm chart export eu.gcr.io/gardener-project/landscaper/charts/landscaper-controller:$LS_VERSION ./charts
helm install ./charts
```

Landscaper's Helm chart can be configured with a values file.

In case of an OCI registry that is not exposed via https, the `allowPlainHttpRegistries` flag can be used.

> Note: Landscaper offloads all deployment specific functionality like deploying Helm charts to deployers.
> By default, the Landscaper deployment contains no deployer so you are unable to reconcile any deploy items. 
> But a subset of internal open-source deployers (`helm`, `manifest` and `container`) can be automatically configured. See [below](#internal-and-external-deployers) for more details.

## Configuration through `values.yaml`

The following snippet shows a sample `values.yaml` file that is used to parameterize the Helm chart:

```yaml
landscaper:
    controller:
      image:
        tag: image version # .e.g. 0.0.0-dev-8bf4b8150f96fed8868618c56787b81fa4e095e6
    
    webhookServer:
    #  disableWebhooks: all # disables specific webhooks. If all are disabled the webhook server is not deployed
      image:
        tag: image version # .e.g. 0.0.0-dev-8bf4b8150f96fed8868618c56787b81fa4e095e6
    
    landscaper:
      cache: {} # Landscaper caches pulled OCI artefacts on disk and optionally in-memory
    #     useInMemoryOverly: false
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
      metrics:
        port: 8080  
    
      # deploy with integrated deployers for quick start
      deployers: 
      - container
      - helm
      - mock
      - manifest 
```

### Landscaper image and tag

If Landscaper is installed with a local copy of the Helm chart, the field `image.tag` has to be defined to specify which container image for Landscaper should be used.

If Landscaper is installed through an OCI chart, the image tag will automatically be matched with the given version.

### Private registry access

Blueprints and component descriptors must reside in one or more OCI registries. During an installation process, Landscaper attempts to pull them from there. 

If component descriptors or blueprints are stored in a non-public OCI registry, the registry's secrets can be provided in the `landscaper.landscaper.registrySecrets` section by providing a map of `<config-name>: <docker auth>`. Different sets of secrets can be provided for blueprints and component descriptors by the `landscaper.landscaper.registrySecrets.blueprint` and `landscaper.landscaper.registrySecrets.components` fields respectively.

The value to provide to `<docker auth>` must be a Docker auth config as plain JSON (not as a string). Refer to Kubernetes' [pull-secret documentation](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#log-in-to-docker) for a comprehensive guide.

### Caching
Landscaper allocates some temporary disk space to cache OCI artefact it pulls. Optionally, artefacts can be cached in-memory as well.

### Metrics
Landscaper is instrumented to collect the default metrics of the controller-runtimes. Additionally, it serves some custom metrics e.g. for its OCI cache. The metrics may be scraped at `/metrics` and a configurable port defaulting to `8080`.

### Internal and external deployers

Landscaper offloads all deployment specific logic (e.g. `helm`) to external deployers that are deployed to a target cluster.

For a very simple setup, internal deployers (`helm`, `manifest` and `container`) can be served by Landscaper.

The default deployers are deployed using the Landscaper integrated [Deployer Lifecycle Management](../technical/deployer_lifecycle_management.md) that are configured with the opensource images and basic defaults.

The default configuration can be overwritten or enhanced by supplying `deployerConfig` in the values.yaml.
See the specific [helm chart values](../../charts) of the deployer for detailed documentation.
```yaml
landscaper:
    landscaper:
      deployers: [container, helm]
      deployersConfig:
        # match the deployer name
        container: 
          # provide any helm charts values.
          deployer:
            oci:
              allowPlainHttp: false
        helm:
          # ...
```

Additional external deployer can be either configured by applying the `DeployerRegistration` directly or providing the Registration in the helm chart.

> Note: When the registration is provided through the helm chart, all defaults of the "default" environment are automatically injected (if not overwritten in the registration).

For detailed information about the DeployerRegistration and its configurations see the [documentation](../technical/deployer_lifecycle_management.md) or the [example](../../examples/80-Example-DeployerRegistration.yaml).

```yaml
landscaper:
    landscaper:
      deployers: [my-custom-deployer] # the external deployer name MUST be set.
      deployersConfig:
        # match the deployer name
        my-custom-deployer: 
          apiVersion: landscaper.gardener.cloud/v1alpha1
          kind: DeployerRegistration
          metadata:
            name: my-deployer # defaulted to "my-custom-deployer"
            
          spec:
            # describe the deploy items types the deployer is able to reconcile
            types: ["my-deploy-item-type"]
            installationTemplate: # note that no exports are allowed here.
              componentDescriptor:
                ref:
                  repositoryContext:
                    type: ociRegistry
                    baseUrl: "example.myregistry.com/my-context"
                  componentName: "my-custom-deployer"
                  version: v1.0.0
    
              blueprint:
                ref:
                  resourceName: my-deployer-blueprint
          ...
```


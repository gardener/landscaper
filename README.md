# Landscaper
**Work in progress... partial and unfinished documentation ahead!**

The _Landscaper_ is an [operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) to install, update & manage a Gardener deployment.
Its model consists of the entities [described here](docs/Entities.md)

For a list of available deployers is maintained [here](docs/deployer)

### Quick Start

The Landscaper is a kubernetes controller that can be easily installed via helm using the helm chart in [charts/landscaper](charts/landscaper).

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

The chart can be configured via the values file:
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
```

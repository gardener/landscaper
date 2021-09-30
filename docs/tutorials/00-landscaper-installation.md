# Install the Landscaper

This tutorial describes how to install the Landscaper on a Kubernetes cluster and optionally how to install an OCI registry
on a Gardener Shoot cluster.

### Prerequisites

- Landscaper CLI (see the [installation documentation](https://github.com/gardener/landscapercli/blob/master/docs/installation.md))
- This tutorial assumes that the landscaper will be installed in the namespace `ls-system`. The namespace has to be created before the landscaper is being installed.

### Install the Landscaper

This tutorial uses the `landscaper-cli` `quickstart` command. For additional details please see the [documentation](https://github.com/gardener/landscapercli/blob/master/docs/commands/quickstart/install.md).

#### Alternative 1: Install with OCI registry

The `landscaper-cli` can install the Landscaper together with an OCI registry. The OCI registry will be exposed by an ingress.

##### Prerequisites

- The target cluster must be a Gardener Shoot (TLS certificate is provided via the Gardener cert manager). 
- A nginx ingress controller must be deployed in the target cluster
- The command "htpasswd" must be installed on your local machine.

First the helm values for the Landscaper configuration (`landscaper-values.yaml`) have to be created:

```yaml
landscaper:
    landscaper:
      registryConfig:
        cache: {}
        allowPlainHttpRegistries: false
        insecureSkipVerify: false
      
      deployers: 
      - container
      - helm
      - manifest

      deployerManagement:
        disable: false
        namespace: ls-system
        agent:
          disable: false
          namespace: ls-system
```

This tutorial assumes that the OCI registry usernames is stored in environment variable `REGISTRY_USER` and the password is stored in environment variable `REGISTRY_PASSWORD`.

To install the Landscaper and the OCI registry, use the following command:

```shell script
landscaper-cli quickstart install \
    --kubeconfig ./kubeconfig.yaml \
    --landscaper-values ./landscaper-values.yaml \
    --namespace ls-system \
    --install-oci-registry \
    --install-registry-ingress \
    --registry-username ${REGISTRY_USER} \
    --registry-password ${REGISTRY_PASSWORD}
```

The Landscaper will automatically be configured with the provided OCI registry credentials.
After a successful installation the `landscaper-cli` will print the URL of the OCI registry endpoint:

```
Landscaper installation succeeded!

The OCI registry can be accessed via the URL https://o.ingress.cluster-domain
It might take some minutes until the TLS certificate is created
```

To test the OCI registry is working, issue the following command:

```shell script
curl --location --request GET https://o.ingress.cluster-domain/v2/_catalog -u "${REGISTRY_USER}:${REGISTRY_PASSWORD}
```

This should give the output:

```json
{"repositories":[]}
```

#### Alternative 2: Install using an external OCI registry

##### Prerequisites

- Kubernetes cluster
- An OCI registry that implements the [OCI Distribution Specification](https://github.com/opencontainers/distribution-spec).

First the helm values for the Landscaper configuration (`landscaper-values.yaml`) have to be created:

```yaml
landscaper:
    landscaper:
      registryConfig:
        cache: {}
        allowPlainHttpRegistries: false
        insecureSkipVerify: false
        secrets:
          default: {
            "auths": {
              "hostname": {
                "auth": "my auth"
              }
            }
          }
      
      deployers: 
      - container
      - helm
      - manifest

      deployerManagement:
        disable: false
        namespace: ls-system
        agent:
          disable: false
          namespace: ls-system
```

The registry secrets have to be provided in a plain Docker auth JSON format at `landscaper.landscaper.registryConfig.secrets`. See [here](../gettingstarted/install-landscaper-controller.md#Private registry access) for more details.

To install the Landscaper with the credentials for the OCI container registry, use the following command:

```shell script
landscaper-cli quickstart install \
    --kubeconfig ./kubeconfig.yaml \
    --landscaper-values ./landscaper-values.yaml \
    --namespace ls-system
```

### Working with the landscaper-cli
In order for the `landscaper-cli` to work with the registry, it needs valid credentials. The easiest way to generate these, would be via `docker login`.
```shell
docker login -u my-user my-oci-registry-url
```
# Deploy and Use the Landscaper locally

This tutorial describes how a landscaper can be deployed locally.
The only prerequisite is a kubernetes cluster 
(find [below](#kubernetes-clusters) a list of possible local kubernetes installers but basically every k8s should work).

**Index**:
- [Install a container registry](#install-a-container-registry)
- [Configure the Landscaper Controller](#configure-the-landscaper)
- [Common Pitfalls](#common-pitfalls)
- [Kubernetes Distributions](#kubernetes-clusters)

### Install a container registry
The Landscaper depends on an oci compliant registry to store and fetch Blueprints, Component Descriptors and other artifacts.
[Harbor](https://github.com/goharbor/harbor-helm) can be used as such a registry that can also run beside the Landscaper in a kubernetes cluster.

Configure a minimal harbor installation with a `values.yaml` file.
The most important components are `core` as well as the `registry`.<br>
See additional configuration options [here](https://github.com/goharbor/harbor-helm).
```
# values.yaml
registry:
  credentials:
    username: "harbor_registry_user"
    password: "harbor_registry_password"
    # If you update the username or password of registry, make sure use cli tool htpasswd to generate the bcrypt hash
    # e.g. "htpasswd -nbBC10 $username $password"
    htpasswd: "harbor_registry_user:$2y$10$9L4Tc0DJbFFMB6RdSCunrOpTHdwhid4ktBJmLD00bYgqkkGOvll3m"
chartmuseum:
  enabled: false
clair:
  enabled: false
trivy:
  enabled: false
notary:
  enabled: false
```

```
helm repo add harbor https://helm.goharbor.io
helm upgrade --install -n harbor registry harbor/harbor -f values.yaml
```

In the example above, helm installs the registry in the `harbor` namespace.
The registry can be used by the landscaper by referencing artifacts via it's internal service `registry-harbor-registry.harbor:5000`

The harbor is now only accessible from within the cluster, so the k8s proxy is needed to upload artifacts:
```shell script
kubectl -n harbor port-forward svc/registry-harbor-registry 5000:5000

# upload helm chart oci artifact

# add open source helm registry
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo add stable https://kubernetes-charts.storage.googleapis.com/
helm repo update

# download the helm artifact locally
helm pull ingress-nginx/ingress-nginx --untar --destination /tmp

# upload the oci artifact to a oci registry
export HELM_EXPERIMENTAL_OCI=1
helm registry login -u harbor_registry_user localhost:5000 # use the username and pw configured in the values.yaml
helm chart save /tmp/ingress-nginx localhost:5000/charts/ingress-nginx:0.0.1
helm chart push localhost:5000/charts/ingress-nginx:0.0.1

# chart ref inside the cluster: registry-harbor-registry.harbor:5000/charts/ingress-nginx:0.0.1
```

__Expose the Harbor__:
The harbor can also be exposed so that no `port-forward` proxy is needed to upload artifacts.
:warning: Keep in mind that the kubernetes cluster must support egress traffic and a ingress controller has to be installed. The ingress resource also targets harbor's portal /UI, not the registry directly.
```yaml
expose:
  type: ingress
  ingress:
    hosts:
      core: "my-host" # you need to set the dns entry on your own
  tls:
    secretName: "" # if you want valid certificates

externalURL: "https://my-host"
```

Upload:
```shell script
# add open source helm registry
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo add stable https://kubernetes-charts.storage.googleapis.com/
helm repo update

# download the helm artifact locally
helm pull ingress-nginx/ingress-nginx --untar --destination /tmp

# upload the oci artifact to a oci registry
export HELM_EXPERIMENTAL_OCI=1
helm registry login -u harbor_registry_user https://my-host # take harborAdminPassword from values.yaml, if no other user is specified
helm chart save /tmp/ingress-nginx https://my-host/charts/ingress-nginx:0.0.1
helm chart push https://my-host/charts/ingress-nginx:0.0.1

# chart ref inside the cluster: https://my-host/charts/ingress-nginx:0.0.1
```

### Configure the landscaper

The harbor installation inside cluster is password protected, so the landscaper needs to be configured with the username and password.

The Landscaper uses the default docker auth format (`docker.config`) to authenticate against the oci registry.
See more details in the kubernetes docs for [pull-secrets](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/).

```shell script
# Generate the auth token
$ AUTH_TOKEN=$(echo "<username>:<password>" | base64)
```

```json
// docker.json
{
    "auths": {
        "http://registry-harbor-registry.harbor:5000/": {
            "auth": "${AUTH_TOKEN}"
        }
    }
}
```

The landscaper can be configured to use these credentials by using the landscaper configuration and pass it via `--config` flag.
```yaml
apiVersion: config.landscaper.gardener.cloud/v1alpha1
kind: LandscaperConfiguration

registries:
  components:
    oci:
      configFiles:
      - "/path/to/docker.json"
  blueprints:
    oci:
      configFiles:
      - "/path/to/docker.json"
```

If the landscaper is deployed via helm, the credentials can be configured using the `values.yaml`:
```yaml
landscaper:
  registrySecrets: # contains optional oci secrets
    blueprints:
      default: {
                   "auths": {
                       "http://registry-harbor-registry.harbor:5000/": {
                           "auth": "${AUTH_TOKEN}"
                       }
                   }
               }
    components:
      default: {
                   "auths": {
                       "http://registry-harbor-registry.harbor:5000/": {
                           "auth": "${AUTH_TOKEN}"
                       }
                   }
               }
```

### Common Pitfalls

#### Working with the landscaper-cli
In order for the `landscaper-cli` to work with the registry, it needs valid credentials. The easiest way to generate these, would be via `docker login`. Here is an example for a registry accessible via port-forwarding:
```shell
docker login -u my-user localhost:5000 # use the user name and pwd as specified in the harbor chart values.yaml
```

Later, when dealing with artifacts like the component descriptor, be aware that the URLs used to push and access the artifacts differ due to the port-forwarding. Make sure the base URL points to the cluster-internal representation of the registry:

```yaml
  repositoryContexts:
  - type: ociRegistry
    baseUrl: harbor-harbor-registry.harbor:5000/comp
```
But push explicitly to `localhost` instead implicitly using the baseUrl:

```shell
landscaper-cli  componentdescriptor push localhost:5000/comp/ github.com/gardener/landscaper/ingress-nginx v0.1.0 component-descriptor.yaml
```

#### Targets
For test purposes it might be feasible to deploy the landscaper as well as installations into the same (local) cluster. When creating a `target` resource, make sure the `clusters[].server` points to the cluster-internal representation of the API server (i.e `kubernetes.default.svc.cluster.local.`).

### Kubernetes Clusters

- [kind](https://github.com/kubernetes-sigs/kind) - _tested_
- [minikube](https://github.com/kubernetes/minikube)
- [Gardener](https://github.com/gardener/gardener) - _tested_
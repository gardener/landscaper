# Deploy and Use the Landscaper locally

This tutorial describes how a landscaper can be deployed locally.
The only prerequisite is a kubernetes cluster 
(find [below](#kubernetes-clusters) a list of possible local kubernetes installers but basically every k8s should work).

### Install a container registry
The Landscaper depends on an oci compliant registry to fetch Blueprints, Component Descriptors and other artifacts.
[Harbor](https://github.com/goharbor/harbor-helm) can be used as such a registry that can also run beside the Landscaper in a kubernetes cluster.

Configure the harbor with a values yaml file:
```
# values.yaml
registry:
  credentials:
    username: "myuser"
    password: ""
```

```
helm repo add harbor https://helm.goharbor.io
helm upgrade --install -n ls-system registry harbor/harbor -f values.yaml
```

Helm installs the registry in the `ls-system` namespace.
The registry can be used by the landscaper by referencing artifacts via it's internal service `harbor.ls-system:80`

The harbor is now only accessible from within the cluster, so the k8s proxy is needed to upload artifacts:
```shell script
kubectl -n ls-system port-forward svc/registry-harbor-registry 5000:5000

# upload helm chart oci artifact

# add open source helm registry
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo add stable https://kubernetes-charts.storage.googleapis.com/
helm repo update

# download the helm artifact locally
helm pull ingress-nginx/ingress-nginx --untar --destination /tmp

# upload the oci artifact to a oci registry
export HELM_EXPERIMENTAL_OCI=1
helm registry login -u myuser localhost:5000 # use the username and pw configured in the values.yaml
helm chart save /tmp/ingress-nginx localhost:5000/charts/ingress-nginx:0.0.1
helm chart push localhost:5000/charts/ingress-nginx:0.0.1

# chart ref inside the cluster: registry-harbor-registry.ls-system:5000/charts/ingress-nginx:0.0.1
```

__Expose the Harbor__:
The harbor can also be exposed so that no `port-forward` proxy is needed to upload artifacts.
:warning: Keep in mind that the kubernetes cluster must support egress traffik and a ingress controller has to be installed.
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
# helm registry login -u myuser localhost:5000 # optional if configured in the values.yaml
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
{
    "auths": {
        "http://registry-harbor-registry.ls-system:5000/": {
            "auth": "${AUTH_TOKEN}"
        }
    }
}
```



##### Kubernetes Clusters

- [kind](https://github.com/kubernetes-sigs/kind) - _tested_
- [minikube](https://github.com/kubernetes/minikube)
- [Gardener](https://github.com/gardener/gardener) - _tested_
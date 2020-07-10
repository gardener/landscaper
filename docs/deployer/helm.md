# Helm Deployer

The helm deployer is a controller that reconciles DeployItems of type `Helm`.
It renders a given helm chart and deploys the resulting manifest into a cluster.

### Configuration
This sections describes the provider specific configuration
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployItem
metadata:
  name: my-nginx
spec:
  type: Helm
  importRef:
    name: secret-item1
    namespace: default

  config:
    apiVersion: helm.deployer.landscaper.gardener.cloud
    kind: ProviderConfiguration

    repository: myrepo/charts/nginx-ingress
    version: 0.5.2

    # base64 encoded kubeconfig pointing to the cluster to install the chart
    kubeconfig: xxx
    
    # Name of the release: helm install [name]
    name: my-release 
    # Namespace of the release: helm --namespace
    namespace: default
    # Values to template the chart
    # optional
    values: {}
    
    # Specifies the resource, that should be read from the templated files
    # The specified jsonPath value is written with the given key to the exported configuration.
    exportsFromManifests:
    - jsonPath: .spec.config
      key: ingressClass
      resource:
        apiGroup: v1
        kind: Secret
        name: my-secret
        namespace: a
```
### Status
This section describes the provider specific status of the resource
```yaml
status:
  providerStatus:
    apiVersion: helm.deployer.landscaper.gardener.cloud
    kind: ProviderStatus
    managedResources:
    - apiGroup: k8s.apigroup.com/v1
      kind: my-type
      name: my-resource
      namespace: default
```
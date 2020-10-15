# Kubernetes Manifest Deployer

The kubernetes manifest deployer is a controller that reconciles DeployItems of type `landscaper.gardener.cloud/kubernetes-manifest`.
It deploys the configured kubernetes manifest into the target cluster.

### Configuration
This sections describes the provider specific configuration
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployItem
metadata:
  name: my-manifests
spec:
  type: landscaper.gardener.cloud/kubernetes-manifest

  target: # has to be of type landscaper.gardener.cloud/kubernetes-cluster
    name: my-cluster
    namespace: test

  config:
    apiVersion: manifest.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderConfiguration

    updateStrategy: update | patch # optional; defaults to update
    
    manifests: # list of kubernetes manifests
    - apiVersion: v1
      kind: Secret
      metadata:
         name: my-secret
         namespace: default
      data:
        config: abc
    - ...
    
```

### Status
This section describes the provider specific status of the resource
```yaml
status:
  providerStatus:
    apiVersion: manifest.deployer.landscaper.gardener.cloud
    kind: ProviderStatus
    managedResources:
    - apiGroup: k8s.apigroup.com/v1
      kind: my-type
      name: my-resource
      namespace: default
```
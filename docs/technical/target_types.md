# Target Types

This documents the known target types that are used by the deployers.
Note that every deployer might support a subset or own target types.

**Index**:
- [Kubernetes Cluster](#kubernetes-cluster)

### Kubernetes Cluster

**Type**: `landscaper.gardener.cloud/kubernetes-cluster`
**Config**:
```yaml
config:
  # either specify the kubeconfig as string or as secret ref
  # Depending on the deployers the secretref can point to a secret in the landscaper cluster or the host cluster of the deployer.
  kubeconfig: | 
     apiVersion: v1
     kind: Config
     ....
  kubeconfig:
    secretRef:
      name: my-secret
      namespace: default
      key: kubeconfig # optional will default to "kubeconfig"
```

**Known supported Deployers**: Helm Deployer, Manifest Deployer

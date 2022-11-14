# Target Types

This documents the known target types that are used by the deployers.
Note that every deployer might support a subset or own target types.

**Index**:
- [Kubernetes Cluster](#kubernetes-cluster)

### Kubernetes Cluster

The target type `landscaper.gardener.cloud/kubernetes-cluster` contains the access data to a kubernetes cluster.

**Type**: `landscaper.gardener.cloud/kubernetes-cluster`

There are two variants for the configuration of targets of type  `landscaper.gardener.cloud/kubernetes-cluster`

**Config Variant 1**:

This variant contains the kubeconfig in the `config` section.

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Target
metadata:
    name: ...
    namespace: ...
spec:
    config:
      kubeconfig: | 
         apiVersion: v1
         kind: Config
         ....
```

**Config Variant 2**:

**DEPRECATED: use the Target's `spec.secretRef` field instead**

This variant contains the kubeconfig in a secrets referenced under an entry `kubeconfig` in the `config` section. The 
key in the data section, where to find the kubeconfig, could be specified. This is the old format before secret references 
where introduced on the top level of the `config` field. Currently, the secret must be in the same namespace as the target.

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Target
metadata:
    name: ...
    namespace: ...
spec:
    config:
      kubeconfig:
        secretRef:
          name: my-secret
          namespace: default
          key: kubeconfig # optional will default to "kubeconfig"
```

**Known supported Deployers**: Helm Deployer, Manifest Deployer, Container Deployer

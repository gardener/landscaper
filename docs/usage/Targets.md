---
title: Targets
sidebar_position: 5
---

# Targets

## Definition

Targets are a specific type of import and contain additional information that is interpreted by deployers.
The concept of a Target is to define the environment where a deployer installs/deploys software.
This means that targets could contain additional information about that environment (e.g. that the target cluster is in 
a fenced environment and needs to be handled by another deployer instance).

The configuration structure of targets is defined by their type (currently the type is only for identification but later 
we plan to add some type registration with checks.)

## Inline Configuration vs. Secret Reference

The content of a Target can be provided in two different ways: either inline in the Target, or as a reference to a secret containing the actual value.

All of the example Targets given below result in the same Target content.

### Inline Configuration

The easiest way is to just put the configuration inline into the Target's `config` field:

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Target
metadata:
  name: my-cluster
spec:
  type: landscaper.gardener.cloud/kubernetes-cluster
  config:
    kubeconfig: |
      apiVersion: v1
      clusters:
      - cluster:
          certificate-authority-data: ...
          server: https://my-apiserver.example.com
        name: default
      contexts:
      - context:
          cluster: default
          user: default
        name: default
      current-context: default
      kind: Config
      preferences: {}
      users:
      - name: default
        user:
          token: ...
```

### Secret Reference

Alternatively, the Target can also reference a secret containing the actual value instead:

Secret:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: cluster-access
type: Opaque
data:
  kubeconfig: |
    YXBpVmVyc2lvbjogdjEKY2x1c3RlcnM6Ci0gY2x1c3RlcjoKICAgIGNlcnRpZmljYXRlLWF1dGhvcml0eS1kYXRhOiAuLi4KICAgIHNlcnZlcjogaHR0cHM6Ly9teS1hcGlzZXJ2ZXIuZXhhbXBsZS5jb20KICBuYW1lOiBkZWZhdWx0CmNvbnRleHRzOgotIGNvbnRleHQ6CiAgICBjbHVzdGVyOiBkZWZhdWx0CiAgICB1c2VyOiBkZWZhdWx0CiAgbmFtZTogZGVmYXVsdApjdXJyZW50LWNvbnRleHQ6IGRlZmF1bHQKa2luZDogQ29uZmlnCnByZWZlcmVuY2VzOiB7fQp1c2VyczoKLSBuYW1lOiBkZWZhdWx0CiAgdXNlcjoKICAgIHRva2VuOiAuLi4K
  # decoded value for convenience:
    # apiVersion: v1
    # clusters:
    # - cluster:
    #     certificate-authority-data: ...
    #     server: https://my-apiserver.example.com
    #   name: default
    # contexts:
    # - context:
    #     cluster: default
    #     user: default
    #   name: default
    # current-context: default
    # kind: Config
    # preferences: {}
    # users:
    # - name: default
    #   user:
    #     token: ...

```

Target:
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Target
metadata:
  name: my-cluster
spec:
  type: landscaper.gardener.cloud/kubernetes-cluster
  secretRef:
    name: cluster-access
```

Targets can only reference secrets in their own namespace.

It is also possible to not reference a complete secret - in which case the complete structure from its `data` field will be interpreted as the Target's content - but only the value of a specific key. To do so, add the `key` to the secret reference:

Secret:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: cluster-access
type: Opaque
data:
  cluster1: |
    a3ViZWNvbmZpZzogfAogIGFwaVZlcnNpb246IHYxCiAgY2x1c3RlcnM6CiAgLSBjbHVzdGVyOgogICAgICBjZXJ0aWZpY2F0ZS1hdXRob3JpdHktZGF0YTogLi4uCiAgICAgIHNlcnZlcjogaHR0cHM6Ly9teS1hcGlzZXJ2ZXIuZXhhbXBsZS5jb20KICAgIG5hbWU6IGRlZmF1bHQKICBjb250ZXh0czoKICAtIGNvbnRleHQ6CiAgICAgIGNsdXN0ZXI6IGRlZmF1bHQKICAgICAgdXNlcjogZGVmYXVsdAogICAgbmFtZTogZGVmYXVsdAogIGN1cnJlbnQtY29udGV4dDogZGVmYXVsdAogIGtpbmQ6IENvbmZpZwogIHByZWZlcmVuY2VzOiB7fQogIHVzZXJzOgogIC0gbmFtZTogZGVmYXVsdAogICAgdXNlcjoKICAgICAgdG9rZW46IC4uLgo=
  # decoded value for convenience:
    # kubeconfig: |
    #   apiVersion: v1
    #   clusters:
    #   - cluster:
    #       certificate-authority-data: ...
    #       server: https://my-apiserver.example.com
    #     name: default
    #   contexts:
    #   - context:
    #       cluster: default
    #       user: default
    #     name: default
    #   current-context: default
    #   kind: Config
    #   preferences: {}
    #   users:
    #   - name: default
    #     user:
    #       token: ...
```

Target:
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Target
metadata:
  name: my-cluster
spec:
  type: landscaper.gardener.cloud/kubernetes-cluster
  secretRef:
    name: cluster-access
    key: cluster1
```

Note that the value of `cluster1` in the secret now not only contains the kubeconfig, but a struct with a `kubeconfig` key instead.


#### Resolving Secret References

The deployers have to take care of resolving secret references in Targets. If the deployer library is used, this is handled by the library and the functions which have to be implemented by the deployer get the already resolved Target in form of a [ResolvedTarget](../api-reference/core.md#resolvedtarget) struct. This struct has a `Content` field which contains the content of the Target, independently of whether it was specified inline or via a reference in the Target.

If you write your own deployer without using the deployer library, you will have to take care of resolving secret references in Targets yourself.

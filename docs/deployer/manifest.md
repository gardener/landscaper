# Kubernetes Manifest Deployer

The kubernetes manifest deployer is a controller that reconciles DeployItems of type `landscaper.gardener.cloud/kubernetes-manifest`.
It deploys the configured kubernetes manifest into the target cluster.

It also checks by default the healthiness of the following resources:
* `Pod`: It is considered healthy if it successfully completed
or if it has the the PodReady condition set to true.
* `Deployment`: It is considered healthy if the controller observed
its current revision and if the number of updated replicas is equal
to the number of replicas.
* `ReplicaSet`: It is considered healthy if its controller observed
its current revision and if the number of updated replicas is equal to the number of replicas.
* `StatefulSet`: It is considered healthy if its controller observed
its current revision, it is not in an update (i.e. UpdateRevision is empty)
and if its current replicas are equal to its desired replicas.
* `DaemonSet`: It is considered healthy if its controller observed
its current revision and if its desired number of scheduled pods is equal
to its updated number of scheduled pods.
* `ReplicationController`: It is considered healthy if its controller observed
its current revision and if the number of updated replicas is equal to the number of replicas.

### Configuration

This sections describes the provider specific configuration

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha2
kind: DeployItem
metadata:
  name: my-manifests
spec:
  type: landscaper.gardener.cloud/kubernetes-manifest

  target: # has to be of type landscaper.gardener.cloud/kubernetes-cluster
    name: my-cluster
    namespace: test

  config:
    apiVersion: manifest.deployer.landscaper.gardener.cloud/v1alpha2
    kind: ProviderConfiguration

    updateStrategy: update | patch # optional; defaults to update

    # Configuration of the health checks for the resources.
    # optional
    healthChecks:
      # Allows to disable the default health checks.
      # optional; set to false by default.
      disableDefault: true
      # Defines the time to wait before giving up on a resource
      # to be healthy. Should be changed with long startup time pods.
      # optional; default to 60 seconds.
      timeout: 30s

    # Defines the time to wait before giving up on a resource to be deleted,
    # for instance when deleting resources that are not anymore managed from this DeployItem.
    # optional; default to 60 seconds.
    deleteTimeout: 2m

    manifests: # list of kubernetes manifests
    - policy: manage | fallback | ignore | keep
      manifest:
        apiVersion: v1
        kind: Secret
        metadata:
          name: my-secret
          namespace: default
        data:
          config: abc
    - ...
```

__Policy__:

- `manage`: create, update and delete (occupies already managed resources)
- `fallback`: create, update and delete (only if not already managed by someone else: check for annotation with landscaper identity, deployitem name + namespace)
- `keep`: create, update
- `ignore`: forget

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

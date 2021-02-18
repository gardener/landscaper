# Helm Deployer

The helm deployer is a controller that reconciles DeployItems of type `landscaper.gardener.cloud/helm`. It renders a given helm chart and deploys the resulting manifest into a cluster.

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

This sections describes the provider specific configuration.

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployItem
metadata:
  name: my-nginx
spec:
  type: landscaper.gardener.cloud/helm
  
  target: # has to be of type landscaper.gardener.cloud/kubernetes-cluster
    name: my-cluster
    namespace: test

  config:
    apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderConfiguration
    
    chart:
      ref: myrepo.example.com/charts/nginx-ingress:0.5.2 # helm oci ref
      fromResource: # will fetch the helm chart from component descriptor resource of type helm chart
#       inline: # define an inline component descriptor instead of referencing a remote
        ref:
          repositoryContext:
            type: ociRegistry
            baseUrl: my-repo
          componentName: github.com/gardener/landscaper
          version: v0.3.0
        resourceName: my-helm-chart
      archive:
        raw: "" # base64 encoded helm chart tar.gz
        remote:
          url: "https://my-domain/maychart.tar.gz"

    # base64 encoded kubeconfig pointing to the cluster to install the chart
    kubeconfig: xxx

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

    # Name of the release: helm install [name]
    name: my-release
    # Namespace of the release: helm --namespace
    namespace: default
    # Values to template the chart
    # optional
    values: {}

    # Describes one export that is read from the templates values or a templated resource.
    # The value will be by default read from the values if fromResource is not specified.
    # The specified jsonPath value is written with the given key to the exported configuration.
    exportsFromManifests:
    - key: KeyA # value is read from the values file
      jsonPath: .Values.keyA
    - key: KeyB # value is read from secret
      jsonPath: .spec.config
      fromResource:
        apiVersion: v1
        kind: Secret
        name: my-secret
        namespace: a
```

Exports can be defined in `exportsFromManifests` by specifying the exported key to export.
The value is taken from a rendered resource and a jsonpath to the value.
For a complete documention of the availabel jsonPath see here (https://kubernetes.io/docs/reference/kubectl/jsonpath/).

:warning: Only unique identifiable resources (_apiVersion_, _kind_, _name_ and _namespace_).

### Status

This section describes the provider specific status of the resource.

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

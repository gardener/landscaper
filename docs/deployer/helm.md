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
  type: landscaper.gardener.cloud/helm
  
  target: # has to be of type landscaper.gardener.cloud/kubernetes-cluster
    name: my-cluster
    namespace: test

  config:
    apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderConfiguration
    
    chart:
      ref: myrepo.example.com/charts/nginx-ingress:0.5.2
#      tar: "" #  base64 encoded helm chart tar

    # base64 encoded kubeconfig pointing to the cluster to install the chart
    kubeconfig: xxx

    updateStrategy: update | patch # optional; defaults to update

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

:warning: only unique identifiable resources (apiVersion, kind, name and namespace).


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
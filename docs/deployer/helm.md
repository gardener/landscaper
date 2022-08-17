# Helm Deployer

The helm deployer is a controller that reconciles DeployItems of type `landscaper.gardener.cloud/helm`. There are two
alternative working modes of the helm deployer. One renders a given helm chart and deploys the resulting manifest into 
a cluster. The other deploys a helm chart with [helm 3](https://helm.sh/). The second case is the default.

By default, the helm deployer checks the health of the deployed resources. See [healthchecks.md](healthchecks.md) for more info.

**Index**:
- [Provider Configuration](#provider-configuration)
- [Provider Status](#status)
- [Deployer Configuration](#deployer-configuration)

## Provider Configuration

This sections describes the provider specific configuration. The following example defines a deployment which deploys
a helm chart with helm 3. 

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

    # settings for the different helm 3 operations 
    helmDeploymentConfig:
      install: # see  https://helm.sh/docs/helm/helm_install/#options
        atomic: true
        timeout: 10m
      upgrade: # see https://helm.sh/docs/helm/helm_upgrade/#options
        atomic: true
        timeout: 10m
      uninstall: # see https://helm.sh/docs/helm/helm_uninstall/#options
        timeout: 15m

    # base64 encoded kubeconfig pointing to the cluster to install the chart
    kubeconfig: xxx

    updateStrategy: update | patch # optional; defaults to update

    # Configuration of the readiness checks for the resources.
    # optional
    readinessChecks:
      # Allows to disable the default readiness checks.
      # optional; set to false by default.
      disableDefault: true
      # Defines the time to wait before giving up on a resource
      # to be ready. Should be changed with long startup time pods.
      # optional; defaults to 180 seconds/3 minutes.
      timeout: 3m
      # Configuration of custom readiness checks which are used
      # to check on custom fields and their values
      # especially useful for resources that came in through CRDs
      # optional
      custom:
      # the name of the custom readiness check, required
      - name: myCustomReadinessCheck
        # timeout of the custom readiness check
        # optional, defaults to the timeout stated above
        timeout: 2m
        # temporarily disable this custom readiness check, useful for test setups
        # optional, defaults to false
        disabled: false
        # a specific resource should be selected for this readiness check to be performed on
        # a resource is uniquely defined by its GVK, namespace and name
        # required if no labelSelector is specified, can be combined with a labelSelector which is potentially harmful
        resourceSelector:
          apiVersion: apps/v1
          kind: Deployment
          name: myDeployment
          namespace: myNamespace
        # multiple resources for the readiness check to be performed on can be selected through labels
        # they are identified by their GVK and a set of labels that all need to match
        # required if no resourceSelector is specified, can be combined with a resourceSelector which is potentially harmful
        labelSelector:
          apiVersion: apps/v1
          kind: Deployment
          matchLabels:
            app: myApp
            component: backendService
        # requirements specifies what condition must hold true for the given objects to pass the readiness check
        # multiple requirements can be given and they all need to successfully evaluate
        requirements:
        # jsonPath denotes the path of the field of the selected object to be checked and compared
        - jsonPath: .status.readyReplicas
          # operator specifies how the contents of the given field should be compared to the desired value
          # allowed operators are: DoesNotExist(!), Exists(exists), Equals(=, ==), NotEquals(!=), In(in), NotIn(notIn)
          operator: In
          # values is a list of values that the field at jsonPath must match to according to the operators
          values:
          - value: 1
          - value: 2
          - value: 3

    # Defines the time to wait before giving up on a resource to be deleted,
    # for instance when deleting resources that are not managed by this DeployItem anymore.
    # optional; default to 180 seconds/3 minutes.
    deleteTimeout: 2m

    # Name of the release: helm install [name]
    name: my-release
    # Namespace of the release: helm --namespace
    namespace: default
    # configure the landscaper to automatically create the release namespace.
    # Works similar to "helm install --create-namespace"
    createNamespace: true
    # Values to template the chart
    # optional
    values:
      KeyA: valA

    # Define exports that are read from the kubernetes resources or helm values,
    # so they can be used by other deployitems or installations.
    # The deployer tries to read the export values until either the global or the specific timeout is exceeded.
    exports:
      defaultTimeout: 5m # global default timeout that is used when no specific timeout is set
      exports:
      - key: KeyA # value is read from the helm values
        jsonPath: .Values.keyA # points to the key in the helm values 
      - key: KeyB # value is read from a secret and exported with name "KeyB"
        timeout: 10m # optional specific timeout
        jsonPath: .data.somekey # points to the value in the resource that is being exported
        fromResource: # required
          apiVersion: v1 # specification of the resource type
          kind: Secret
          name: my-secret # name of the resource
          namespace: a # namespace of the resource
      - key: KeyC # value is read from secret that is referenced by a service account and exported with name "KeyC"
        timeout: 10m # optional specific timeout
        jsonPath: .secrets[0] # points to an object reference that consists of a name and namespace
        fromResource:
          apiVersion: v1 # specification of the resource type
          kind: ServiceAccount
          name: my-user # name of the resource
          namespace: a # namespace of the resource
        # Defines the referenced objects kind and version. 
        # The name and namespace is taken from the jsonPath defined in "fromResource".
        fromObjectRef:
          apiVersion: v1
          kind: Secret
          jsonPath: ".data.somekey" # points to the value in the resource that is being exported
```

Exports can be defined in `exportsFromManifests` by specifying the exported key to export.
The value is taken from a rendered resource and a jsonpath to the value.
For a complete documentation of the available jsonPath see here (https://kubernetes.io/docs/reference/kubectl/jsonpath/).

:warning: Only unique identifiable resources (_apiVersion_, _kind_, _name_ and _namespace_).

## Manifest-Only Deployment

If you want to deploy the chart not with helm 3 but only apply the manifests you just need to add the field 
`helmDeployment: false` to the provider configuration as shown in the following example. 

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
      ...

    # specifies that only the rendered manifests are applied
    helmDeployment: false
```

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

## Deployer Configuration

When deploying the helm deployer controller it can be configured using the `--config` flag and providing a configuration file.

The structure of the provided configuration file is defined as follows.

:warning: Keep in mind that when deploying with the helm chart the configuration is abstracted using the helm values. 
See the [helm values file](../../charts/helm-deployer/values.yaml) for details when deploying with the helm chart.

```yaml
apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
kind: Configuration
oci:
  # allow plain http connections to the oci registry.
  # Use with care as the default docker registry does not serve http with any authentication
  allowPlainHttp: false
  # skip the tls validation
  insecureSkipVerify: false
  # path to docker compatible auth configuration files.
#  configFiles:
#  - "somepath"

# target selector to only react on specific deploy items.
# see the common config in "./README.md" for detailed documentation.
targetSelector:
  annotations: []
  labels: []
```

## Support of Helm Chart Repositories

The example above requires that the helm chart is stored in an OCI registry, but also helm chart repositories are 
supported. There are two alternatives how to specify the helm chart: either directly in the provider configuration of 
the deploy item, or via a resource in a component descriptor.

#### Specifying a helm chart in the provider configuration

To read a chart from a helm chart repository, you can specify the repository URL, chart name, and chart version 
in field `chart.helmChartRepo` of the provider configuration, as in this example:

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployItem
metadata:
  name: my-nginx
spec:
  type: landscaper.gardener.cloud/helm

  config:
    apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderConfiguration
    
    chart:
      helmChartRepo:
        helmChartRepoUrl: https://charts.bitnami.com/bitnami
        helmChartName: nginx
        helmChartVersion: 9.7.1
    ...
```

The full example can be found 
[here](https://github.com/gardener/landscaper-examples/tree/master/helm-deployer/real-helm-deployment).

#### Specifying a helm chart via component descriptor

Alternatively, the provider configuration can reference a resource in the component descriptor.
Repository URL, chart name, and chart version are then specified in that resource in the component descriptor. 

Provider configuration:

```yaml
    chart:
      helmChartRepo:
        {{- $resource := getResource .cd "name" "nginx-chart" }}
        helmChartRepoUrl: {{ $resource.access.helmChartRepoUrl }}
        helmChartName:    {{ $resource.access.helmChartName }}
        helmChartVersion: {{ $resource.access.helmChartVersion }}
```

Component descriptor:

```yaml
meta:
  schemaVersion: 'v2'
component:
  name: 'github.com/gardener/landscaper-examples/helm-deployer/real-helm-deployment-cd'
  version: 'v0.1.0'
  ...
  resources:
    - type: helm.io/chart
      name: nginx-chart
      version: 9.7.0
      relation: external
      access:
        type: helmChartRepository
        mediaType: application/octet-stream
        helmChartRepoUrl: https://charts.bitnami.com/bitnami
        helmChartName: nginx
        helmChartVersion: 9.7.0
```

Note that the access type of the resource (field `access.type`) must be `helmChartRepository`.

The full example can be found 
[here](https://github.com/gardener/landscaper-examples/tree/master/helm-deployer/real-helm-deployment-cd).

#### Access to Helm Chart Repo with Authentication

If your helm chart repository is protected proceed as follows:

1) Create a [context CR](../usage/Context.md) in the same namespace as your installation.

```
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Context
metadata:
  name: helm-repo-protected
  namespace: example

repositoryContext:
  baseUrl: eu.gcr.io/gardener-project/landscaper/examples
  type: ociRegistry

registryPullSecrets: []

configurations:
  helmChartRepoCredentials:
    auths:
      - url: "your.protected.helmchart.repo.com"
        authHeader: "Basic dX3d...cmQ=" 
```

The field `repositoryContext` contains the base URL of your component repository. 

The field `configurations` contains an entry `helmChartRepoCredentials` with the URL and the authentication header of your 
protected helm chart repo (you might configure more than one).

2) Use the context in your installation

```
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: helm-repo-protected
  namespace: example

spec:
  blueprint:
    ref:
      resourceName: blueprint

  context: helm-repo-protected

  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper-examples/helm-deployer/helm-repo-protected
      version: v0.1.0
```

You can skip the field `repositoryContext` under `spec.componentDescriptor.ref` because this is fetched from the context
CR.

You find a complete example [here](https://github.com/gardener/landscaper-examples/tree/master/helm-deployer/helm-repo-protected).

## Examples

Other example could be found
[here](https://github.com/gardener/landscaper-examples/tree/master/helm-deployer).

# Tutorial 01: Developing a simple Blueprint

This tutorial describes the basics of developing Blueprints. It covers the whole manual workflow from wrtting the Blueprint together with a Component Descriptor and storing them in a remote OCI repository.

For this tutorial, we are going to use the [NGINX ingress controller](https://github.com/kubernetes/ingress-nginx/tree/master/charts/ingress-nginx) as the example application which will get deployed via its upstream helm chart.

## Prerequisites

For this tutorial, you will need:

- the Helm (v3) commandline tool (see https://helm.sh/docs/intro/install/)
- [OPTIONAL] an OCI compatible registry (e.g. GCR or Harbor)
- a Kubernetes Cluster (better use two different clusters: one which Landscaper runs in and one that NGINX gets installed into)

You will also need the `landscaper-cli` and `component-cli` command line tools. Their installation is described [here](https://github.com/gardener/landscapercli/blob/master/docs/installation.md) and [here](https://github.com/gardener/component-cli) respectively.

All example resources can be found in the folder [./resources/ingress-nginx](./resources/ingress-nginx) of this repository.

:warning: Note that the repository `eu.gcr.io/gardener-project/landscaper/tutorials` that is used throughout this tutorial is an example repository and has to be replaced with the path to your own registry if you want to upload your own artifacts.
If you do not have your own OCI registry, you can of course reuse the artifacts that we provided at `eu.gcr.io/gardener-project/landscaper/tutorials` which are publicly readable.

## Structure

- [Prerequisites](#prerequisites)
1. [Prepare the NGINX helm chart](#Step-1:-Prepare-the-NGINX-helm-chart)
1. [Define the Component Descriptor](#Step-2:-Define-the-Component-Descriptor)
1. [Create a Blueprint](#Step-3:-Create-a-Blueprint)
1. [Render and Validate the Blueprint locally](#Step-4:-Render-and-Validate-the-Blueprint-locally)
1. [Remote Upload](#Step-5:-Remote-Upload)
1. [Installation](#Step-6:-Installation)
- [Summary](#summary)
- [Up next](#up-next)

## Step 1: Prepare the NGINX helm chart

The current helm deployer only supports helm charts stored in an OCI registry. We therefore have to convert and upload the open source helm chart as an OCI artifact to our registry.

```shell script
# add open source and nginx helm registries
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo add stable https://charts.helm.sh/stable
helm repo update

# download the nginx ingress helm chart and extract it to /tmp/nginx-ingress
helm pull ingress-nginx/ingress-nginx --untar --destination /tmp

# upload the helm chart to an OCI registry
export OCI_REGISTRY="eu.gcr.io" # <-- replace this with the URL of your own OCI registry
export CHART_REF="$OCI_REGISTRY/mychart/reference:my-version" # e.g. eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:v3.29.0
export HELM_EXPERIMENTAL_OCI=1
helm registry login -u myuser $OCI_REGISTRY
helm chart save /tmp/ingress-nginx $CHART_REF
helm chart push $CHART_REF
```

## Step 2: Define the Component Descriptor

A Component Descriptor contains references and locations to all _resources_ that are used by Landscaper to deploy and install an application. In this example, the only kind of _resources_ is a `helm` chart (that of the nginx-ingress controller that we uploaded to an OCI registry in the previous step) but it could also be `oci images` or even `node modules`.

If a Helm chart is referenced through a component descriptor, the version of the chart in the component descriptor should match the version of the chart itself. Since we are using version v3.29.0 of the _ingress-nginx_ Helm chart in this tutorial, the component descriptor references it accordingly.

For more information about the component descriptor and the usage of the different fields, refer to the [component descriptor documentation](https://github.com/gardener/component-spec).

```yaml
meta:
  schemaVersion: v2

component:
  name: github.com/gardener/landscaper/nginx-ingress
  version: v0.2.1

  provider: internal
  sources: []
  componentReferences: []

  resources:
  - type: helm
    name: ingress-nginx-chart
    version: v3.29.0
    relation: external
    access:
      type: ociRegistry
      imageReference: eu.gcr.io.gardener-project/landscaper/tutorials/charts/ingress-nginx:v3.29.0
```

## Step 3: Create a Blueprint

Blueprints describe how _DeployItems_ are created by taking the values of `imports` and applying them to templates inside `deployExecutions`. Additionally, they specify which pieces of data appear as `exports` from the executed _DeployItems_.

For detailed documentation about Blueprints, look at [docs/usage/Blueprints.md](/docs/usage/Blueprints.md).

### Imports and Exports declaration

The `imports` are described as a list of import declarations. Each _import_ is declared by a unique name and a type which is either a JSON schema or a `targetType`.

<details>

Imports with the type `schema` import their data from a data object with a given JSON schema. Imports with the type `targetType` are imported from the specified _Target_.

```yaml
# import with type JSON schema

- name: myimport
  schema: # valid jsonschema
    type: string | object | number | ...
```

```yaml
# import from/into targetType

- name: myimport
  targetType: "" # e.g. landscaper.gardener.cloud/kubernetes-cluster
```

</details>

Our _nginx-ingress_ controller in this tutorial only needs to import a target Kubernetes cluster and a namespace in the target cluster. 
The target will be used as the k8s cluster where the Helm chart gets deployed to. 
The following YAML snippet _declares_ the imports:

```yaml
imports:
- name: cluster
  targetType: landscaper.gardener.cloud/kubernetes-cluster
# the namespace is expected to be a string
- name: namespace
  schema:
    type: string
```

The declaration of `exports` works just like declaring `imports`. Again, each _export_ is declared as a list-item with a unique name and a data type (again, JSON schema or `targetType`).

To be able to use the ingress in a later Blueprint or Installation, this Blueprint will export the name of the ingress class as a simple string. With this piece of YAML, the export is _declared_:

```yaml
exports:
- name: ingressClass
  schema: # here comes a valid jsonschema
    type: string
```

### DeployItems

_DeployItems_ are created from templates that are given in the `deployExecutions` section. Each element specifies a templating step which will result in one or multiple _DeployItems_ (returned as a list).

```yaml
- name: "unique name of the deployitem"
  type: landscaper.gardener.cloud/helm | landscaper.gardener.cloud/container | ... # deployer identifier
  # names of other deployitems that the deploy item depends on.
  # If a item depends on another, the landscaper ensures that dependencies are executed and reconciled before the item.
  dependsOn: []
  config: # provider specific configuration
    apiVersion: mydeployer.landscaper.gardener.cloud/test
    kind: ProviderConfiguration
    ...
```

At the moment, the supported templating engines are [GoTemplate](https://golang.org/pkg/text/template/) and [Spiff](https://github.com/mandelsoft/spiff). For detailed information about the template executors, [read this](/docs/usage/TemplateExecutors.md).

While processing the templates, Landscaper offers access to the `imports` and the fields of the component descriptor through the following structure:

```yaml
imports:
  <import name>: <data value> or <target custom resource>
cd:
 component:
   resources: ...
```

Access to individual component resources is possible through the template function `getResource` and any of their fields (e.g. the `name` field and its value `ingess-nginx-chart`):

```yaml
{{ $resource := getResource .cd "name" "ingress-nginx-chart" }}
```

Exports can be described the same way as imports and they can be templated using template executions in the `exportExecutions` - just like `deployExections`.
Export execution are expected to output the exports as a map of <export name>: <value> .

<details>

If a target gets exported, it is expected to adhere to the following structure:

```yaml
<target export name>:
  annotations: {} # optional
  lables: {} # optional
  config:
    type: ""
    config: {}
```

To export values of _DeployItems_ and _Installations_, Landscaper gives access to them via templating imports:

```yaml
values:
  deployitems:
    <deploy item name>: <deploy item export value (is type specific)>
  dataobjects:
    <data object name>: <data of the dataobject> (currently only exports of the subinstallations are accessible)
  targets:
    <target name>: <the target cr>
```

</details>

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

imports:
- name: cluster
  type: target
  targetType: landscaper.gardener.cloud/kubernetes-cluster

deployExecutions:
- name: default
  type: GoTemplate
  template: |
    deployItems:
    - name: deploy
      type: landscaper.gardener.cloud/helm
      target: 
        name: {{ .imports.cluster.metadata.name }}
        namespace: {{ .imports.cluster.metadata.namespace }}
      config:
        apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
        kind: ProviderConfiguration
        
        chart:
          {{ $resource := getResource .cd "name" "ingress-nginx-chart" }}
          ref: {{ $resource.access.imageReference" }}
        
        updateStrategy: patch
        
        name: test
        namespace: {{ .imports.namespace }}
        
        exportsFromManifests:
        - key: ingressClass
          jsonPath: .Values.controller.ingressClass

exportExecutions:
- name: default
  type: GoTemplate
  template: |
    exports:
      ingressClass: {{ .values.deployitems.deploy.ingressClass }}

exports:
- name: ingressClass
  type: data
  schema: # here comes a valid jsonschema
    type: string
```

A Blueprint is actually a directory that contains the above described Blueprint manifest as file called `blueprint.yaml`.
The directory may contain any other data that is necessary for the deployment/templating.
For an example see [./resources/ingress-nginx/blueprint](resources/ingress-nginx/blueprint).

## Step 4: Render and Validate the Blueprint locally

The Blueprint would result in a _DeployItem_ of type _Helm_ that was derived from a template and one import. This step is called rendering.

To test the rendering locally and to have a look at the resulting DeployItem, `landscaper-cli` can be used (`landscaper-cli` will use the same rendering library as the landscaper-controller that would run within Kubernetes).

First, the import values that are used by the templating step need to be put into a file that will be provided to `landscaper-cli` (e.g. [docs/tutorials/resources/ingress-nginx/import-values.yaml](./resources/ingress-nginx/import-values.yaml)).

```yaml
imports:
  namespace: tutorial
  cluster:
    metadata:
      name: my-target
      namespace: test
    spec:
      type: ""
      config:
        kubeconfig: |
          apiVersion: ...
```

Now, with this file and the Component Descriptor (e.g. [docs/tutorials/resources/ingress-nginx/component-descriptor.yaml](./resources/ingress-nginx/component-descriptor.yaml)) at hand, the Blueprint can be rendered.

```shell script
landscaper-cli blueprints render ./docs/tutorials/resources/ingress-nginx/blueprint \
  -c ./docs/tutorials/resources/ingress-nginx/component-descriptor.yaml \
  -f ./docs/tutorials/resources/ingress-nginx/import-values.yaml
```

This is result in a DeployItem that could get picked up by a deployer in a later step.

```yaml
--------------------------------------
-- deployitems deploy
--------------------------------------
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployItem
metadata:
  annotations:
    execution.landscaper.gardener.cloud/dependsOn: ""
    landscaper.gardener.cloud/operation: reconcile
  creationTimestamp: null
  labels:
    execution.landscaper.gardener.cloud/name: deploy
spec:
  config:
    apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
    chart:
      ref: eu.gcr.io/myproject/charts/nginx-ingress:v3.29.0
    exportsFromManifests:
    - jsonPath: .Values.controller.ingressClass
      key: ingressClass
    kind: ProviderConfiguration
    name: test
    namespace: default
    updateStrategy: patch
  target:
    name: my-cluster
    namespace: tutorial
  type: landscaper.gardener.cloud/helm
status:
  observedGeneration: 0
```

## Step 5: Remote Upload

Once the development of the Blueprint is finished and it renders successfully, it has to be uploaded to an OCI registry and its reference needs to be added to the Component Descriptor.

The Blueprint can easily be uploaded with the `landscaper-cli` tool which will package the Blueprint and upload it to the given OCI registry.

```shell script
# replace the values to match your registry and file locations
landscaper-cli blueprints push \
  my-registry/my-path/ingress-nginx:v0.1.0 \
  my-local-workspace/my-blueprint-directory

# e.g. if you were to use the provided sample content
# (this will fail as you have no write access to gardener-project on eu.gcr.io)
landscaper-cli blueprints push \
  eu.gcr.io/gardener-project/landscaper/tutorials/blueprints/ingress-nginx:v0.3.0 \
  docs/tutorials/resources/ingress-nginx/blueprint
```

Blueprints are also just resources/artifacts of a Component Descriptor. Therefore, after the Blueprint got uploaded, its reference needs to be added to the Component Descriptor. This is necessary to make sure that all resources of an application are known and stored - and the Blueprint is just one of the resources of an application.
In addition, Landscaper needs this information to resolve the location of the Blueprint resource.

Note that the repository context as well as the Blueprint resource should be added to the Component Descriptor.

```yaml
meta:
  schemaVersion: v2

component:
  name: github.com/gardener/landscaper/ingress-nginx
  version: v0.3.0

  provider: internal
  sources: []
  componentReferences: []

  respositoryContext:
  - type: ociRegistry
    baseUrl: eu.gcr.io/gardener-project/landscaper/tutorials/components

  resources:  
  - type: helm
    name: ingress-nginx-chart
    version: v3.29.0
    relation: external
    access:
      type: ociRegistry
      imageReference: eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:v3.29.0
  - type: blueprint
    name: ingress-nginx-blueprint
    relation: local
    access:
      type: ociRegistry
      imageReference: eu.gcr.io/gardener-project/landscaper/tutorials/blueprints/ingress-nginx:v0.3.0
```

Finally, the Component Descriptor must be uploaded to an OCI registry. This is done witl the `component-cli` tool, which has been integrated into the `landscaper-cli`.

```shell script

# replace the values to match your registry and file locations
landscaper-cli components-cli ca remote push <path to directory with component-descriptor.yaml>

# e.g. if you were to use the provided sample content
# (this will fail as you have no write access to gardener-project on eu.gcr.io)
landscaper-cli components-cli ca remote push ./docs/tutorials/resources/ingress-nginx
```

Once the upload succeeds, the Component Descriptor should be accessible at `eu.gcr.io/gardener-project/landscaper/tutorials/components/component-descriptors/github.com/gardener/landscaper/ingress-nginx/v0.3.0` in the registry.

## Step 6: Installation

Now that all external resources are defined and uploaded to OCI registries, the nginx-ingress can finally get installed by Landscaper into our target Kubernetes cluster.

Before the runtime resources are defined, the landscaper-controller has to be installed into the first (the Landscaper-) Kubernetes cluster. For a detailed installation instructions, see the [Landscaper Controller Installation](../gettingstarted/install-landscaper-controller.md) document.

The Blueprint that we created so far in the previous steps can be installed by Landscaper into a target cluster by creating an _Installation_ resource in our Landscaper-cluster.

### Defining the _Target_ that is used as import

The Blueprint defines one import of a Kubernetes cluster, therefore, a _Target_ resource of type `landscaper.gardener.cloud/kubernetes-cluster` that points to the target cluster has to be defined. This target basically needs to contain the target cluster's kubeconfig.

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Target
metadata:
  name: my-target-cluster
spec:
  type: landscaper.gardener.cloud/kubernetes-cluster
  config:
    kubeconfig: |
      apiVersion:...
      # here goes the kubeconfig of the target cluster
```

### Providing the "namespace"-Import as configmap

The Blueprint defines another import for the namespace that should be of type `string`.
Imports that are defined by a jsonschema are called data imports.
These imports can be defined either via `DataObject`, `Secret` or `ConfigMap`.

In this tutorial the import is defined as configmap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-imports
data:
  namespace: default
```

### Defining the _Installation_ resource

An _Installation_ is an instance of a Blueprint, i.e. it is the runtime representation of one specific Blueprint installation.

An installation consists of a Blueprint, Imports and Exports.

__Component Descriptor__: Remember that a Blueprint is just another resource of a software component and thus is referenced by the Component Descriptor. We need to spcify the Component Descriptor through its repository context, the component name and its version.

```yaml
componentDescriptor:
  ref:
    repositoryContext:
      type: ociRegistry
      baseUrl: eu.gcr.io/gardener-project/landscaper/tutorials/components
    componentName: github.com/gardener/landscaper/ingress-nginx
    version: v0.3.0
```

__Blueprint__: Once the Component Descriptor is given, the Blueprint artifact in the component descriptor is specified by its resource with the unique name `ingress-nginx-blueprint`.

```yaml
blueprint:
  ref:
    resourceName: ingress-nginx-blueprint
```

__Imports__: The Blueprint needs a _Target_ import of type _kubernetes-cluster_ and a data import for the namespace.
The target `my-target-cluster` and the configmap that we created before, needs to be connected to the Blueprint's import in the Installation.

:warning: The "#" has to be used to reference the previously created target. Otherwise, Landscaper would try to import the target from another component's export.

```yaml
imports:
  targets:
  - name: cluster
    # the "#" forces the landscaper to use the target with the name "my-cluster" in the same namespace
    target: "#my-target-cluster"
  data:
  - name: namespace # name of the import in the blueprint
    configMapRef:
      key: "namespace"
      name: "my-imports" # name of the configmap;
      # namespace: default # the namespace will be defaulted to the namespace of the installation.

```

__Exports__: The nginx ingress Blueprint exports the used `ingressClass` so that it can be reused by other components. To give the generic ingress class more semantic meaning in the current installation, the export is exported as `myIngressClass`.
Other installation are now able to consume the data with this specific name.

:warning: Note that this name has to be unique so that it will not be overwritten by other installations.

The export is a _DataObject_ export, therefore the export is defined under `.spec.exports.data` and is written to the `dataRef: myIngressClass`.

```yaml
exports:
  data:
  - name: ingressClass
    dataRef: "myIngressClass"
```

### The final _Installation_ resource

The final _Installation_ resource will look like the following YAML snippet.

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: my-ingress
spec:
  componentDescriptor:
    ref:
      repositoryContext:
        type: ociRegistry
        baseUrl: eu.gcr.io/gardener-project/landscaper/tutorials/components
      componentName: github.com/gardener/landscaper/ingress-nginx
      version: v0.3.0

  blueprint:
    ref:
      resourceName: ingress-nginx-blueprint

  imports:
    targets:
    - name: cluster
      # the "#" forces the landscaper to use the target with the name "my-cluster" in the same namespace
      target: "#my-target-cluster"
    data:
    - name: namespace # name of the import in the blueprint
      configMapRef:
        key: "namespace"
        name: "my-imports" # name of the configmap;
  
  exports:
    data:
    - name: ingressClass
      dataRef: "myIngressClass"
```

### Applying the resources to Kubernetes and let Landscaper do is work

The _Target_ and the _Installation_ resources can now be applied to the Kubernetes cluster running the landscaper-controller.

```shell script
kubectl apply -f docs/tutorials/resources/ingress-nginx/my-target.yaml
kubectl apply -f docs/tutorials/resources/ingress-nginx/configmap.yaml
kubectl apply -f docs/tutorials/resources/ingress-nginx/installation.yaml
```

Landscaper will now immediately start to reconcile the _Installation_ as all imports are satisfied.

The first resource that will be created is the execution object which is a helper resource that contains the rendered deployitems. The status shows the one specified Helm _DeployItem_ which has been automatically created by Landscaper.

```shell output
$ kubectl get installation

NAME                        PHASE       CONFIGGEN   EXECUTION                   AGE
my-ingress                  Succeeded               my-ingress                  4m11s
```

```shell output
$ kubectl get execution my-execution -o yaml

apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Execution
metadata:
  [...]
spec:
  deployItems:
  - config:
      apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
      chart:
        ref: eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:v3.29.0
      exportsFromManifests:
      - jsonPath: .Values.controller.ingressClass
        key: ingressClass
      kind: ProviderConfiguration
      name: test
      namespace: default
    name: deploy
    target:
      name: ts-test-cluster
      namespace: default
    type: landscaper.gardener.cloud/helm
status:
  [...]
  deployItemRefs:
  - name: deploy
    ref:
      name: my-ingress-deploy-xxx
      namespace: default
      observedGeneration: 1
  [...]
```

### Deployers

The newly created _DeployItem_ will be reconciled by the Helm deployer. It is the Helm deployer that creates and updates the configured resources of the Helm chart in the target cluster. 

When the deployer successfully reconciled the DeployItem, the phase is set to `Succeeded` and all managed resources are added to the DeployItem's status.

```shell output
$ kubectl get di my-ingress-deploy-xxx -oyaml

apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployItem
metadata:
  [...]
spec:
  config:
    apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
    chart:
      ref: eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:v3.29.0
    exportsFromManifests:
    - jsonPath: .Values.controller.ingressClass
      key: ingressClass
    kind: ProviderConfiguration
    name: test
    namespace: default
  target:
    name: ts-test-cluster
    namespace: default
  type: landscaper.gardener.cloud/helm
status:
  exportRef:
    name: my-ingress-deploy-5stgr-export
    namespace: default
  phase: Succeeded
  providerStatus:
    apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderStatus
    managedResources:
    - apiVersion: rbac.authorization.k8s.io/v1
      kind: Role
      name: test-ingress-nginx
      namespace: default
    [...]
```

The Blueprint configured export values, therefore, the Helm deployer also creates a secret that contains the exported values.

```shell
# A kubectl plugin is used to automatically decode the base64 encoded secret
$ kubectl ksd get secret my-ingress-deploy-5stgr-export -o yaml

apiVersion: v1
kind: Secret
metadata:
  [...]
stringData:
  config: |
    ingressClass: nginx
type: Opaque
```

This exported value is then propagated to the execution object and is then used in the `exportExecutions` to create the exports.

The execution resource combines all deployitem exports into one data object.

```shell
$ kubectl get exec my-execution

NAME                        PHASE       EXPORTREF                          AGE
my-ingress                  Succeeded   3a4cwhagjhl5i6iu3vvljkjkzffxbk4p   5m
```

```shell
$ kubectl get do 3a4cwhagjhl5i6iu3vvljkjkzffxbk4p -oyaml

apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DataObject
metadata:
  ...
data:
  deploy:
    ingressClass: nginx
```

Landscaper collects the export from the execution and creates the configured exported dataobject `myIngressClass`.
The exported dataobject is a contextified dataobject, which means that it can only be imported by other installations in the same context. The dataobject's context is the root context `""` so that all root installations could use the export as import.

Contextified dataobjects name is a hash of the exported key and the context, so that they can be unqiely identified by the landscaper.

:warning: Note: also targets are contextified but global target/dataobjects can be referenced with a prefix `#` as in the current target import.

```shell script
$ kubectl get do -l data.landscaper.gardener.cloud/key=myIngressClass
NAME                               CONTEXT   KEY
dole6tby5kerlxruq2n2efxiql6onp3h             myIngressClass
```

```
$ kubectl get do -l data.landscaper.gardener.cloud/key=myIngressClass -oyaml
apiVersion: v1
kind: List
items:
- apiVersion: landscaper.gardener.cloud/v1alpha1
  kind: DataObject
  metadata:
    creationTimestamp: "2020-10-12T10:11:01Z"
    generation: 1
    labels:
      data.landscaper.gardener.cloud/key: myIngressClass
      data.landscaper.gardener.cloud/source: Installation.default.my-ingress
      data.landscaper.gardener.cloud/sourceType: export
    name: dole6tby5kerlxruq2n2efxiql6onp3h
    namespace: default
  data: nginx
```

## Summary
- A blueprint has been created that describes how a nginx ingress can be deployed into a kubernetes cluster.
- A component descriptor has been created that contains the blueprint and another external resources as local resources with access type `localOciBlob`.
- The blueprint and the component descriptor are uploaded to the oci registry.
- An installation has been defined and applied to the cluster which resulted in the deployed nginx application.

## Up Next
In the [next tutorial](./02-local-simple-blueprint.md), the same blueprint is deployed but using only component local artifacts.

# Tutorial 01: Developing a simple Blueprint

This tutorial describes the basics of developing Blueprints. It covers the whole manual workflow from writing the Blueprint together with a Component Descriptor and storing them in a remote OCI repository.

For this tutorial, we are going to use the [NGINX ingress controller](https://github.com/kubernetes/ingress-nginx/tree/master/charts/ingress-nginx) as the example application which will get deployed via its helm chart.

## Structure

  - [Prerequisites](#prerequisites)
  - [Step 1: Prepare the NGINX helm chart](#step-1-prepare-the-nginx-helm-chart)
  - [Step 2: Define the Component Descriptor](#step-2-define-the-component-descriptor)
  - [Step 3: Create a Blueprint](#step-3-create-a-blueprint)
    - [Imports and Exports declaration](#imports-and-exports-declaration)
    - [DeployItems](#deployitems)
  - [Step 4: Render and Validate the Blueprint locally](#step-4-render-and-validate-the-blueprint-locally)
  - [Step 5: Remote Upload](#step-5-remote-upload)
  - [Step 6: Installation](#step-6-installation)
    - [Defining the _Target_ that is used as import](#defining-the-target-that-is-used-as-import)
    - [Providing the "namespace"-Import as configmap](#providing-the-namespace-import-as-configmap)
    - [Defining the _Installation_ resource](#defining-the-installation-resource)
    - [The final _Installation_ resource](#the-final-installation-resource)
    - [Applying the resources to Kubernetes and let Landscaper do is work](#applying-the-resources-to-kubernetes-and-let-landscaper-do-is-work)
    - [Deployers](#deployers)
  - [Summary](#summary)
  - [Up Next](#up-next)
  
  
## Prerequisites

For this tutorial, you will need:

- the Helm (v3) commandline tool (see https://helm.sh/docs/intro/install/)
  - your helm version should be at least `3.7` 
  - legacy commands can be found in the details
- [OPTIONAL] an OCI compatible registry (e.g. GCR or Harbor)
- a Kubernetes Cluster (better use two different clusters: one in which Landscaper runs and one that NGINX gets installed into)
- the `landscaper-cli` and `component-cli` command line tools. Their installation is described [here](https://github.com/gardener/landscapercli/blob/master/docs/installation.md) and [here](https://github.com/gardener/component-cli).

All example resources can be found in the folder [./resources/ingress-nginx](./resources/ingress-nginx) of this repository.

:warning: Note that the repository `eu.gcr.io/gardener-project/landscaper/tutorials` that is used throughout this tutorial is an example repository and has to be replaced with the path to your own registry if you want to upload your own artifacts.
If you do not have your own OCI registry, you can of course reuse the artifacts that we provided at `eu.gcr.io/gardener-project/landscaper/tutorials` which are publicly readable.

## Step 1: Prepare the NGINX helm chart

The current helm deployer only supports helm charts stored in an OCI registry. We therefore have to convert and upload the open source helm chart as an OCI artifact to our registry.

```
# add open source and nginx helm registries
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo add stable https://charts.helm.sh/stable
helm repo update

# download the nginx ingress helm chart and extract it to /tmp/nginx-ingress
helm pull ingress-nginx/ingress-nginx --version 4.0.17 --untar --destination /tmp

# upload the helm chart to an OCI registry
export OCI_REGISTRY="oci://eu.gcr.io" # <-- replace this with the URL of your own OCI registry, DO NOT FORGET the OCI protocol prefix oci://
export CHART_REF_PREFIX="$OCI_REGISTRY/chart-prefix/" # e.g. eu.gcr.io/gardener-project/landscaper/tutorials/charts
export HELM_EXPERIMENTAL_OCI=1
helm registry login -u myuser $OCI_REGISTRY
helm package /tmp/ingress-nginx -d /tmp
helm push /tmp/ingress-nginx-4.0.17.tgz $CHART_REF_PREFIX
# the helm chart is uploaded as oci artifact to $CHART_REF_PREFIX/chart-name:chart-version" e.g. eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:4.0.17
```

Expand the below details for Helm version < `3.7`.
<details>

```shell script
# add open source and nginx helm registries
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo add stable https://charts.helm.sh/stable
helm repo update

# download the nginx ingress helm chart and extract it to /tmp/nginx-ingress
helm pull ingress-nginx/ingress-nginx --version 4.0.17 --untar --destination /tmp

# upload the helm chart to an OCI registry
export OCI_REGISTRY="eu.gcr.io" # <-- replace this with the URL of your own OCI registry
export CHART_REF="$OCI_REGISTRY/mychart/reference:my-version" # e.g. eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:4.0.17
export HELM_EXPERIMENTAL_OCI=1
helm registry login -u myuser $OCI_REGISTRY
helm package /tmp/ingress-nginx $CHART_REF
helm push $CHART_REF
```

</details>

## Step 2: Define the Component Descriptor

A Component Descriptor contains references and locations to all _resources_ that are used by the Landscaper to deploy and install an application. In this example, the only resource is a `helm` chart of the nginx-ingress controller we uploaded to an OCI registry in the [previous step](#step-1-prepare-the-nginx-helm-chart).

If a Helm chart is referenced through a component descriptor, the version of the chart in the component descriptor should match the version of the chart itself (in this example 4.0.17)

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

  repositoryContexts:
    - type: ociRegistry
      baseUrl: eu.gcr.io/gardener-project/landscaper/tutorials/components

  resources:
  - type: helm
    name: ingress-nginx-chart
    version: 4.0.17
    relation: external
    access:
      type: ociRegistry
      imageReference: eu.gcr.io.gardener-project/landscaper/tutorials/charts/ingress-nginx:4.0.17
```

## Step 3: Create a Blueprint

Blueprints contain instructions on how to install a component and what is needed to perform this deployment. In blueprints, it is possible to declare `import` and `export parameters`, which in a sense describe the interface of a blueprint.
Technically, blueprints describe how `DeployItems` are created by taking data from import parameters and applying that data to templates inside of so-called `deployExecutions`. Blueprints also specify which data is exported via export parameters from the DeployItems.

For detailed documentation about Blueprints, look at [docs/usage/Blueprints.md](/docs/usage/Blueprints.md).

### Imports and Exports declaration

The `imports` are described as a list of import declarations. Each import is declared by a unique name and a type, which is either a JSON schema or a `targetType`.

<details>

Imports with the type `schema` import their data from a data object with a given JSON schema. 
```yaml
# import with type JSON schema

- name: myimport
  schema: # valid jsonschema
    type: string | object | number | ...
```
Imports with the type `targetType` are imported from the specified _Target_.
```yaml
# import from/into targetType

- name: myimport
  targetType: "" # e.g. landscaper.gardener.cloud/kubernetes-cluster
```

</details>

The nginx-ingress controller in this tutorial only needs to import a target Kubernetes cluster and a namespace in that cluster. 
This target will be used as the k8s cluster where the Helm chart is deployed to. 

The following YAML snippet declares both imports:

```yaml
imports:
- name: cluster
  targetType: landscaper.gardener.cloud/kubernetes-cluster
- name: namespace
  type: data
  schema:
    type: string    # namespace is expected to be a string
```

The declaration of `exports` works in the same way. Again, each export is declared with a unique name and a data type (again, JSON schema or `targetType`).

To use the ingress in another Blueprint, this Blueprint exports the name of the ingress class as a simple string. With this following YAML, the export parameter is declared:

```yaml
exports:
- name: ingressClass
  schema: # here comes a valid jsonschema
    type: string
```

### DeployItems

DeployItems are created from templates, which are specified in the `deployExecutions` section of a Blueprint. Each element specifies a templating step which will result in one or multiple DeployItems (returned as a list).

```yaml
- name: "unique name of the deployitem"
  type: landscaper.gardener.cloud/helm | landscaper.gardener.cloud/container | ... # deployer identifier
  dependsOn: [] # names of other deployitems that the deploy item depends on.
  config: # provider specific configuration
    apiVersion: mydeployer.landscaper.gardener.cloud/test
    kind: ProviderConfiguration
    ...
```

Note that if a deployitem depends on another deployitem, the Landscaper ensures that dependencies are executed and reconciled in the correct sequence.

The currently supported templating engines are [GoTemplate](https://golang.org/pkg/text/template/) and [Spiff](https://github.com/mandelsoft/spiff). For detailed information about the template executors, read [/docs/usage/TemplateExecutors](/docs/usage/TemplateExecutors.md).

While processing the templates, the Landscaper offers access to the `imports` and the fields of the component descriptor through the following structure:

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

Just like `deployExections` are templates for individual DeployItems, `exportExecutions` are templates for export parameters of the blueprint.
These export execution are expected to output the exports as a map of `<export name>`:`<value>`.

<details>

If a target is exported, it is expected to adhere to the following structure:

```yaml
<target export name>:
  annotations: {} # optional
  lables: {} # optional
  config:
    type: ""
    config: {}
```

To export values of DeployItems and Installations, the Landscaper provides access to them via templating imports:

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

Let's have a look at the complete Blueprint for the ingress-nginx Helm Chart:

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

imports:
- name: cluster
  type: target
  targetType: landscaper.gardener.cloud/kubernetes-cluster
- name: namespace
  type: data
  schema:
    type: string

deployExecutions:
- name: default
  type: GoTemplate
  template: |
    deployItems:
    - name: deploy
      type: landscaper.gardener.cloud/helm
      target: 
        import: cluster
      config:
        apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
        kind: ProviderConfiguration
        
        chart:
          {{ $resource := getResource .cd "name" "ingress-nginx-chart" }}
          ref: {{ $resource.access.imageReference }}
        
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
  schema: 
    type: string
```

A Blueprint is actually a directory that contains the above described Blueprint manifest as file called `blueprint.yaml`.
The directory may contain any other data that is necessary for the deployment/templating.
For an example see [./resources/ingress-nginx/blueprint](resources/ingress-nginx/blueprint).

## Step 4: Render and Validate the Blueprint locally

The Blueprint as shown above will result in a DeployItem named _deploy_ of type `landscaper.gardener.cloud/helm`, derived from a template defined in the `deployExecution` and two imports `cluster`and `namespace`. The process of producing a concrete DeployItem is called `rendering`.

To test this rendering locally in order to have a look at the resulting DeployItem, `landscaper-cli` can be used. This CLI uses the same rendering library as the landscaper-controller that runs within Kubernetes.

First, the import values that are used by the templating step need to be put into a file that will be provided to the `landscaper-cli` ( see also [docs/tutorials/resources/ingress-nginx/import-values.yaml](./resources/ingress-nginx/import-values.yaml)).

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

With this file and the Component Descriptor (e.g. [docs/tutorials/resources/ingress-nginx/component-descriptor.yaml](./resources/ingress-nginx/component-descriptor.yaml)) at hand, the Blueprint can be rendered.

```shell script
landscaper-cli blueprints render ./docs/tutorials/resources/ingress-nginx/blueprint \
  -c ./docs/tutorials/resources/ingress-nginx/component-descriptor.yaml \
  -f ./docs/tutorials/resources/ingress-nginx/import-values.yaml
```

This will result in the following DeployItem:

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
      ref: eu.gcr.io/myproject/charts/nginx-ingress:4.0.17
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

The Blueprint can easily be uploaded with the `landscaper-cli` tool. The CLI will package the Blueprint and upload it to the given OCI registry.

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

Blueprints are also just resources of a Component Descriptor. Therefore, after the Blueprint got uploaded, its reference needs to be added to the Component Descriptor. This is necessary to make sure that all resources of an application are known to the Component Descriptor - and the Blueprint is just one of the resources of an application.
In addition, the Landscaper needs this information to resolve the location of the Blueprint resource.

Note that also the repository context needs to be added to the Component Descriptor. (**ToDo**: Add more info about this here or link to respective docs ...)

```yaml
meta:
  schemaVersion: v2

component:
  name: github.com/gardener/landscaper/ingress-nginx
  version: v0.3.2

  provider: internal
  sources: []
  componentReferences: []

  respositoryContext:
  - type: ociRegistry
    baseUrl: eu.gcr.io/gardener-project/landscaper/tutorials/components

  resources:  
  - type: helm
    name: ingress-nginx-chart
    version: 4.0.17
    relation: external
    access:
      type: ociRegistry
      imageReference: eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:4.0.17
  - type: blueprint
    name: ingress-nginx-blueprint
    relation: local
    access:
      type: ociRegistry
      imageReference: eu.gcr.io/gardener-project/landscaper/tutorials/blueprints/ingress-nginx:v0.3.0
```

Finally, the Component Descriptor must be uploaded to an OCI registry. This is done with the `component-cli` tool, which is included already in the `landscaper-cli`.

```shell script

# replace the values to match your registry and file locations
landscaper-cli component-cli ca remote push <path to directory with component-descriptor.yaml>

# e.g. if you were to use the provided sample content
# (this will fail as you have no write access to gardener-project on eu.gcr.io)
landscaper-cli component-cli ca remote push ./docs/tutorials/resources/ingress-nginx
```

Once the upload succeeds, the Component Descriptor should be accessible at `eu.gcr.io/gardener-project/landscaper/tutorials/components/component-descriptors/github.com/gardener/landscaper/ingress-nginx/v0.3.2` in the registry.
(**ToDo**: This will NOT be the registry, if the user performing this tutorial created his own registry. Needs to be updated to make this more clear.)

## Step 6: Installation

Now all resources are added to the Component Descriptor, and everything is uploaded to the OCI registry. The nginx-ingress can finally be installed by the Landscaper into the target Kubernetes cluster.

For this, a working Landscaper Installation is needed. For a detailed installation instruction, see the [Landscaper Controller Installation](../gettingstarted/install-landscaper-controller.md) document.

### Defining the _Target_ that is used as import

The Blueprint that has been created above defines one import parameter of type `landscaper.gardener.cloud/kubernetes-cluster`, therefore, a Target resource of type `landscaper.gardener.cloud/kubernetes-cluster` that points to the target cluster has to be defined. This target basically needs to contain the target cluster's kubeconfig.

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

The Blueprint defines another import for the namespace that should be of type `string`. Imports that are defined by a jsonschema are called data imports. These imports can be defined either via `DataObject`, `Secret` or `ConfigMap`.

In this tutorial, we will define this import as `ConfigMap`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-imports
data:
  namespace: default
```

### Defining the _Installation_ resource

An _Installation_ is an instance of a Blueprint, i.e. it is the runtime representation of one specific Blueprint.

An installation resource needs to provide references to a Component Descriptor and a Blueprint, and concretely specify import and export data.

__Component Descriptor__: Remember that a Blueprint is just another resource of a software component and thus is referenced by the Component Descriptor. In the `installation.yaml`, we need to specify the Component Descriptor through its repository context, the component name and its version.

```yaml
componentDescriptor:
  ref:
    repositoryContext:
      type: ociRegistry
      baseUrl: eu.gcr.io/gardener-project/landscaper/tutorials/components
    componentName: github.com/gardener/landscaper/ingress-nginx
    version: v0.3.2
```

__Blueprint__: Once the Component Descriptor is known to the installation, the Blueprint artifact can be referenced by its unique name `ingress-nginx-blueprint`.

```yaml
blueprint:
  ref:
    resourceName: ingress-nginx-blueprint
```

__Imports__: The Blueprint requires a _Target_ import of type `landscaper.gardener.cloud/kubernetes-cluster` and a data import for the namespace. This means that the target `my-target-cluster` and the configmap we created before need to be connected to the Blueprints import in the Installation.

```yaml
imports:
  targets:
  - name: cluster
    target: "my-target-cluster"
  data:
  - name: namespace # name of the import in the blueprint
    configMapRef:
      key: "namespace"
      name: "my-imports" # name of the configmap;
      # namespace: default # the namespace will be defaulted to the namespace of the installation.

```

__Exports__: The nginx ingress Blueprint exports the used `ingressClass`, so that it can be reused by other components. To give the generic ingress class more semantic meaning in the current installation, the export is exported as `myIngressClass`.
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
  annotations:
    # this annotation is required such that the installation is picked up by the Landscaper
    # it will be removed when processing has started
    landscaper.gardener.cloud/operation: reconcile
  name: my-ingress
spec:
  componentDescriptor:
    ref:
      repositoryContext:
        type: ociRegistry
        baseUrl: eu.gcr.io/gardener-project/landscaper/tutorials/components
      componentName: github.com/gardener/landscaper/ingress-nginx
      version: v0.3.2

  blueprint:
    ref:
      resourceName: ingress-nginx-blueprint

  imports:
    targets:
    - name: cluster
      target: "my-target-cluster"
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

The _Target_ and the _Installation_ resources can now be applied to the Kubernetes cluster where the landscaper-controller runs.

```shell script
kubectl apply -f docs/tutorials/resources/ingress-nginx/my-target.yaml
kubectl apply -f docs/tutorials/resources/ingress-nginx/configmap.yaml
kubectl apply -f docs/tutorials/resources/ingress-nginx/installation.yaml
```

The Landscaper will immediately start to reconcile the _Installation_ as all imports are satisfied.

The first resource that will be created is the execution object, which is a helper resource that contains the rendered deployitems. The status shows the one specified Helm DeployItem, which has been automatically created by the Landscaper.

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
        ref: eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:4.0.17
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

The created _DeployItem_ will be reconciled by the Helm deployer. It is the Helm deployer that creates and updates the configured resources of the Helm chart in the target cluster. 

After the deployer successfully reconciled the DeployItem, the phase is set to `Succeeded` and all managed resources are added to the DeployItem's status.

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
      ref: eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:4.0.17
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

The Blueprint declared export parameters, and therefore, the Helm deployer creates a secret which contains the exported values.

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

The Landscaper collects the export from the execution and creates the configured exported dataobject `myIngressClass`.
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
- A component descriptor has been created that contains the blueprint and another external resources as resources.
- The blueprint and the component descriptor have been uploaded to the OCI registry.
- An installation has been defined and applied to the cluster which resulted in the deployed nginx application.

## Up Next
In the [next tutorial](./02-local-simple-blueprint.md), the same blueprint is deployed but using only local component artifacts.

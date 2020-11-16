# Develop a simple Blueprint

This tutorial describes the basic development of blueprints.
It covers the whole manual workflow from blueprint creation with a component descriptor and the usage in a remote OCI repository.

As example application a nginx ingress is deployed via its upstream helm chart.
(ref https://github.com/kubernetes/ingress-nginx/tree/master/charts/ingress-nginx)

__Prerequisites__:
- Helm commandline tool (see https://helm.sh/docs/intro/install/)
- OCI compatible oci registry (e.g. GCR or Harbor)
- Kubernetes Cluster (better use two different clusters: one for the landscaper and one for the installation)

All example resources can be found in [./resources/ingress-nginx](./resources/ingress-nginx).<br>
:warning: note that the repository `eu.gcr.io/gardener-project/landscaper/tutorials` is an example repository 
and has to be replaced with your own registry if you want to upload your own artifacts.
Although the artifacts are public readable so they can be used out-of-the-box without a need for your own oci registry.

Structure:
- [Resources](#resources)
    - [Prepare nginx helm chart](#prepare-nginx-helm-chart)
    - [Define Component Descriptor](#define-the-component-descriptor)
    - [Create Blueprint](#create-blueprint)
    - [Render and Validate](#render-and-validate-locally)
- [Remote Upload](#remote-upload)
- [Installation](#installation)
- [Summary](#summary)
- [Up next](#up-next)

### Resources

#### Prepare nginx helm chart
As the current helm deployer only supports oci charts, we have to convert and upload the open source helm chart as oci artifact.

```shell script
# add open source helm registry
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo add stable https://kubernetes-charts.storage.googleapis.com/
helm repo update

# download the helm artifact locally
helm pull ingress-nginx/ingress-nginx --untar --destination /tmp

# upload the oci artifact to a oci registry
export OCI_REGISTRY="eu.gcr.io" # replace with your own oci registry
export CHART_REF="$OCI_REGISTRY/mychart/reference:my-version" # e.g. eu.gcr.io.gardener-project/landscaper/tutorials/charts/ingress-nginx:v0.1.0
export HELM_EXPERIMENTAL_OCI=1
helm registry login -u myuser $OCI_REGISTRY
helm chart save /tmp/ingress-nginx $CHART_REF
helm chart push $CHART_REF
```

#### Define the Component Descriptor

A component descriptor contains all resources that are used by the application installation.
Resources are in this example the ingress-nginx helm chart but could also be `oci images` or even `node modules`.

For more information about the component descriptor and the usage of the different fields see the [component descriptor docs](https://github.com/gardener/component-spec).

```yaml
meta:
  schemaVersion: v2

component:
  name: github.com/gardener/landscaper/nginx-ingress
  version: v0.1.0

  provider: internal

  resources:
  - type: helm
    name: ingress-nginx-chart
    version: v0.1.0
    relation: external
    access:
      type: ociRegistry
      imageReference: eu.gcr.io.gardener-project/landscaper/tutorials/charts/ingress-nginx:v0.1.0
```

The component descriptor will be transformed into a "resolved" component descriptor by the landscaper during access.
This is done to ease the accessibility of resources inside the component descriptor with a templating language.

:warning: this might chage in the future.
```yaml
meta:
  schemaVersion: v2

component:
  name: github.com/gardener/landscaper/ingress-nginx
  version: v0.1.0

  provider: internal

  resources:
    ingress-nginx-chart:
      type: helm
      name: ingress-nginx-chart
      version: v0.1.0
      relation: external
      access:
        type: ociRegistry
        imageReference: eu.gcr.io.gardener-project/landscaper/tutorials/charts/ingress-nginx:v0.1.0
```

#### Create Blueprint

Blueprints describe the imports that are used to template the deployitems and exports that result from the executed deploy items.

For detailed documentation about the blueprint and see [docs/usage/Blueprints.md](/docs/usage/Blueprints.md).

The imports are describes as list of import definitions.
A import is defined by a unique name and a type definition.
The type definition is either a jsonschema definition or a `targetType`.

A jsonschema imports the data from a dataobject with a given jsonschema.<br>
A target with a specific type is imported if a targetType is defined.

```yaml
# jsonschema
- name: myimport
  schema: # valid jsonschema
    type: string | object | number | ...
```

```yaml
# targetType
- name: myimport
  targetType: "" # e.g. landscaper.gardener.cloud/kubernetes-cluster
```

For the example nginx application, only a kubernetes cluster target is imported.
The target will be used as the target cluster for the helm chart.

As a simple export, the ingress class of the ingress is exported as type string.

DeployItems are templated in the `deployExecutions` section by specifying different templating steps.
Each template step has to output a list of deploy item templates of the following form.
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

Currently `GoTemplate` and `Spiff` are supported templating engines.
For detailed information about the template executors see [here](/docs/usage/TemplateExecutors.md).

The landscaper offers access to the imports and the component descriptor in the following structure.
```yaml
imports:
  <import name>: <data value> or <target cr>
cd:
 component:
   resources: ...
```

Exports can be described the same way as imports.
Also exports can be templated using templating in the `exportExecutions`.
The export execution are expected to output the exports as a map of <export name>: <value> .<br>
If a target is exported the following structure is expected to be exported:
```yaml
<target export name>:
  annotations: {} # optional
  lables: {} # optional
  config:
    type: ""
    config: {}
```

In order to export values of deploy items and installations, the landscaper give access to these values via templating imports:
```yaml
values:
  deployitems:
    <deploy item name>: <deploy item export value (is type specific)>
  dataobjects:
    <data object name>: <data of the dataobject> (currently only exports of the subinstallations are accessible)
  targets:
    <target name>: <the target cr>
```

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

imports:
- name: cluster
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
          ref: {{ index .cd.component.resources "ingress-nginx-chart" "access" "imageReference" }}
        
        updateStrategy: patch
        
        name: test
        namespace: default
        
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
  schema: # here comes a valid jsonschema
    type: string
```

A blueprint is defined by a directory that contains the above described Blueprint Manifest as file called `blueprint.yaml`.
The directory can contain any other data that is necessary for the deployment/templating.
For an example see [./resources/ingress-nginx/blueprint](resources/ingress-nginx/blueprint).

##### Render and Validate locally

The blueprint will result in a deploy item of helm that is templated using one import.
This resulting deploy item can be rendered and the templating can be tested by 
1. providing some sample import as file (e.g. docs/tutorials/resources/ingress-nginx/import-values.yaml)
   ```
   imports:
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
2. and render the blueprint with the component descriptor: 
   ```
   landscaper-cli blueprints render ./docs/tutorials/resources/ingress-nginx/blueprint \
      -c ./docs/tutorials/resources/ingress-nginx/component-descriptor.yaml \
      -f ./docs/tutorials/resources/ingress-nginx/import-values.yaml
   ```
   
   ```
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
         ref: eu.gcr.io/myproject/charts/nginx-ingress:v0.1.0
       exportsFromManifests:
       - jsonPath: .Values.controller.ingressClass
         key: ingressClass
       kind: ProviderConfiguration
       name: test
       namespace: default
       updateStrategy: patch
     target:
       name: my-cluster
       namespace: <no value>
     type: landscaper.gardener.cloud/helm
   status:
     observedGeneration: 0
   ```


### Remote Upload

After the blueprint is build it has to be uploaded to the oci registry and the reference needs to be added to the component descriptor.
The blueprint can be easily uploaded by using the landscaper cli tool which packages the blueprint and uploads to the given oci registry.

To install the landscaper see [Landscaper CLI Installation](../gettingstarted/install-landscaper-cli.md)

```shell script
# landscaper-cli blueprints push myregistry/mypath/ingress-nginx:v0.1.0 docs/tutorials/resources/ingress-nginx/blueprint
landscaper-cli blueprints push eu.gcr.io/gardener-project/landscaper/tutorials/blueprints/ingress-nginx:v0.1.0 docs/tutorials/resources/ingress-nginx/blueprint
```

Blueprints are also just resources/artifacts of a component descriptor.
Therefore, after the blueprint is uploaded, the reference to that blueprint has to be added to the component descriptor.
This is done to ensure that all resources of a application are known and stored.
In addition, it is used by the landscaper to resolve the location of the blueprint resource.

Note that the repository context as well as the blueprint resource should be added to the component descriptor
```yaml
meta:
  schemaVersion: v2

component:
  name: github.com/gardener/landscaper/ingress-nginx
  version: v0.1.0

  provider: internal

  respositoryContext:
  - type: ociRegistry
    baseUrl: eu.gcr.io/my-project/comp

    
  resources:
  - type: blueprint
    name: ingress-nginx-blueprint
    relation: local
    access:
      type: ociRegistry
      imageReference: myregistry/mypath/ingress-nginx:v0.1.0
  - type: helm
    name: ingress-nginx-chart
    version: v0.1.0
    relation: external
    access:
      type: ociRegistry
      imageReference: eu.gcr.io/myproject/charts/nginx-ingress:v0.1.0
```

Then the component descriptor can be uploaded to a oci registry using again the landscaper cli.
When the upload succeeds, the component should be accessible at `eu.gcr.io/my-project/comp/github.com/gardener/landscaper/ingress-nginx/v0.1.0` in the registry.
```shell script
landscaper-cli cd push docs/tutorials/resources/ingress-nginx/component-descriptor.yaml
```

### Installation

As all external resources are defined and uploaded, the nginx ingress can be installed into the second kubernetes cluster.

Before the runtime resources are defined, the landscaper controller has to be installed into the first kubernetes cluster.
For a detailed installation instructions see [Landscaper Controller Installation](../gettingstarted/install-landscaper-controller.md).

The previously created blueprint can be installed in a target system by instructing the landscaper via a Installation resource to install it to the second cluster.

The blueprint defines one import of a kubernetes, therefore, a `Target` resource of type `landscaper.gardener.cloud/kubernetes-cluster` that points to the second cluster has to be defined.

See the example of a `Target` below.
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Target
metadata:
  name: my-cluster
spec:
  type: landscaper.gardener.cloud/kubernetes-cluster
  config:
    kubeconfig: |
      apiVersion:...
      # here goes the kubeconfig of the target cluster
```

An Installation is an instance of a blueprint, which means that it is the runtime representation of one specific blueprint installation.

The installation consists of a blueprint, imports and exports.<br>
__blueprint__:
To reference the previously uploaded Blueprint, the component descriptor is referenced by specifying the repository context, the component name and version.
With that, the landscaper is able to resolve the component descriptor from the oci registry.<br>
The Blueprint artifact in the component descriptor is specified as resources with the unique name `ingress-nginx-blueprint` 
which can be referenced with `resourceName: ingress-nginx-blueprint`
```yaml
spec:
  blueprint:
    ref:
      repositoryContext:
        type: ociRegistry
        baseUrl: eu.gcr.io/my-project/comp
      componentName: github.com/gardener/landscaper/ingress-nginx
      version: v0.1.0
```

__imports__:
The blueprint needs a target import of type kubernetes cluster.
The target `mycluster` is created as mentioned above and has to be connected to the import.
This is done by specifying the target as targets import.

:warning: The "#" has to be used to reference the previously created target. Otherwise, the landscaper would try to import the target from another component's export.
```yaml
imports:
  targets:
  - name: cluster
    # the "#" forces the landscaper to use the target with the name "my-cluster" in the same namespace
    target: "#my-cluster"
```

__exports__:

The nginx ingress blueprint export the used ingressClass so that it can be reused by other components.
To give the generic ingress class more semantic meaning in the current installation, the export is exported as `myIngressClass`.
Other installation are now able to consume the data with this specific name.

:warning: Note that this name has to be unique so that it will not be overwritten by other installations.

The export is a data object export, therefore the eyport is defined under `spec.exports.data` and is written to the `dataRef: myIngressClass`.
```yaml
exports:
  data:
  - name: ingressClass
    dataRef: "myIngressClass"
```

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: my-ingress
spec:
  blueprint:
    ref:
      repositoryContext:
        type: ociRegistry
        baseUrl: eu.gcr.io/my-project/comp
      componentName: github.com/gardener/landscaper/ingress-nginx
      version: v0.1.0
      kind: localResource
      resourceName: ingress-nginx-blueprint

  imports:
    targets:
    - name: cluster
      # the "#" forces the landscaper to use the target with the name "my-cluster" in the same namespace
      target: "#my-cluster"
  
  exports:
    data:
    - name: ingressClass
      dataRef: "myIngressClass"
```

When the `Target` and the Installation CRs are properly configured, they can be applied to the kubernetes cluster running the landscaper.

```shell script
kubectl create -f docs/tutorials/resources/ingress-nginx/my-target.yaml
kubectl create -f docs/tutorials/resources/ingress-nginx/installation.yaml
```

The landscaper will then immediately start to reconcile the installation as all imports are satisfied.

The first resources that will be created is the execution object which is a helper resource that contains the rendered deployitems.
The status shows the one specified Helm deploy item which has been automatically created by the landscaper.
```shell script
$ kubectl get inst
NAME                        PHASE       CONFIGGEN   EXECUTION                   AGE
my-ingress                  Succeeded               my-ingress                  4m11s

$ kubectl get exec my-execution -oyaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Execution
metadata:
  ...
spec:
  deployItems:
  - config:
      apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
      chart:
        ref: eu.gcr.io/myproject/charts/nginx-ingress:v0.1.0
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
status:
  ...
  deployItemRefs:
  - name: deploy
    ref:
      name: my-ingress-deploy-xxx
      namespace: default
      observedGeneration: 1
  ...
```

The newly created deploy item will be reconciled by the Helm deployer.
The helm deployer actually creates and updates the configured resources of the helm chart in the target cluster.
When the deployer successfully reconciles the deploy item, the phase is set to `Succeeded` and the all managed resources are added to the DeployItem's status.

```shell script
$ kubectl get di my-ingress-deploy-xxx -oyaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployItem
metadata:
  ...
spec:
  config:
    apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
    chart:
      ref: eu.gcr.io/myproject/charts/nginx-ingress:v0.1.0
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
    ...
```

The blueprint also configures export values.
Therefore, the helm deployer also creates a secret that contains the exported values.

```shell script
# A kubectl plugin is used to automatically decode the base64 encoded secret
$ kubectl ksd get secret my-ingress-deploy-5stgr-export -oyaml
apiVersion: v1
kind: Secret
metadata:
  ...
stringData:
  config: |
    ingressClass: nginx
type: Opaque
```

This exported value is then propagated to the execution object and then used in the `exportExecutions` to create the exports.
The execution resource combines all deployitem exports into a data object.
```shell script
$ kubectl get exec my-execution
NAME                        PHASE       EXPORTREF                          AGE
my-ingress                  Succeeded   3a4cwhagjhl5i6iu3vvljkjkzffxbk4p   5m

$ kubectl get do 3a4cwhagjhl5i6iu3vvljkjkzffxbk4p -oyaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DataObject
metadata:
  ...
data:
  deploy:
    ingressClass: nginx
```

The landscaper collects the export from the execution and creates the configured exported dataobject `myIngressClass`.
The exported dataobject is a contextified dataobject, which means that it can only be imported by other installations in the same context.
The dataobject's context is the root context `""` so that all root installations could use the export as import.

Contextified dataobjects name is a hash of the exported key and the context, so that they can be unqiely identified by the landscaper.
:warning: Note: also targets are contextified but global target/dataobjects can be referenced with a prefix `#` as in the current target import.

```shell script
$ kubectl get do -l data.landscaper.gardener.cloud/key=myIngressClass
NAME                               CONTEXT   KEY
dole6tby5kerlxruq2n2efxiql6onp3h             myIngressClass

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

### Summary
- A blueprint has been created that describes how a nginx ingress can be deployed into a kubernetes cluster.
- A component descriptor has been created that contains the blueprint and another external resources as resources.
- The blueprint and the component descriptor are uploaded to the oci registry.
- A installation has been defined and applied to the cluster which resulted in teh deployed nginx application. 

### Up Next
In the [next tutorial](./02-simple-import.md), another application is deployed that used the exported ingressClass data.

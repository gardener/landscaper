# Component Creation from Scratch

## Goals and Structure

This article aims at providing a guide for landscaper users who wish to create an own component. It follows a simple scenario: you have a helm chart and you want to wrap this helm chart in a landscaper component.

Each step below consists of up to four different kinds of information:
- **Instructions** tell you what _you_ need to do in this step to build your component
- **General Information** provides general information about what the respective configuration is used for
- **Details** provide further details, such as other accepted values for fields, the specific purpose of fields, and so on.
- **Examples** show what the configuration could look like in practice.

For a better reading flow, some of these information will be collapsed by default.





## 1 - Create Component Descriptor

### 1.1 - Metadata

###### Instructions
Choose a name and a version for your component. Within a given repository context, the combination of name and version must be unique.
The name has to be a DNS label and relaxed semver is used for versioning.

###### Example
```yaml
meta:
  schemaVersion: v2 # schema of the CD

component:
  name: github.com/gardener/landscaper/ingress-nginx # component name
  version: v0.3.1 # component version

  provider: internal # no idea
```

### 1.2 - Repository Contexts

###### General Information
`component.repositoryContexts` contains a list of repository contexts. It must contain the context of the repository where the component is uploaded to.
When copying components from one repository to another, the new context can simply be added to the list, in this case the list also serves as a kind of history.

###### Instructions
Add the base URL to the repository where you want to upload the component to. It's also possible to skip this step for now, as the component-cli tool can inject the correct `baseUrl` automatically when uploading the component descriptor to a registry.

###### Example
```yaml
  repositoryContexts:
  - type: ociRegistry # what is supported?
    baseUrl: eu.gcr.io/gardener-project/landscaper/tutorials/components
```

<details>
  <summary>Details</summary>
  
  Currently, the only supported `type` is `ociRegistry`.
  Its fields are:
  - `baseUrl`: The base URL of the repository. The component's URL will be `<base-url>/component-descriptors/<component-name>`.
</details>


### 1.3 - Resources

###### General Information
Under `component.resources`, **everything** that is needed by the component must be listed. This includes - among other things - blueprints, helm charts, and container images. A component should never 'pull' something from anywhere if it is not specified in its component descriptor. To install a component in a closed environment without internet access, all dependencies have to be copied into a registry that is reachable from within the environment. Hence the requirement that components must be self-contained.

###### Instructions
Make sure you add all required resources to the component descriptor. There are currently two possibilites of including resources: either by a reference to an OCI artifact or by bundling a local blob to the component descriptor.
Both options are described below.

<details>
  <summary>Details</summary>
  
  `type` specifies the type of the referenced resource. TODO: supported types/how is this field used?

  `name` and `version` are used to reference the resource in the context of this component descriptor. Both can differ from the referenced resource's own name/version.

  `relation` differentiates between `local` resources which are part of this component (and are expected to share its version) and `external` resources for everything else. TODO: default?
</details>

#### 1.3a - Referencing an Artifact in a Registry

The preferred way of referencing a resource is to push this resource into an OCI-compliant registry and simply reference it.

###### Example
```yaml
type: blueprint
name: ingress-nginx-blueprint
version: v0.3.1
relation: local # will be released with component / has same version (default?)
access:
  type: ociRegistry
  imageReference: eu.gcr.io/gardener-project/landscaper/tutorials/blueprints/ingress-nginx:v0.3.1
```
<details>
  <summary>Details</summary>
  
  `access` describes how the resource can be accessed. 
  Currently, only the `ociRegistry` is supported. 
  Its configuration needs an `imageReference` field which points to the path of the resource in an OCI registry.
</details>

#### 1.3b - Referencing Local Blobs

It is also possible to reference a local blob. The `component-cli` tool will pack the referenced directory into a tarball - optionally compressing it - and attach it to the component descriptor when uploading the component into the registry.

###### Example
```yaml
type: blueprint
name: blueprint
version: v0.1.0
relation: local
input: 
  type: dir
  path: ./blueprint
  mediaType: application/vnd.gardener.landscaper.blueprint.v1+tar+gzip
  compress: true
```

<details>
  <summary>Example 2</summary>
  
```yaml
type: helm
name: ingress-nginx-chart
version: v3.29.0
relation: external
input: 
  type: dir
  path: /Users/johndoe/documents/nginx/charts
  compress: true
```
</details>
<details>
  <summary>Details</summary>
  
  Instead of `access`, `input` is used for local blobs.

  `type` specifies the form of the local blob. TODO: allowed values?

  `path` is the path to the local resource. This can be a path outside of the component directory, the `component-cli` tool will take care of collecting the local blobs when uploading the component to the registry.

  `mediaType` ... TODO

  If `compress` is true, the local blob is not only packed into a tarball, but also compressed using gzip.
</details>




## Blueprint

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint
```


### DeployItems

example chart in OCI registry
```yaml
deployItems:
- name: nginx
  type: landscaper.gardener.cloud/helm
  target:
    name: {{ index .imports "target-cluster" "metadata" "name" }}
    namespace: {{ index .imports "target-cluster" "metadata" "namespace" }}
  config:
    apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderConfiguration

    chart:
      ref: {{ with (getResource .cd "name" "nginx-chart") }} {{ .access.imageReference }} {{ end }}

    updateStrategy: patch

    name: nginx
    namespace: {{ index .imports "nginx-namespace" }}
```


example local chart
```yaml
deployItems:
- name: echo
  type: landscaper.gardener.cloud/helm
  target:
    name: {{ index .imports "target-cluster" "metadata" "name" }}
    namespace: {{ index .imports "target-cluster" "metadata" "namespace" }}
  config:
    apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderConfiguration

    chart:
      fromResource: 
{{ toYaml .componentDescriptorDef | indent 8 }}
      resourceName: echo-chart

    updateStrategy: patch

    name: echo
    namespace: {{ index .imports "echo-server-namespace" }}
```
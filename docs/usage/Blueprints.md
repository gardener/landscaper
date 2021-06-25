# Blueprints

Blueprints describe the steps that are necessary to deploy a component/application. These steps can consist of deploy items or other blueprints which are assembled using installation templates.

A Blueprint is a filesystem that contains the blueprint definition at `/blueprint.yaml`. Other additional files can be added optionally.

Every Blueprint must have a corresponding component descriptor that is used to reference the Blueprint and define its the dependencies.

```
my-blueprint
├── data
│   └── myadditional data
└── blueprint.yaml
```

**Index**:
- [Blueprint](#blueprint)
  - [blueprint.yaml Definition](#blueprintyaml-definition)
    - [Import and Export Definitions](#import-and-export-definitions)
    - [DeployExecutions](#deployexecutions)
    - [ExportExecutions](#exportexecutions)
    - [Installation Templates](#installation-templates)
  - [Remote Access](#remote-access)
    - [Local](#local)
    - [OCI](#oci)

## blueprint.yaml

A Blueprint is a versioned configuration file that consists of 

- imports
- exports
- deployExecutions
- exportExecution
- subinstallation

It is defined by a YAML definition that sits on the top level inside the blueprints filesystem.

The following snippet shows the structure of a `blueprint.yaml` file. Refer to [.schemas/landscaper_Blueprint.json](../../.schemas/landscaper_Blueprint.json) for the automatically generated jsonschema definition.

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

# jsonSchemaVersion describes the default jsonschema definition 
# for the import and export definitions.
jsonSchemaVersion: "https://json-schema.org/draft/2019-09/schema"

# localTypes defines shared jsonschema types that can be used in the 
# import and export definitions.
localTypes:
  mytype: # expects a jsonschema
    type: object

# imports defines all imported values that are expected.
# Data can be either imported as data object or target.
imports:
# Import a data object by specifying the expected structure of data
# as jsonschema.
- name: my-import-key
  required: true # required, defaults to true
  type: data # this is a data import
  schema: # expects a jsonschema
    type: string
# Import a target by specifying the targetType
- name: my-target-import-key
  required: true # required, defaults to true
  type: target # this is a target import
  targetType: landscaper.gardener.cloud/kubernetes-cluster

# exports defines all values that are produced by the blueprint
# and that are exported.
# Exported values can be consumed by other blueprints.
# Data can be either exported as data object or target.
exports:
# Export a data object by specifying the expected structure of data
# as jsonschema.
- name: my-export-key
  type: data # this is a data export
  schema: # expects a jsonschema
    type: string
# Export a target by specifying the targetType
- name: my-target-export-key
  type: target # this is a target export
  targetType: landscaper.gardener.cloud/kubernetes-cluster

# deployExecutions are a templating mechanism to 
# template the deploy items.
# For detailed documentation see #DeployExecutions
deployExecutions: 
- name: execution-name
  type: GoTemplate | Spiff
  file: path to file # path is relative to the blueprint's filesystem root
  template: # inline template

# exportExecutions are a templating mechanism to 
# template the export.
# For detailed documentation see #ExportExecutions
exportExecutions:
- name: execution-name
  type: GoTemplate | Spiff
  file: path to file # path is relative to the blueprint's filesystem root
  template: # inline template

# subinstallations is a list of installation templates.
# A installation template expose specific installation configuration are 
# used to assemble multiple blueprints together.
subinstallations:
- file: /installations/dns/dns-installation.yaml
- apiVersion: landscaper.gardener.cloud/v1alpha1
  kind: InstallationTemplate
  name: ingress # must be unique
  blueprint:
    ref: cd://componentReferences/ingress/resources/blueprint #cd://resources/myblueprint
#    filesystem:
#      blueprint.yaml: abc...
  
  # define imported dataobjects and target from other installations or the 
  # parents import.
  # It's the same syntax as for default installations.
  imports:
    data:
    - name: "" # data import name
      dataRef: "" # dataobject name
    targets:
    - name: "" # target import name
      target: "" # target name
  #importMappings: {}

  exports:
    targets:
    - name: "" # target export name
      target: "" # target name
  #exportMappings: {}

```

### Import and Export Definitions

The interface of a blueprint can be described using imports and exports.<br>
The import definitions describe the data that needed by the blueprint.<br>
The export definitions describe the data that is produced by the blueprint.<br>

Imports and Exports always consists of a name (_MUST_ be unique for the blueprint) and can be generally described as _Target_-Im/Exports or _Data_-Im/Exports. The `type` field specifies which one it is.

> The `type` field is currently optional to ensure backward compatibility. This is likely to change in the future and it is strongly recommended to specify a type for each im-/export.

```yaml
imports:
- name: <string> # some unique name
  required: <boolean> # defaults to true
  type: <data|target> # type of the imported object
```

:warning: in the following data import and target import is used as synonym for import and export. The specification applies to both.

For advanced configuration options regarding import definitions, have a look at [conditional imports](./ConditionalImports.md).

#### Data Imports

Data imports are the default import type that can describe arbitrary data in json or yaml format.
[JSONSchema](https://json-schema.org/) is used to describe the structure fo the data.

It is recommended to provide a description and an example for the structure, so that users of the blueprint know what to provide (see the [json docs](http://json-schema.org/understanding-json-schema/reference/generic.html#annotations)).

```yaml
imports:
- name: my-import
  type: data
  schema:
    <jsonschema>
```

For detailed information about the jsonschema and landscaper specifics see [JSONSchema Docs](./JSONSchema.md)

Data im-/exports have the type `data`.

#### Targets

Targets are a specific type of import and contains additional information that is interpreted by deployers.
The concept of a Target is to define the environment where a deployer installs/deploys software.
This means that targets could contain additional information about that environment (e.g. that the target cluster is in a fenced environment and needs to be handled by another deployer instance).

The configuration structure of targets is defined by their type (currently the type is only for identification but later we plan to add some type registration with checks.)

```yaml
imports:
- name: my-import
  type: target
  targetType: <string> # name of the target type
```

Target im-/exports have the type `target`.

### DeployExecutions

A Blueprint's deploy executions may contain any number of template executors. 
A template executor must return a list of deploy items templates.<br>
A deploy item template exposes specific deploy item fields and will be rendered to DeployItem CRs by the landscaper.

__DeployItem Template__:
```yaml
deployItems:
- name: deploy-item-name # unique identifier of the step
  target:
    name: ""
    namespace: ""
  config:
    apiVersion: mydeployer/v1
    kind: ProviderConfiguration
    ...
```

##### Executor Imports

All template executors are given the same input data that can be used while templating.
The input consists of the imported values as well as the installations component descriptor.

For the specific documentation about the available templating engines see [Template Executors](./TemplateExecutors.md).

```yaml
imports:
  <import-name>: <import value>
cd: <component descriptor>
components: <list of all referenced component descriptors>
blueprint: <blueprint definition> # blueprint definition from the Installation
componentDescriptorDef: <component descriptor definition> # component descriptor definition from the installation
```

All list of deployitem templates of all template executors are appended to one list as they are specified in the deployExecution.

*Example*:

Input values:
```yaml
imports:
  replicas: 3
  cluster:
    apiVersion: landscaper.gardener.cloud/v1alpha1
    kind: Target
    metadata:
       name: dev-cluster
       namespace: default
    spec:
      type: landscaper.gardener.cloud/kubernetes-cluster
      config:
        kubeconfig: |
          apiVersion: ...
cd:
  meta:
    schemaVersion: v2
  component:
    name: my-component
    version: v1.0.0
    componentReferences:
    - name: abc
      componentName: my-referenced-component
      version: v1.0.0
    resources:
    - name: nginx-ingress-chart
      version: 0.30.0
      relation: external
      acccess:
        type: ociRegistry
        imageReference: nginx:0.30.0
components:
- meta: # the resolved component referenced in "cd.component.componentReferences[0]"
    schemaVersion: v2
  component:
    name: my-referenced-component
    version: v1.0.0
    resources:
    - name: ubuntu
      version: 0.18.0
      relation: external
      acccess:
        type: ociRegistry
        imageReference: ubuntu:0.18.0
blueprint:
 ref:
  #      repositoryContext:
  #        type: ociRegistry
  #        baseUrl: eu.gcr.io/myproj
  componentName: github.com/gardener/gardener
  version: v1.7.2
  resourceName: gardener
#    inline:
#      filesystem: # vfs filesystem
#        blueprint.yaml: 
#          apiVersion: landscaper.gardener.cloud/v1alpha1
#          kind: Blueprint
#          ...
```


```yaml
deployExecutions:
- name: default
  type: GoTemplate
  template: |
    deployItems:
    - name: deploy
      type: landscaper.gardener.cloud/helm
      target:
        name: {{ .imports.cluster.metadata.name }} # will resolve to "dev-cluster"
        namespace: {{ .imports.cluster.metadata.namespace  }} # will resolve to "default"
      config:
        apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
        kind: ProviderConfiguration
        
        chart:
          {{ $resource := getResource .cd "name" "nginx-ingress-chart" }}
          ref: {{ $resource.access.imageReference }} # resolves to nginx:0.30.0
        
        values:
          replicas: {{ .imports.replicas  }} # will resolve to 3
          
          {{ $component := getComponent .cd "name" "my-referenced-component" }} # get a component that is referenced
          {{ $resource := getResource $component "name" "ubuntu" }}
          usesImage: {{ $resource.access.imageReference }} # resolves to ubuntu:0.18.0
```

### ExportExecutions

A Blueprint's export executions may contain any number of template executors. 
A template executor must return a map of `export name` to `exported value`.<br>
Multiple template executor exports will be merged in the defined order, whereas the later defined values overwrites previous templates.

__exports__:
```yaml
exports:
  export-name: export-value
  target-export-name:
    labels: {}
    annotations: {}
    type: "" # target type
    config: any # target specific config data
```

All template executors are given the same input data that can be used while templating.
The input consists of the deploy items export values and all exports of subinstallations.

For the specific documentation about the available templating engines see [Template Executors](./TemplateExecutors.md).

```yaml
values:
  deployitems:
    <deployitem step name>: <exported values>
  dataobjects:
      <databject name>: <data object value>
  targets:
        <target name>: <whole target>
```

All list of deployitem templates of all template executors are appended to one list as they are specified in the deployExecution.

*Example*:

Input values:
```yaml
values:
  deployitems:
    deploy:
      ingressPrefix: my-pref
  dataobjects:
     domain: example.com
  targets:
    dev-cluster:
        apiVersion: landscaper.gardener.cloud/v1alpha1
        kind: Target
        metadata:
           name: dev-cluster
           namespace: default
        spec:
          type: landscaper.gardener.cloud/kubernetes-cluster
          config:
            kubeconfig: |
              apiVersion: ...
```

```yaml
exportExecutions:
- name: default
  type: GoTemplate
  template: |
    exports:
      url: http://{{ .values.deployitems.ingressPrefix  }}.{{ .values.dataobjects.domain }} # resolves to http://my-pref.example.com
      cluster:
        type: {{ .values.targets.dev-cluster.spec.type  }}
        config: {{ .values.targets.dev-cluster.spec.config  }}
```

### Installation Templates
Installation Templates are used to include subinstallation in a blueprint.
As the name suggest, they are templates for installation which means that the landscaper will create installation based on these templates.

These subinstallations have a context that is defined by the parent installation.
Context means that subinstallations can only import data that is also imported by the parent or exported by other subinstallations with the same parent.

Installation templates offer the same configuration as real installation 
except that blueprints have to be defined in the component descriptor of the blueprint (either as resource or by a component reference).
Inline blueprints are also possible.

Subinstallations can also be defined in a separate file or templated via executor (templated executors are defined in a separate field `.subinstallationExecutions`).
If defined by file it is expected that that file contains one InstallationTemplate.

All possible options to define a subinstallation can be used in parallel and are summed up.

```yaml
- apiVersion: landscaper.gardener.cloud/v1alpha1
  kind: InstallationTemplate
  name: my-subinstallation # must be unique
  blueprint:
    ref: cd://componentReferences/ingress/resources/blueprint #cd://resources/myblueprint
#    filesystem:
#      blueprint.yaml: abc...
  
  # define imported dataobjects and target from other installations or the 
  # parents import.
  # It's the same syntax as for default installations.
  imports:
    data:
    - name: "" # data import name
      dataRef: "" # dataobject name
    targets:
    - name: "" # target import name
      target: "" # target name
  #importMappings: {}

  exports:
    targets:
    - name: "" # target export name
      target: "" # target name
  #exportMappings: {}
```

Similar to how deploy items can be defined, it is also possible to create template subinstallations based on the imports.
A Blueprint's subinstallations executions may contain any number of template executors.
A template executor must return a list of installation templates.<br>

For a list of available templating imports see the [deploy item executor docs](#executor-imports).

__Subinstallation Template__:
```yaml
subinstallationExecutions:
- name: default
  type: GoTemplate
  template: |
    subinstallations:
    - apiVersion: landscaper.gardener.cloud/v1alpha1
      kind: InstallationTemplate
      name: my-subinstallation # must be unique
      blueprint:
        ref: cd://componentReferences/ingress/resources/blueprint
      ...
```


## Remote Access

Blueprints are referenced in installations or installation templates via the component descriptors access.

Basically blueprints are a filesystem, therefore, any blob store could be supported.<br>
Currently, local and OCI registry access is supported.

:warning: Be aware that a local reigstry should be only used for testing and development, whereas the OCI registry is the preferred productive method.


### Local

A local registry can be defined in the landscaper configuration by providing the below configuration.
The landscaper expects the given paths to be a directory that contains the definitions in subdirectory.
The subdirectory should contain the file `description.yaml`, that contains the actual ComponentDefinition with its version and name.
The whole subdirectory is used as the blob content of the Component.
```yaml
apiVersion: config.landscaper.gardener.cloud/v1alpha1
kind: LandscaperConfiguration

registry:
  local:
    paths:
    - "/path/to/definitions"
```

The blueprints are referenced via `local` access type in the component descriptor.
```
component:
  localResource:
  - name: blueprint
    type: blueprint
    access:
      type: local
```

### OCI

ComponentDefinitions can be stored in a OCI compliant registry which is the preferred way to create and offer ComponentDefinitions.
The Landscaper uses [OCI Artifacts](https://github.com/opencontainers/artifacts) which means that a OCI compliant registry has to be used.
For more information about the [OCI distribution spec](https://github.com/opencontainers/distribution-spec/blob/master/spec.md) and OCI compliant registries refer to the official documents.

The OCI manifest is stored in the below format in the registry.
Whereas the config is ignored and there must be exactly one layer with the containing a bluprints filesystem as `application/tar+gzip`.
 
 The layers can be identified via their title annotation or via their media type as only one component descriptor per layer is allowed.
```json
{
   "schemaVersion": 2,
   "annotations": {},
   "config": {},
   "layers": [
      {
         "digest": "sha256:6f4e69a5ff18d92e7315e3ee31c62165ebf25bfa05cad05c0d09d8f412dae401",
         "mediaType": "application/tar+gzip",
         "size": 78343,
         "annotations": {
            "org.opencontainers.image.title": "definition"
         }
      }
   ]
}
```

The blueprints are referenced via `ociRegistry` access type in the component descriptor.
```
component:
  localResource:
  - name: blueprint
    type: blueprint
    access:
      type: ociRegistry
      imgageReference: oci-ref:1.0.0
```

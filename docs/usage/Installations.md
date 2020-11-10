# Installations

Installations are kubernetes resources that represent instances of [Blueprints](./Blueprints.md).
Each installation contains the state of its executed blueprint and has its own context.

**Index**
- [Structure](#basic-structure)
- [Blueprint](#blueprint)
- [Imports](#imports)
  - [Data Imports](#data-imports)
  - [Target Imports](#target-imports)
  - [Data Mappings](#import-data-mappings)
- [Exports](#exports)
  - [Data Export](#data-exports)
  - [Target Export](#target-exports)
  - [Data Mappings](#export-data-mappings)
- [Operations](#operations)

#### Basic structure:
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: my-installation
spec:

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
  
  imports:
    data:
    - name: "" # logical internal name
      dataRef: "" # reference a contextified data object or a global dataobject with a '#' prefix.
#      secretRef: # reference a secret
#        name: ""
#        namespace: ""
#        key: ""
#      configMapRef: # reference a configmap
#        name: ""
#        namespace: ""
#        key: ""
    targets:
    - name: "" # logical internal name
      target: "" # reference a contextified target or a global taret with a '#' prefix.

  # defaulted from blueprints whereas the logical internal name is mapped to the 
  # blueprints import name
  # Note: only spiff templating is supported in this context
  importDataMappings: 
    name: value
  
  # defaulted from blueprints whereas the logical internal name is mapped to the 
  # blueprints export name
  # Note: only spiff templating is supported in this context
  exportDataMappings:
    name: value

  exports:
    data:
    - name: "" # logical internal name
      dataRef: "" # reference a contextified data object or a global dataobject with a '#' prefix.
#      secretRef: # reference a secret
#        name: ""
#        namespace: ""
#        key: ""
#      configMapRef: # reference a configmap
#        name: ""
#        namespace: ""
#        key: ""
    targets:
    - name: "" # logical internal name
      target: "" # reference a contextified target or a global taret with a '#' prefix.

status:
  phase: Progressing | Pending | Deleting | Completed | Failed

  imports:
  - name: "" # logical internal name
    type: dataobject | target
    dataRef: ""
    configGeneration: 0
  
  # Reference to the execution of the installation which is templated
  # based on the ComponentDefinition (.spec.definitionRef).
  executionRef:  
    name: my-execution
    namespace: default
  
  # References to subinstallations that were automatically created 
  # based on the ComponentDefinition (.spec.definitionRef).
  installationRefs: 
  - name: my-sub-component
    ref:
      name: my-sub-component
      namespace: default
```

## Blueprint

An Installation is an instance of a blueprint therefore every installation must reference a [blueprint](./Blueprints.md).

A blueprint can be referenced in an installation via a remote reference or inline.

__Remote Reference__

Like any other artifact, a blueprint can be a resource of a component.
Therefore, it can be defined in a component descriptor.

The landscaper uses the component descriptor's resource definitions and enhances it with another resource of type `landscaper.gardener.cloud/blueprint`(alternatively `blueprint`).
This resource definition is then used to reference the remote blueprint for the Installation.

Given a component descriptor that defines a blueprint as resource:
```yaml
meta:
  schemaVersion: v2

component:
  name: github.com/my-comp
  version: v0.0.1
  
  resources:
  - name: my-application
    type: blueprint
    relation: external
    access:
      type: ociRegistry
      imageReference: registry.example.com/blueprints/my-application
```

The defined blueprint in the component descriptor can be referenced as described in the below example in `.spec.blueprint.ref`.<br>
The component descriptor is resolved within a defined repository context as described in the [component descriptor spec](https://gardener.github.io/component-spec/component_descriptor_registries.html).
This repository context is optional and can be defaulted in the landscaper deployment.
```yaml
spec:
  blueprint:
    ref:
#      repositoryContext:
#        type: ociRegistry
#        baseUrl: ""
      componentName: github.com/my-comp
      version: v0.0.1
      resourceName: my-application
```

__Inline Blueprint__

In addition to a remote reference, a blueprint can also be defined inline directly in an installation's spec.

A blueprint is a filesystem that contains a blueprint definition file at its root.
Therefore, it must be possible to define such a filesystem within the installation manifest.
The landscaper uses the [vfs yaml filesystem definition](https://pkg.go.dev/github.com/mandelsoft/vfs/pkg/yamlfs) to define such a filesystem.

A component descriptor can be optionally referenced in `spec.blueprint.inline.cdRef`.
Currently, it is only possible to use a remote component descriptor.

```yaml
spec:
  blueprint:
    inline:
#      cdRef:
#        repositoryContext:
#          type: ociRegistry
#          baseUrl: ""
#        componentName: github.com/my-comp
#        version: v0.0.1
      filesystem:
        blueprint.yaml:
          apiVersion: landscaper.gardener.cloud/v1alpha1
          kind: Blueprint
          ...
```

## Imports

Imports define the data that should be used by the installation to satisfy the imports of the referenced blueprint.

There are two basic types of imports:

1. Data imports which are used to satisfy blueprint imports defined by a schema.
   These kind of imports can also be mapped/transformed in the installation.
2. Target imports which are used to satisfy blueprint imports defined by a target type.
   The target types must match and cannot be mapped/transformed by the installation.
   
### Data Imports

Data imports are defined by a logical name and reference to the data.

The logical name must be unique within all imports (data and targets).
If this name matches an import name in the blueprint, the imported value is directly used in the blueprint.
If an import has to be modified see [import data mapping](#import-data-mappings).

By default dataobjects are used but for usability users can directly specify secrets or configmaps as data imports.<br>
Dataobjects are the internal format of the landscaper for its data flow, therefore they are [contextified](../concepts/Context.md) by default and can also be referenced directly by their name. 
To reference a dataobject directly, prefix the `dataRef` with `#`.

```yaml
imports:
  data:
  - name: my-import
    dataRef: ""
  - name: my-import
    secretRef: 
      name: ""
      namespace: ""
      key: ""
  - name: my-import
    configMapRef: 
      name: ""
      namespace: ""
      key: ""
```

### Target Imports

Target imports are defined by a unique logical name and a reference to the actual target.

The logical name must be unique within all imports (data and targets) and must match the import name in the blueprint.<br>
The `target` attribute defines the reference to a real target within the same namespace.
By default the target name is [contextified](../concepts/Context.md) but 
a target can also be directly referenced by its name by prefixing the `target` with a `#`.

```yaml
imports:
  targets:
  - name: my-target
    target: ""
```


### Import Data Mappings

It can happen that imported data is of a different format than the expected schema defined in the blueprint.<br>
One possible solution is to add an additional blueprint that transforms the data.
As this approach would result in a big overhead for just transforming some data. 
It should be possible to easily transform imported data into imports of blueprints.

This transformation can be done in `spec.importDataMappings`.
`ImportDataMappings` define a map of all imports of a blueprint that can be templated using [spiff](https://github.com/mandelsoft/spiff). <br>
These mappings open up the following possibilities:
- combine multiple imports into one import structure
- use hard-coded values for imports
- use only parts of imports

All imported values can be used in the templating by their logical internal names.
All blueprint import data mappings are optional, by default the logical internal name is matched to the blueprint import.
```yaml
spec:
  imports:
    data:
    - name: imp1
    - name: imp2
  importDataMappings:
    <blueprintimport1>: hardcoded value
    <blueprintimport2>: (( imp1.subkey ))
    <blueprintimport3>: 
      a: (( imp1 ))
      b: (( imp2 ))
```

__Example__
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint
imports:
- name: providers
  schema:
    type: array
    items:
      type: string
- name: identifier
  schema:
    type: string
- name: aws-credentials
  schema:
    type: object
    properties:
      accessKeyID:
        type: string
      accessKeySecret:
        type: string
```

```yaml
spec:
  imports:
    data:
    - name: aws-provider #  value: { "type": "aws", "creds": { "accessKeyID": "adfa", "accessKeySec": "1234" } } }
    - name: gcp-provider-type #  value: "gcp"
  importDataMappings:
    identifier: my-controller
    providers:
    - (( aws-provider-type.type ))
    - (( gcp-provider-type ))
    aws-credentials: 
      accessKeyID: (( aws-provider-type.creds.accessKeyID ))
      accessKeySecret: (( aws-provider-type.creds.accessKeySec ))
```


## Exports

Exports define the data that is created by the installation.

There are two basic types of exports:

1. Data exports that result in data objects.
   These kind of exports can also be mapped/transformed in the installation.
2. Target exports that result in targets.
   The target types much match and cannot be mapped/transformed by the installation.
   
### Data Exports

Data exports are defined by a logical name and reference to the data.

The logical name must be unique within all exports (data and targets).
If this name matches a blueprint export, the exported value is directly used.
If an export has to be modified see [export data mapping](#export-data-mappings).

Exported data will always result in contextified data objects.
Export to secrets or configmaps are not possible.

```yaml
exports:
  data:
  - name: my-target
    dataRef: "my-exported-data"
```

will result in
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DataObject
metadata:
  name: <some hash>
  labels:
    data.landscaper.gardener.cloud/context: Installation.<namespace>.<installation name>
    data.landscaper.gardener.cloud/key: my-exported-data
    data.landscaper.gardener.cloud/source: Installation.<namespace>.<installation name>
    data.landscaper.gardener.cloud/sourceType: export
data: <exported data>
```

### Target Exports

Target exports are defined by a unique logical name and a reference to the actual target.

The logical name must be unique within all exports (data and targets) and must match the export name in the blueprint.<br>
The `target` attribute defines the name of contextified target that is created within the same namespace.

```yaml
exports:
  targets:
  - name: my-target
    target: "my-exported-target"
```

will result in
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Target
metadata:
  name: <some hash>
  labels:
    data.landscaper.gardener.cloud/context: Installation.<namespace>.<installation name>
    data.landscaper.gardener.cloud/key: my-exported-target
    data.landscaper.gardener.cloud/source: Installation.<namespace>.<installation name>
    data.landscaper.gardener.cloud/sourceType: export
spec:
  type: my-type
  config: <exported target data>
```

### Export Data Mappings

It can happen that exported data is of a different format than what is needed in the context.<br>
One possible solution is to add an additional blueprint that transforms the data.
As this approach would result in a big overhead for just transforming some data, an additional method is needed to transform that data.

This transformation can be done in `spec.exportDataMappings`.
`ExportDataMappings` define a map of all exports of a installation that can be templated using [spiff](https://github.com/mandelsoft/spiff). <br>
These mappings make it possible to 
- create more exports from one or multiple exports of a blueprint
- combine multiple exports to one
- export hard coded values

All exported values can be accessed in the templating by their logical internal names.
All blueprint export data mappings are optional, by default the logical internal name is matched to the blueprint export.
```yaml
spec:
  exports:
    data:
    - name: imp1
    - name: imp2
  exportDataMappings:
    imp1: hardcoded value
    imp2: (( exp1.subkey ))
```

__Example__
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint
exports:
- name: identifier
  schema:
    type: string
- name: aws-credentials
  schema:
    type: object
    properties:
      accessKeyID:
        type: string
      accessKeySecret:
        type: string
- name: gcp-credentials
  schema:
    type: object
    properties:
      serviceaccount.yaml:
        type: string
```

```yaml
spec:
  exports:
    data:
    - name: identifier
    - name: creds 
  exportDataMappings:
    identifier: (( identifier ))
    creds:
    - type: aws
      creds: (( aws-credentials ))
    - type: gcp
      creds: (( gcp-credentials ))
```

## Operations

The behavior of an installation is set by using operation annotations.
These annotations are set automatically by the landscaper as part of the default reconciliation loop.
An operator can also set annotations manually to enforce a specific behavior.

`landscaper.gardener.cloud/operation`:
  - `reconcile`: start a default reconcile on the installation
  - `force-reconcile`: skip the reconcile/pending check and directly start a new reconcilition flow. 
    - :warning: Imports still have to be satisfied.
  - `abort`: abort the current run which will abort all subinstallation but will wait until all current running components have finished.
 
`landscaper.gardener.cloud/skip=true`: skips the reconciliation of a component which means that it will not be triggered by configuration or import change.

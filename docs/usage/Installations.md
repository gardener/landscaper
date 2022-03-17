# Installations

_Installations_ are Kubernetes resources that represent concrete instantiations of
[_Blueprints_](./Blueprints.md). The task of an _Installation_ is to provide
dedicated values for the imports of the referenced _Blueprint_ and to forward
values provided for the exports of the _Blueprint_ to _DataObjects_ and _Targets_
into the scope it is livining in. Additionally the installation contains the
state of its executed blueprint.

The import values can be taken from _DataObjects_, _Targets_, _ConfigMaps_ or 
_Secrets_found in the scope of the _Installation_.


**Index**
- [Installations](#installations)
  - [Basic Structure](#basic-structure)
  - [Context](#context) 
  - [Component Descriptor](#component-descriptor) 
  - [Blueprint](#blueprint)
  - [Scopes](#scopes)
  - [Imports](#imports)
    - [Data Imports](#data-imports)
    - [Target Imports](#target-imports)
    - [Component Descriptor Imports](#component-descriptor-imports)
    - [Import Data Mappings](#import-data-mappings)
  - [Exports](#exports)
    - [Data Exports](#data-exports)
    - [Target Exports](#target-exports)
    - [Export Data Mappings](#export-data-mappings)
  - [Operations](#operations)

## Basic Structure

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: my-installation
spec:
  
  context: "" # defaults to "default"
  
  componentDescriptor:
    ref:
#      repositoryContext: # overwrite the context defined repository context.
#        type: ociRegistry
#        baseUrl: eu.gcr.io/myproj
      componentName: github.com/gardener/gardener
      version: v1.7.2
#    inline:    # https://gardener.github.io/component-spec/component-descriptor-v2.html
#      meta:
#        schemaVersion: v2
#      component:
#        name: github.com/gardener/gardener
#        version: v.1.7.2
#        ...

  blueprint:
    ref:
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
      target: "" # reference a contextified target or a global target with a '#' prefix.
    - name: ""
      targets: # reference multiple targets by name (either contextified or with a '#' prefix)
      - "target1"
      - "target2"

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

## Context

The context is a configuration resource containing shared configuration for installations.
This config can contain the repository context, registry pull secrets, etc. . 
For detailed documentation see [./Context.md](./Context.md).

Installations may reference a context by its name in `.spec.context`.
If the context reference is not defined, it is defaulted to the `default` context.
The context must be present in the same namespace as the installation.
> Note: Cross-namespace consumption is not possible.

The context is automatically passed to subinstallations and deploy items.

```yaml
spec:
  # reference to the Context resource in the same namespace
  context: "my-context"
```

## Component Descriptor

A component descriptor defines a 'component' with all its resources and dependencies.
An installation may use this information to render items with artefact locations
applicable to the actual installation environment. A component
descriptor can be located via a component version in the [context](#context)
repository or declared inline (only for development purposes).

Though technically a component descriptor is optional, most installations will
use it to manage their dependencies and access resources described in the 
component descriptor.

The component descriptor to use for an _Installation_ is specified by the spec
field `component-descriptor`.

### Reference by Component Version

A component descriptor can be identified by its name and version. With this identity 
it is resolved within a defined repository context as described in the [component descriptor spec](https://gardener.github.io/component-spec/component_descriptor_registries.html).
This repository context SHOULD be defined by the [Context](./Context.md) that is referenced by the installation.
Optionally the repository context can explicitly be overwritten in the reference.

For the reference version for specifying the component descriptor the field `ref`
is used. It supports the following fields:

- **`repositoryContext`** *optional*

  This optional field can be used to override the repository context
  specified by the actually used context. It uses the following fields:

  - **`type`** *string*<br/>
    The type of the repository. (typically the type `ociRegistry` is used here)
  
  - **`baseURL`** *string*<br/>
    Additional fields specify the access to the respository. They depend on the
    type of the repository. For an oci registry a `baseURL` must be specified.


- **`componentName`** *string*

  The name of the component of the component descriptor


- **`version`** *string*

  The version of he component descriptor


**Example**
```yaml
spec:
  
  context: "default" # optional repository context defined in the referenced context resource.
  
  componentDescriptor:
    ref: 
#      repositoryContext:
#        type: ociRegistry
#        baseUrl: ""
      componentName: github.com/my-comp
      version: v0.0.1
```

### Inline Component Descriptor

For a local development or test scenario, the landscaper allows to specify a
component descriptor directly inline within the installation. For the inline
version the field `inline` is used. It directly contains the structure of
the component descriptor.

**Example**
```yaml
spec:
  componentDescriptor:
    inline:
      meta:
        schemaVersion: v2
      component:
        name: github.com/my-comp
        version: v0.0.1
        provider: internal
        repositoryContexts:
        - type: ociRegistry
          baseUrl: "registry.example.com/test"
        sources: []
        componentReferences: []
        resources:
          - type: ociImage
            name: echo-server-image
            version: v0.2.3
            relation: external
            access:
              type: ociRegistry
              imageReference: hashicorp/http-echo:0.2.3
```

When resolving the component descriptor an inline definition takes precedence
even though a similar component descriptor may exist at the given location. 

It is important to keep in mind that only the component descriptor is inline.
To resolve any resource defined by the descriptor the given specified location
will be used. In the example above, the inline component descriptor points to
an OCI image stored remotely. If it does not exist, subsequent steps could
potentially fail.

To allow component references to be resolved locally, inline component descriptors
may be nested by adding a label to the component reference with the name
`landscaper.gardener.cloud/component-descriptor` and the actual
component descriptor as value:

```yaml
componentReferences:
  - name: ingress
    componentName: github.com/gardener/landscaper/ingress-nginx
    version: v0.2.0
    labels:
    - name: landscaper.gardener.cloud/component-descriptor
      value:
        meta:
          schemaVersion: v2
        component:
          name: github.com/gardener/landscaper/ingress-nginx
          version: v0.2.0
```

## Blueprint

An Installation described the instantiation of a blueprint, therefore, every
installation must reference a [blueprint](./Blueprints.md).

A blueprint can be referenced in an installation via a reference or inline.
It is specified in the spec field `blueprint`.

###  Reference

Like any other artifact, a blueprint can be a resource of a component. Therefore,
it can be described in a component descriptor.

The landscaper uses the component descriptor's resource definitions and enhances
it with another resource of type `landscaper.gardener.cloud/blueprint`
(alternatively `blueprint`).
This resource definition is then used to reference the remote blueprint for the
Installation.

To use a reference to a resource the blueprint specification must
contain the field `ref.resourceName`.

**Example**

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

After having referenced the component descriptor, the defined blueprint can be resolved via its name as described in the below example in `.spec.blueprint.ref.resourceName`.<br>

```yaml
spec:
  componentDescriptor:
    ref: 
#      repositoryContext:
#        type: ociRegistry
#        baseUrl: ""
      componentName: github.com/my-comp
      version: v0.0.1

  blueprint:
    ref:
      resourceName: my-application
```

### Inline Blueprint

In addition to a reference, a blueprint can also be defined inline directly in
an installation's spec.

A blueprint is a filesystem that contains a blueprint definition file at its root.
Therefore, it must be possible to define such a filesystem within the installation manifest.
The landscaper uses the [vfs yaml filesystem definition](https://pkg.go.dev/github.com/mandelsoft/vfs/pkg/yamlfs)
to define such a filesystem.

A remote or inline component descriptor can be referenced optionally in `spec.componentDescriptor`.

To use an inline blueprint, the blueprint specification must
contain the field `inline.filesystem`.

```yaml
spec:
  blueprint:
    inline:
      filesystem:
        blueprint.yaml: |
          apiVersion: landscaper.gardener.cloud/v1alpha1
          kind: Blueprint
          ...
```

## Scopes

An Installation always lives in a dedicated _Scope_. The scope is given
by the creation context of an installation. Root installations 
are explicitly created _Installations_ in a namespace of the _Landscaper_
data plane. They live in the root scope, their parent scope is the root scope.
Every installation implicitly defines a new local scope for installations
created by the referenced blueprint as [nested installations](./Blueprints.md#nested-installations).

The objects an installation can use are always restricted to the scope the
installation live in (its parent scope). This means that _DataObjects_ and
_Targets_ live are scoped, also.

**Example**
```
namespace (root scope)
├── configmap              [ConfigMap]
├── config                 [DataObject]
├── cluster                [Target]
├── application            [Installation]
│   ├── database       [Installation]
│   ├── databaseaccess [DataObject]
│   ├── webui          [Installation]
│   └── uiaccess       [DataObject]
└── exports               [DataObject]
```

The root installation (`application`) can import the _ConfigMap_ `configmap`,
the _DataObject_ `config` and the _Target_  `cluster`. 
It refers to an aggregated blueprint that creates two nested _Installations_,
`database` and `webui`. Both live in the scope of their _Installation_ (`application`).
The database installation exports a _DataObject_ `databaseaccess`. This lives
in this nested scope, also, and can only be consumed by the second nested
_Installation_ `webui`, which exposes its ui access information in the
_DataObject_ `uiaccess`.
The information provided by the nested _Installations_ and _DataObjects_ are
then aggregated by the _Blueprint_ to its exports, which will then be
stored in the _DataObject_ `exports` in the root scope to be accessible
by other top-level installations.

This kind of scoping enables the usage of local names in blueprints
without the problem of name collisions, if the same blueprint is instantiated
more than once in the same scope (typically the root scope).

So, it is easily possible to add a second installation of the application
within the same _Landscaper_ and namespace:

```
namespace (root scope)
├── configmap              [ConfigMap]
├── config                 [DataObject]
├── cluster                [Target]
├── application            [Installation]
│   ├── database       [Installation]
│   ├── databaseaccess [DataObject]
│   ├── webui          [Installation]
│   └── uiaccess       [DataObject]
├── exports                [DataObject]
│
├── config2                [DataObject]
├── cluster2               [Target]
├── application2           [Installation]
│   ├── database       [Installation]
│   ├── databaseaccess [DataObject]
│   ├── webui          [Installation]
│   └── uiaccess       [DataObject]
└── exports2               [DataObject]
```

The second installation is put into a second target (`cluster2`) and
in the scope of the second installation the same nameing structure can be
used with out collisions.

## Imports

Imports define the data that should be used by the installation to satisfy the
imports of the referenced blueprint.

Using the spec field `imports` it is possible to import data for the installation
from various sources. This data must then be used to satisfy the imports
of the referenced blueprint. By default, there is a matching mechanism in place
that matches the imports of an installation directly with the imports of
the blueprint by the used name.

This default mapping requires the imported data to directly match the data
structure requested by the imports of blueprint. Because this must not necessarily
be the case under all circumstances it is possible to define an explicit mapping
of data from tthe installation imports to the blueprint imports. This
is done by [import data mappings](#import-data-mappings).

For both purposes (the default mapping by name and the explicit data mapping),
every installation import features a `name` attribute, that must be unique 
for all imports, regardless of their types.

There are several types of imports for an installation:
- **[Data imports](#data-imports)** which are used to satisfy blueprint imports defined by a schema.
   These kind of imports can also be mapped/transformed in the installation. 
- **[Target imports](#target-imports)** must match and cannot be mapped/transformed by the installation.
- **[Component Descriptor imports](#component-descriptor-imports)** must match and cannot be mapped/transformed by the installation.

### Data Imports

Data imports are grouped in a `data` sub-section of the `imports` specification.
They are defined by the following fields:

- **`name`** *string*

  The name of the imports used for the implicit or explicit mapping to the blueprint imports.


- **`dataRef`** *string (optional)*

  This field can be used to import the data provided by a _DataObject_ with the given
  name in the scope the installation is living in.

  Exactly one of `dataRef`, `confimapRef` or `secretRef` must be given.

- **`secretRef`** *struct (optional)*

  This field can be used to import the data provided by a Kubernetes _Secret_ with the given
  name.

  Exactly one of `dataRef`, `confimapRef` or `secretRef` must be given.

  The reference field supports the following fields:

  - **`name`** *string*<br/>
    The name of the _Secret_.
  
  - **`namespace`** *string (optional)*<br/>
    The namespace of the _Secret_

  - **`key`** *string (optional)*<br/>
    The key of the secret field to use. If the key is not given, the complete
    field set of the secret is imported. The base64 encoding of the values is removed.

- **`configMapRef`** *struct (optional)*

  This field can be used to import the data provided by a Kubernetes _ConfigMap_ with the given
  name.

  Exactly one of `dataRef`, `confimapRef` or `secretRef` must be given.

  The reference field supports the following fields:

  - **`name`** *string*<br/>
    The name of the _ConfigMap_

  - **`namespace`** *string (optional)*<br/>
    The namespace of the _ConfigMap_.

  - **`key`** *string (optional)*<br/>
    The key of the configmap field to use. If the key is not given, the complete
    field set of the configmap is imported.

  
_DataObjects_ are the internal format of the landscaper for its data flow,
therefore they are [scoped](#scopes) by default and can also be referenced directly
by their name. To reference a _DataObject_ by its global name directly,
prefix the `dataRef` with `#` (deprecated). This way it is possible for nested 
installations to access global data. Because this violates the contract of
the blueprint, this feature is deprecated.


**Example**
```yaml
imports:
  data:
  - name: dataobject
    dataRef: "config"
  - name: secret
    secretRef: 
      name: "my-secret"
      namespace: "" #  optional, defaulted to installation namespace
      key: "" # optional
  - name: configmap
    configMapRef: 
      name: "my-configmap"
      namespace: "" #  optional, defaulted to installation namespace
      key: "" # optional
```

Imported data may be subject to [data import mappings](#import-data-mappings).

### Target Imports

Target imports are grouped in a `targets` sub-section of the `imports` specification.
They are defined by the following fields:

- **`name`** *string*

  The name of the imports used for the implicit or explicit mapping to the blueprint imports.


- **`target`** *string (optional)*

  This field can be used to specify the name of the _Target_ object in the scope
  the installation is living in.

  Exactly one of `target` or `targetList` must be given

- **`targetList`** *string list (optional)*

  This field can be used to specify a target list, that can match a [targetlist import](./Blueprints.md#import-definitions) 
  of a blueprint. The value is a list of the names of the _Target_ objects with the given
  name in the scope the installation is living in.

  Exactly one of `target` or `targetList` must be given. `targets: []` counts as
  specifying the `targets` field - an empty list is a valid value - while setting
  it to nil (`targets: ~`) counts as not specifying it.


_Target_ and _TargetList_ imports must directly match the required target imports of the used blueprint.
An explicit mapping is not possible.

**Example**
```yaml
imports:
  targets:
  - name: my-target
    target: "target1"

  - name: my-targetlist
    targets:
    - "target1"
    - "target2"
```


### Component Descriptor Imports

Component descriptor imports are grouped in a `componentDescriptors` sub-section of the `imports` specification.
They are defined by the following fields:

- **`name`** *string*

  The name of the imports used for the implicit or explicit mapping to the blueprint imports.


- **`ref`** *string (optional)*

  This field can be used to specify the [reference](#reference-by-component-version) to a component decsriptor.

  Exactly one of `ref`, `confimapRef`, `secretRef` or `list` must be given.

- **`secretRef`** *struct*

  This field can be used to import the component descriptor provided by a Kubernetes _Secret_ with the given
  name.

  Exactly one of `ref`, `confimapRef`, `secretRef` or `list` must be given.

  The reference field supports the following fields:

    - **`name`** *string*<br/>
      The name of the _Secret_.

    - **`namespace`** *string (optional)*<br/>
      The namespace of the _Secret_

    - **`key`** *string*<br/>
      The key of the secret field to use. The base64 encoding of the values is removed.

- **`configMapRef`** *struct*

  This field can be used to import the component descriptor provided by a Kubernetes _ConfigMap_ with the given
  name.

  Exactly one of `ref`, `confimapRef`, `secretRef` or `list` must be given.

  The reference field supports the following fields:

    - **`name`** *string*<br/>
      The name of the _ConfigMap_

    - **`namespace`** *string (optional)*<br/>
      The namespace of the _ConfigMap_.

    - **`key`** *string*<br/>
      The key of the configmap field to use. 

- **`list`** *list of the ref variants described above*

  Import a list of component descriptors. Every entry may be 
  one of the variants `ref`, `confimapRef` or `secretRef`.

  Exactly one of `ref`, `confimapRef`, `secretRef` or `list` must be given.

The variants `confimapRef` or `secretRef` are ONLY intended for development purposes,
for productive use cases, only the external reference variant. The data value described by
the key must be a component descriptor in YAML format.

Single descriptor and list imports must directly match the required component descriptor
imports of the used blueprint.
An explicit mapping is not possible. Therefore, list imports of a blueprint can
only be satisfied by list imports of the installation and descriptor imports by
descriptor imports.

Providing an inline component descriptor is not possible here.

**Example**
```yaml
imports:
  componentDescriptors:
  - name: ""
      ref:
        componentName: github.com/my-comp
        version: v0.0.1
  #      repositoryContext:
  #        type: ociRegistry
  #        baseUrl: ""
  - name: my-import
    secretRef: 
      name: ""
      namespace: "" #  optional, defaulted to installation namespace
      key: "" 
  - name: my-import
    configMapRef: 
      name: ""
      namespace: "" #  optional, defaulted to installation namespace
      key: "" 
  - name: my-list
    list:
      - ref:
          componentName: github.com/my-comp
          version: v0.0.1
      #        repositoryContext:
      #          type: ociRegistry
      #          baseUrl: ""
      - secretRef:
          name: ""
          namespace: "" #  optional, defaulted to installation namespace
          key: "" # optional
      - configMapRef:
          name: ""
          namespace: "" #  optional, defaulted to installation namespace
          key: "" # optional
```

For data and target imports, the imported values are copied for nested installations.
This ensures that, if a secret containing data is modified, the nested installations still
have access to the old data until everything has been properly reconciled. As component
descriptors are expected to be immutable, only the references are copied for
nested installations, not the actual values. Modyfing or deleting a component
descriptor which is currently being imported by one or more installations is therefore
strongly discouraged and could lead to undesired behaviour of the respective installation(s).


### Import Data Mappings

It can happen that imported data is of a different format than the expected schema
defined in the blueprint. One possible solution is to add an additional blueprint
that transforms the data.
As this approach would result in a big overhead for just transforming some data. 
It should be possible to easily transform imported data to satisfy the required
structure of imports of blueprints.

This transformation can be done in with _Import Data Mappings_. They are
specified in the spec field `importDataMappings`.

They define a map of imports of a blueprint that can be templated using
[spiff](https://github.com/mandelsoft/spiff). The mapping might provide values
for a subset of the blueprint imports. Unmapped imports are expected to
be satisfied directly by the installation imports.

These mappings open up the following possibilities:
- combine multiple installation imports into one import structure
- use hard-coded values for blueprint imports
- use only parts of the installation imports

All values imported by an installation can be accessed in the templating by
their import names.

**Example**

*Blueprint specification:*
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint
imports:
- name: providers
  type: data
  schema:
    type: array
    items:
      type: string
- name: identifier
  type: data
  schema:
    type: string
- name: aws-credentials
  type: data
  schema:
    type: object
    properties:
      accessKeyID:
        type: string
      accessKeySecret:
        type: string
```

*Installation snippet:*
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

Exports define the data that is created by the installation and exported
for consumption by other installations in the same scope or by
parent installations.

By default there is a matching mechanism in place
that matches the exports of an installation directly with the exports of
the blueprint by the used name.

This default mapping requires the exported data to directly match the data
structure provided by the exports of blueprint. Because this must not necessarily
be the case under all circumtances, it is possible to define an explicit mapping
of data from the blueprint exports to the installation exports. This
is done by [export data mappings](#export-data-mappings).
This might be required, if the exported _DataObject_ is intended to be consumed
ba another installation requiring a dedicated data structure not provided
this way by the blueprint. Basically this export data mapping the 
comparable with the [import data mapping](#import-data-mappings) on the
consuming side. When establishing the flow between two installations
under the same responsibility a required mapping can be done on either side.
But this is not the case if the concerned installations are under different
responsibilities, or if there are multiple providing and consuming installations.

For both purposes (the default mapping by name and the explicit data mapping),
every installation export features a `name` attribute, that must be unique
for all exports, regardless of their types.

There are two basic types of exports:
- [Data exports](#data-exports) that result in data objects.
   These kind of exports can also be mapped/transformed in the installation. 
- [Target exports](#target-exports) that result in targets.
   The target types much match and cannot be mapped/transformed by the installation.
   
Exports are declared in the spec field `exports` as list within a type specific
nested field.

### Data Exports

The export field `data` is used to declare a list of data exports.
An export declaration uses the following fields:

- **`name`** *string*

  The name of the export used for the implicit or explicit mapping to the blueprint exports.


- **`dataRef`** *string*

  This field can be used to specify the name of a _DataObject_ in the parent [scope](#scopes) 
  of an installation that should be created. For top-level installations the name
  must comply to the Kubernetes rules for object names.

Export to secrets or configmaps are not possible.

If this name matches a blueprint export, the exported value is directly used.
If an export has to be modified see [export data mapping](#export-data-mappings).


**Example**
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

The export field `targets` is used to declare a list of target exports.
An export declaration uses the following fields:

- **`name`** *string*

  The name of the export used for the implicit or explicit mapping to the blueprint exports.


- **`target`** *string*

  This field can be used to specify the name of a _Target_ in the parent [scope](#scopes)
  of an installation that should be created. For top-level installations the name
  must comply to the Kubernetes rules for object names.

The export of target lists is not possible.

**Example**
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

It can happen that data exported by a blueprint is of a different format than
what is needed in the scope.  One possible solution is to add an additional
blueprint that transforms the data. As this approach would result in a big
overhead for just transforming some data, an additional method is needed to
transform that data.

This transformation can be done in with _Export Data Mappings_. They are
specified in the spec field `exportDataMappings`.


They define a map of exports of an installation that can be templated using
[spiff](https://github.com/mandelsoft/spiff). The mapping might provide values
for a subset of the installations exports. Unmapped exports are expected to
be satisfied directly by the blueprint imports.

These mappings open up the following possibilities:
- create more exports from one or multiple exports of a blueprint
- combine multiple exports to one
- export hard coded values

All values exported by a blueprint can be accessed in the templating by their export names.


**Example**

*Blueprint specification:*
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint
exports:
- name: identifier
  type: data
  schema:
    type: string
- name: aws-credentials
  type: data
  schema:
    type: object
    properties:
      accessKeyID:
        type: string
      accessKeySecret:
        type: string
- name: gcp-credentials
  type: data
  schema:
    type: object
    properties:
      serviceaccount.yaml:
        type: string
```

*Installation snippet:*
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

- **`landscaper.gardener.cloud/operation`**:

  - `reconcile`: start a default reconcile on the installation
  - `force-reconcile`: skip the reconcile/pending check and directly start a new reconcilition flow. 
    > **Warning:** Imports still have to be satisfied.
  - `abort`: abort the current run which will abort all subinstallation but will wait until all current running components have finished.
 
- **`landscaper.gardener.cloud/skip`**: 
  - `true`: skips the reconciliation of a component which means that it will not be triggered by configuration or import change.

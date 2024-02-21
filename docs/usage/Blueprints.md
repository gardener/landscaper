---
title: Blueprints
sidebar_position: 1
---

# Blueprints

## Definition

A Blueprint is a parameterized description of how to deploy a specific component.

The description follows the kubernetes controller approach:

The task of a Blueprint is to provide deployitem descriptions based on its input and outputs based on the input and the state of the deployment.

The rendered deployitems are then handled by independent kubernetes controllers, which perform the real deployment tasks. 
This way, the Blueprint does not execute deployment actions, but provides the target state of formally described 
deployitems. The actions described by the Blueprint itself are therefore restricted to YAML-based manifest rendering. 
These actions are described by [template executions](./Templating.md).

A Blueprint is a filesystem structure that contains the blueprint definition at `/blueprint.yaml`. Any other additional file can be referred to in the blueprint.yaml for JSON schema definitions and templates.

Every Blueprint must have a corresponding component descriptor that is used to reference the Blueprint and define its dependencies.

```
my-blueprint
├── data
│   ├── gotemplate.tmpl
│   └── <myadditional files>
├── installations
│   └── installation.yaml
└── blueprint.yaml
```

The blueprint definition (blueprint.yaml) describes
- declaration of import parameters
- declaration of export parameters
- JSONSchema definitions
- generation rules for deployitems
- generation rules for export values
- generation of nested installations

## Example

The following snippet shows the structure of a `blueprint.yaml` file. It is expected as top-level file in the blueprint filesystem structure. Refer to [apis/.schemes/core-v1alpha1-Blueprint.json](../../apis/.schemes/core-v1alpha1-Blueprint.json) for the automatically generated jsonschema definition.

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
    "$ref": "local://mytype" # references local type
# Import a target by specifying the targetType
- name: my-target-import-key
  required: true # required, defaults to true
  type: target # this is a target import
  targetType: landscaper.gardener.cloud/kubernetes-cluster
# Import a targetlist
- name: my-targetlist-import-key
  # required: true # defaults to true
  type: targetList # this is a targetlist import
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
# template the deployitems.
# For detailed documentation see #DeployExecutions
deployExecutions: 
- name: execution-name
  type: GoTemplate
  file: <path to file> # path is relative to the blueprint's filesystem root

# exportExecutions are a templating mechanism to 
# template the export.
# For detailed documentation see #ExportExecutions
exportExecutions:
- name: execution-name
  type: Spiff
  template: # inline template

# subinstallations is a list of installation templates.
# An installation template expose specific installation configuration are 
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
    - name: "parent-data" # data import name
      dataRef: "data" # dataobject name - refers to import of parent
    targets:
    - name: "" # target import name
      target: "" # target name
  #importDataMappings: {}

  exports:
    targets:
    - name: "" # target export name
      target: "" # target name
  #exportDataMappings: {}

```

## Import Definitions

Blueprints describe formal imports. A formal import parameter has a name and a *value type*. It may describe a single simple value or a complex data structure. There are several *types* of imports, indicating different use cases:
- **`data`**

  This type of import is used to import arbitrary data according to its value type. The value type is described by a [JSONSchema](#jsonschema).


- **`target`**

  This type declares an import of a [deployment target object](./Targets.md). It is used in the rendered deployitems to specify the target environment for the deployment of the deployitem.


- **`targetList`**

  This type can be used if, instead of a single target object, an arbitrary number of targets should be imported. All targets imported as part of a targetlist import must have the same `targetType`. For more information on how to work with TargetList imports, see the documentation [here](./TargetLists.md).


The imports are described as a list of import declarations in the blueprint top-level field `imports`. An import declaration has the following fields:

- **`name`** *string*

  Identifier for the import _parameter_. Can be used in the templating to access the actual import value provided by the installation.


- **`type`** *type*

  The type of the import as described above.
  For backward compatibility, the `type` field is currently optional for *data* and *target* imports, but it is strongly recommended to specify it for each import declaration.


- **`required`** *bool* (default: `true`)

  If set to false, the installation does not have to provide this import.


- **`default`** *any*

  If the import is not required and not provided by the installation, this default value will be used for it. 


- **`imports`** *list of import declarations*

  Nested imports only exist if the owning import is satisfied. Cannot be specified for a required import. See [here](./ConditionalImports.md) for further details.


- **`schema`** *JSONSchema*

  Must be set for imports of type `data` (only). Describes the structure of the expected import value as [JSONSchema](#jsonschema).


- **`targetType`** *string*

  Must be set for imports of type `target` and `targetList` (only). It declares
  the type of the expected [*Target*](./Targets.md) object. If the `targetType` 
  does not contain a `/`, it will be prefixed with `landscaper.gardener.cloud/`.


**Example**
```yaml
imports:
- name: myimport # some unique name
  required: false # defaults to true if not set
  type: data # type of the imported object
  schema:
    type: object
    properties:
      username:
        type: string
      password:
        type: string
  default:
    value: 
      username: foo
      password: bar
- name: mycluster
  type: target
  targetType: kubernetes-cluster # will be defaulted to 'landscaper.gardener.cloud/kubernetes-cluster'
```

Values provided by _Installations_ for import parameters are validated
before any invasive action is done for an installation. There are two
possibilities to validate the input:
- using the [`schema`](#jsonschema) attribute to describe a validation using a
  JSON schema. 
  
  This validation always refers to a single import parameter. The definition
  of simple value parameters should be avoided, if possible. Instead, always
  a group of semantically related attributes should be aggregated into
  a structural value import. This can be validated as a whole.
- using the [`importExecutions`](#import-values) it is possible to describe templated
  checks on the complete set of imports and to provide a list of validation errors.

## Export Definitions

Blueprints describe formal exports whose values can be exported by using
_Installations_ to _DataObjects_ to be consumed by other _Installations_, again.
The following types can be exported:

- **`data`**

  This type of export is used to export arbitrary data according to its value type. The value type is described by a [JSONSchema](#jsonschema).


- **`target`**

  This type declares an export of a [deployment target object](./Targets.md). It
  is used in the rendered deployitems to specify the target environment for the 
  deployment of the deployitem.

The exports are described as a list of export declarations in the blueprint
top-level field `exports`. An export declaration has the following fields:

- **`name`** *string*

  Identifier for the export _parameter_. Can be used in the templating to access the actual export value provided by the installation.


- **`type`** *type*

  The type of the export as described above.
  For backward compatibility, the `type` field is currently optional for *data* and *target* exports, but it is strongly recommended to specify it for each export declaration.


- **`schema`** *JSONSchema*

  Must be set for exports of type `data` (only). Describes the structure of the expected export value as [JSONSchema](#jsonschema).


- **`targetType`** *string*

  Must be set for exports of type `target` (only). It declares the type of the expected [*Target*](./Targets.md) object. If the `targetType` does not contain a `/`, it will be prefixed with `landscaper.gardener.cloud/`.


**Example**
```yaml
exports:
- name: myexport
  type: data
  schema:
    type: object
    properties:
      username:
        type: string
      password:
        type: string
- name: myclusterexport
  type: target
  targetType: kubernetes-cluster # will be defaulted to 'landscaper.gardener.cloud/kubernetes-cluster'
```

## JSONSchema

[JSONSchemas](https://json-schema.org/) are used to describe the structure of `data` imports and exports. The provided import schema is used to validate the actual import value before executing the blueprint.

It is recommended to provide a description and an example for the structure, so that users of the blueprint know what to provide (see the [json docs](http://json-schema.org/understanding-json-schema/reference/generic.html#annotations)).

For detailed information about the jsonschema and landscaper specifics see [JSONSchema Docs](./JSONSchema.md)

## Rendering

The task of a _Blueprint_ is to provide deployitems and final output for
the data flow among _Installations_ based of their input values provided
by the actual _Installation_ is evaluated for.

This is described by rule sets consisting of [templates](./Templating) carried together
with the blueprint.

All template [executions](./Templating.md) get a common standardized binding:

- **`imports`**

  the imports of the installation, as a mapping from import name to assigned values


- **`cd`**

  the component descriptor of the owning component


- **`components`**

  the component descriptors of all referenced components, either directly or transitively (includes the component descriptor of the owning component too)


- **`blueprintDef`**

  the blueprint definition, as given in the installation (not the blueprint.yaml itself)


- **`componentDescriptorDef`**

  the component descriptor definition, as given in the installation (not the component descriptor itself)


Additionally, there are context specific bindings and those depending on the chosen
template processor.

**Example**
```yaml
imports:
  <import-name>: <import value>
cd: <component descriptor>
blueprintDef: <blueprint definition> # blueprint definition from the Installation
componentDescriptorDef: <component descriptor definition> # component descriptor definition from the installation
```

The rendering result must be a YAML map document.
The rendered elements are typically expected under a dedicated certain top-level
node (e.g. `deployItems` for a deployitem execution).

There are several rendering contexts:
- [`importExecutions`](#import-values) rendering of additional import values derived from the input values provided by the _Installation_ and/or cross-import input validation.
- [`deployExecutions`](#deployitems) rendering of deployitems produced by the blueprint for the actual installation.
- [`exportExecutions`](#export-values) rendering of values for the [export parameters](#export-definitions) of the blueprint.
- [`subinstallationExecutions`](#nested-installations) rendering of installations to be instantiated in the context of the actual blueprint execution.

### Import Values

If there are several templates used for some rendering tasks it might be 
useful to share some attributes derived from the values for the declared
imports. This can avoid replicating the calculation of those values to
several templates.

To support this, _Blueprints_ may declare import executions using the
top-level field `importExecutions`. It may list any number of appropriate
template [executions](./Templating.md).  This can be used to enrich the set
of import bindings for further template processing steps.

The template processing is fed with the [standard binding](#rendering).

A template execution should return a YAML document with two optional
top-level nodes:

- **`bindings`** *map*

  This node is expected to contain a map with additional bindings which
  will be added to the regular import bindings. This way it is even possible
  to modify or replace original import values for the further processing steps,
  for example, to provide more complex defaults based on other import values.

  The additional bindings are added incrementally to the `imports` binding,
  meaning the order of the executions is relevant and previously added
  bindings are available for the processing of the following executions.


- **`errors`** *string list*

  Alternatively the template processing may provide a list of validation errors.
  Like [schema](#jsonschema) validations, this could be used to validate imports
  before invasive actions are started. But for import executions the template
  has access to the complete set of imports provided by the _Installation_ and
  can therefore perform cross-checks.

  The first execution providing errors will abort the execution of further steps.

Other nodes in the document are ignored.

**Example**
```yaml
imports:
  - name: prefix
    type: data
    scheme:
      type: string
  - name: suffix
    type: data
    scheme:
      type: string
      
importExecutions:
  - name: check
    type: Spiff
    template:
      errors:
        - (( imports.prefix == imports.suffix ? "prefix and suffix must be different" :~~ ))
      bindings:
        basename: "tempfile"
  - name: compose
    type: Spiff
    template:
       bindings:
         compound: (( imports.prefix imports.basename imports.suffix ))
```

If the blueprint is fed with

```yaml
imports:
  prefix: /tmp/
  suffix: .tmp
```

the final import bindings would look like this:

```yaml
imports:
  prefix: /tmp/
  suffix: .tmp
  basename: tempfile
  compound: /tmp/tempfile.tmp
```

### DeployItems

The main task of a _Blueprint_ is to provide _DeployItems_. Therefore, the blueprint
manifest uses a top-level field `deployExecutions` listing any number of appropriate
template [executions](./Templating.md). 

**Example**
```yaml
deployExecutions:
  - name: kubernetes
    type: GoTemplate
    file: data/gotemplate.tmpl
```

The template processing is fed with the [standard binding](#rendering) and supports [state handling](./Templating.md#state-handling).

A template execution must return a YAML document with at least the top-level
node `deployItems`. It is expected to contain a list of deployitem specifications.
Other nodes in the document are ignored.
These specifications will then be mapped to final _DeployItems_ by the _Landscaper_.

A deployitem specification has the following fields:

- **`name`** *string*

  The name of the item in the context of the blueprint. It is used to generate
  the name of the final deployitem together with its scope and for referring
  to exported values in ete [export value](#export-values) rendering.

  The name must be unique in the context of the blueprint over all executions.


- **`dependsOn`** *string list*

  This list of item names can be used to enforce an ordering for the creation.
  The deletion is done in the opposite order.


- **`type`** *string*

  The type of the deployitem described. This type finally determines the expected
  structure for the configuration, and it determines the kind of _Deployer_ 
  responsible for the item.  This is the contract between the deployers and
  the deployitem creator.


- **`target`** *target import reference*

  This reference denotes the target object
  describing the target environment the depoyitem should be deployed to,
  e.g. a dedicated kubernetes cluster.
  It refers to a target import or targetlist import and has the following fields:


  - **`import`** *string*

  Name of the import importing the referenced target.


  - **`index`** *int (optional)*

  If the import refers to a targetlist import, `index` specifies the list index of the referenced target.

  - **`updateOnChangeOnly`** *bool*

  If set on true, a successfully processed deployitem is only deployed again, if something has changed in its spec. 


  - **`name`** *string (deprecated)*

  Name of the in-cluster target object. It must be taken from the `.metadata.name` field of the imported target
  (e.g. `.imports.mytarget.metadata.name`).
  Deprecated: reference an imported target via `import` instead.


- **`labels`** *string map*

  This map is used to attach labels to the generated deployitem.


- **`configuration`** *any*

  The structure of this field depends on the type of the deployitem.
  It is intended to tell the deployer about the expected target state to achieve
  Depending on the deployer it is possible to request information about
  the deployment, for example, the address of a generated load balancer for
  a Kubernetes service object.

  This structure is the formal contract towards the deployer. It should
  be versioned according to the Kubernetes mechanism with `apiVersion` and
  `kind`. But this is not enforced by the _Landscaper_.

  While the requesting of imports is not formalized (please refer to the
  [documentation of the dedicated deployer](../deployer/README.md)), the
  deployer can provide exports in formal way in the status of a deployitem.
  These values are then available in the binding for the
  [export executions](#export-values).


**Example rendered document**:
```yaml
deployItems:
- name: myfirstitem # unique identifier of the step
  target:
    name: my-kubernetes-cluster
    namespace: my-application
  config:
    apiVersion: mydeployer/v1
    kind: ProviderConfiguration
    ...
```

All lists of deployitem specifications of all template executions are appended to one list as they are specified in the deployExecution.

**Example**:

*Bindings*:
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
  my-cdlist: # import of a component descriptor list
    meta:
      schemaVersion: v2
    components:
      - meta:
          schemaVersion: v2
        component: 
          name: component-1
          version: v1.0.1
          ...  # same structure as for key "cd"   
      - meta:
          schemaVersion: v2
        component:
          name: component-2
          version: v1.0.1
          ...

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
      access:
        type: ociRegistry
        imageReference: nginx:0.30.0

blueprintDef:
  ref:
    resourceName: gardener
  # inline:
  #   filesystem: # vfs filesystem
  #     blueprint.yaml: 
  #       apiVersion: landscaper.gardener.cloud/v1alpha1
  #       kind: Blueprint
  #       ...

componentDescriptorDef:
  ref:
    # repositoryContext:
    #   type: ociRegistry
    #   baseUrl: eu.gcr.io/myproj
    componentName: github.com/gardener/gardener
    version: v1.7.2

```

*Deploy Execution*:
```yaml
deployExecutions:
- name: default
  type: GoTemplate
  template: |
    deployItems:
    - name: deploy
      type: landscaper.gardener.cloud/helm
      target:
        import: cluster # will be replaced with a reference to the in-cluster target object the referenced import refers to
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

### Export Values

After a successful deployment of the generated _DeployItems_ the _Blueprint_ 
may use the provided export information of the deployitems to generate
values for its export parameters.

Therefore, the blueprint manifest uses a top-level field `exportExecutions` listing
any number of appropriate template [executions](./Templating.md).

**Example**
```yaml
exportExecutions:
  - name: kubernetes
    type: GoTemplate
    template: |
       exports:
         myexport:
           username: ...
           password: ...
```

The template processing is fed with the [standard binding](#rendering) and supports [state handling](./Templating#state-handling).
Additional bindings are provided to access the exports of generated elements:

- **`deployitems`** *map*

  This map contains a map entry for all generated deployitems according to their configured name.
  The entry then contains the configured deployitem exports.


- **`dataobjects`** *map*

  This map contains the values of the nested data objects provided by nested
  installations.


- **`targets`** *map*

  This map contains the values of the nested targets provided by all nested
  installations.


- **`values`** *map* *(deprecated)*

  This node is a map with the entries shown above.


A template execution must return a YAML document with at least the top-level
node `exports`. It is expected to contain a map of export parameters mapped to
concrete values. Other nodes in the document are ignored.

**Example rendered document**:
```yaml
exports:
  myexport: # name of the export parameter
    username: ...
    password: ...
```

The result of multiple template executions exports will be merged in the defined
order, whereas the latter defined values overwrites previous templates. The final
result must provide a value for all export parameters.


**Example**
```yaml
exportExecutions:
- name: default
  type: GoTemplate
  template: |
    exports:
      url: https://{{ .deployitems.ingressPrefix  }}.{{ .dataobjects.domain }} # resolves to https://my-pref.example.com
      cluster:
        type: {{ .targets.dev-cluster.spec.type  }}
        config: {{ .targets.dev-cluster.spec.config  }}
```

#### Data Exports

The typical export is a data value. The structure for arbitrary data
is completely free. The value is just taken as it is defined
by the dedicated exports map entry.

A data export of a blueprint can be exported by a using installation.
The result will be a _DataObject_ of the name specified in the exporting
installation in the parent scope of this installation.

#### Target Exports

It is possible to export targets, also. Hereby the target will be created
in the scope of the installation the blueprint is instantiated for.
To export a target the structure below the export name must match
a target specification:

- **`type`** *string*

  The type of the target.


- **`configuration`** *any*

  The configuration of the target. The structure depends on the [type of
  the target](../technical/target_types.md).


- **`labels`** *string map*

  This map is passed to the `labels` section of the generated target object.


- **`annotations`** *string map*

  This map is passed to the `annotations` section of the generated target object.


The specification does not allow to provide a name for the target. It is
created in the parent scope of the installation the blueprint is instantiated for, 
if this export is exported by the installation, also.
In this case the name in this scope is provided by the export definition of
the installation.


## Nested Installations

Blueprints may contain installation specifications which will result in installations when the blueprint is instantiated through an installation.
They have a naming scope that is finally defined by the installation the blueprint is instantiated for.
Naming scope means that nested installations can only import data that is also imported
by the parent or exported by other nested installations with the same parent.

Nested installation specifications offer the same configuration as real installations
except that the used blueprints have to be defined in the component descriptor
of the blueprint (either as resource or by a component reference).
Inline blueprints are also possible, although not recommended for productive purposes.

Nested installations can be defined in two flavors;

- [Static Installations](#static-installations)
- [Templated Installations](#templated-installations)

In any case the result is a list of installation specifications.
Every specification is mapped to a dedicated nested installation object
in the context of the actual installation the blueprint is instantiated for.

All possible options to define a nested installation can be used in parallel and are summed up.
The name of all described nested installations must be unique in the context of
the blueprint. But they don't need to be unique in the landscaper namespace,
because the live only in the context of installation the blueprint is
instantiated for. If the blueprint is used for multiple installations (flat or
nested, again), the generated installations and used data objects are living
in separate naming scopes.

The installation specification (although technically called `InstallationTemplate` it is not 
templated by a template processor but used by the _Landscaper_ to generate a final _Installation_ object)
has the following format:

A specification will still be processed by the landscaper
before putting it as regular installations into the landscaper data plane:
- it handles the scopes
- it maps the usage of directly used parent imports
- it maps the usage of the component descriptor of the top-level installation.

**Example**
```yaml
  apiVersion: landscaper.gardener.cloud/v1alpha1
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
    - name: ""
      targetListRef: "" # references a targetlist import of the parent
  #importMappings: {}

  exports:
    targets:
    - name: "" # target export name
      target: "" # target name
  #exportMappings: {}
```

### Static Installations

Static installations are configured as a list under the top-level field
`subinstallations` in the blueprint manifest.

Every entry can either be a reference to file (using the field `file`) provided
in the blueprint's [filesystem](#blueprints) or directly inlined. Only one of
both flavors can be used for one list entry.

**Example**
```yaml
subinstallations:
- file: installations/installation.yaml
- apiVersion: landscaper.gardener.cloud/v1alpha1
  kind: InstallationTemplate
  name: mysubinst
  imports:
    targets:
    - name: "also-my-foo-targets"
      targetListRef: "my-foo-targets"
```

Static installations are not templated and cannot refer to import values.

### Templated Installations

Similar to how deployitems can be defined, it is also possible to create nested
installations based on the imports by using template [executions](./Templating.md).
Templated installations are configured as a list under the top-level field
`subinstallationExecutions` in the blueprint manifest.

The template processing is fed with the [standard binding](#rendering) and supports [state handling](./Templating#state-handling).

A template execution must return a YAML document with at least the top-level
node `subinstallations`. It is expected to contain a list of installation
specifications. Other nodes in the document are ignored.


_**Example**:
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

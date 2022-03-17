# Templating

Landscaper uses templating to dynamically generate configurations based on a given data binding for various purposes. An example is the generation of deployitem manifests based on the imports of a blueprint. 
The Landscaper supports multiple [template engines](#template-engines). Depending on the purpose, dedicated value bindings are provided as input for the templating. Templating is executed in so-called 'executions', defining the context for the templating. Landscaper uses executions for:
- [**`deployExecutions`**](./Blueprints.md#deployitem-templates) for rendering deployitems
- [**`exportExecutions`**](./Blueprints.md#export-templates) for rendering exports
- [**`subinstallationExecutions`**](./Blueprints.md#installation-templates) for rendering nested installations

For each of these purposes, a list of executions can be specified. Every execution can use a different template engine. The results of all specified executions with the same purpose will be merged. 

For detailed information of blueprints see the [Blueprint Docs](./Blueprints.md).

- [Templating](#templating)
    - [Template Execution](#template-execution)
    - [State Handling](#state-handling)
    - [Template Engines](#template-engines)
      - [Go Template](#go-template)
      - [Spiff](#spiff)

### Template Execution

Landscaper uses templating for various purposes, most prominently in the blueprint (e.g. for deployitem and export generation). The dedicated section in the respective manifests is always a list of template execution configurations. Each execution is defined by a set of attributes:
- **`name`** *string*
  The _name_ is used for providing error messages during the templating execution. It is also used as an identifier for the [state](#state-handling) of the execution.

- **`type`** *string*
  The _type_ specifies which template engine should be used. Currently supported types are [`GoTemplate`](#go-template) and [`Spiff`](#spiff).

- **`file`** *string* [optional]
  If this property is set, the template is read from the specified file of the blueprint file structure. Exactly one of `file` and `template` has to be specified.

- **`template`** *template* [optional]
  If this property is set, the template is read from the given inline data, according to the specification of the specified template engine type. Exactly one of `file` and `template` has to be specified.

The the rendered output of the templating must always be a YAML document. The document is expected to be a map. The structure is the same, independent of which template engine is used. The expected result is always read from a dedicated key, depending on the execution (e.g. `deployItems` for deployitem executions).

**Example**
```yaml
deployExecutions:
- name: my-spiff-template
  type: Spiff
  template:
    deployItems:
    - name: my-first-deploy-item
      type: landscaper.gardener.cloud/mock
      config: ...
- name: my-go-template
  type: GoTemplate
  template: |
    deployitems:
    - name: my-second-deploy-item
      type: landscaper.gardener.cloud/mock
      config: ...
```

### Filesystem

The blueprint's filesystem structure is accessible for the template engines as root file system.

**Example**
- Filesystem
  ```
  my-blueprint
  ├── data
  │   ├── template
  │   └── config
  └── blueprint.yaml
  ```
- Execution snippet from blueprint.yaml
  ```yaml
  - name: my-go-template
    type: GoTemplate
    file: "data/template"
  ```
- Template file
  ```
  deployitems:
    - name: my-second-deploy-item
      type: landscaper.gardener.cloud/mock
      config:
  {{ include "data/config" . | indent 6 }}
  ```


### State Handling

Depending on the purpose of the execution, Landscaper supports state handling. An execution can provide information that should be kept among multiple evaluations of the execution (e.g. when the installation is updated). The mechanism, how the state is past to and read from an execution depends on its template engine.


### Template Engines

The Landscaper currently supports two template engines:
- [**`GoTemplate`**](#go-template) [Go Template]((https://golang.org/pkg/text/template/)) enhanced with [sprig](http://masterminds.github.io/sprig/) functions.
- [**`Spiff`**](#spiff) [Spiff++](https://github.com/mandelsoft/spiff) templating.

Regardless of the chosen engine, the output is always expected to have the same structure.

:warning: Note that OS functions are not available for security reasons.


#### Go Template

The execution type to use for go templates is `GoTemplate`. As go templates are not valid YAML, they have to be provided as a string. Because this is typically a multi-line string, the `|` notation is mostly used.

**Example**
```yaml
- name: my-go-template
  type: GoTemplate
  template: |
    deployitems:
    - name: my-second-deploy-item
      type: landscaper.gardener.cloud/mock
      config: {{ .imports.config }}
```

##### Additional Functions

The `GoTemplate` executor simply is standard [go template](https://golang.org/pkg/text/template/) enhanced with [sprig](http://masterminds.github.io/sprig/) functions.

The following additional functions are available:

- **`include(path string, binding interface{}): string`**
  reads and executes a template from the given file with the provided binding (similar to helm's 'include') 
- **`readFile(path string): []byte`**
  reads a file from the blueprints filesystem
- **`readDir(path string): []FileInfo`**
  returns all files and directories in the given directory of the blueprint's filesystem.
- **`toYaml(interface{}): string`**
  converts the given object to valid yaml
- **`getResource(ComponentDescriptor, keyValuePairs ...string): Resource`**
  searches a resource in the given component descriptors that matches the specified selector. The selector are key-value pairs that describe the resource's identity.
  e.g. `getResource .cd "name" "myResource"` -> returns the resource with the name `myResource`
- **`getComponent(componentDescriptor, keyValuePairs ...string): ComponentDescriptor`**
  searches a component in the given component descriptors that matches the specified selector. The selector are key-value pairs that describe the component reference's identity.
  e.g. `getComponent .cd "name" "myComp"` -> seraches in the component descriptor for a component reference with the name `myComp` and returns the referenced component descriptor.
- **`getRepositoryContext(componentDescriptor): RepositoryContext`**
  returns the effective repository context of the given component descriptor
- **`parseOCIRef(ref string): [2]string`**
  parses an oci reference and returns the repository and the version.
  e.g. `host:5000/myrepo/myimage:1.0.0` -> `["host:5000/myrepo/myimage", "1.0.0"]`
- **`ociRefRepo(ref string): string`**
  parses an oci reference and returns the repository.
  e.g. `host:5000/myrepo/myimage:1.0.0` -> `"host:5000/myrepo/myimage"`
- **`ociRefVersion(ref string): string`**
  parses an oci reference and returns the version.
  e.g. `host:5000/myrepo/myimage:1.0.0` -> `"1.0.0"`
- **`resolve(access Access): []byte`**
  resolves an artifact defined by a typed access definition.
   The resolve function is currently able to handle artifacts of type `ociRegistry`, others may be added in the future.
   The function always returns a byte array of the artifact response
   ```
   # e.g for a oci registry artifact
   type: ociRegistry
   imageReference: host:5000/myrepo/myimage:1.0.0
   ```
- **`generateImageOverwrite(componentDescriptor,componentDescriptorList): ImageVector`**
  returns the image vector overwrite for the component descriptor and the componentDescriptorList. The arguments componentDescriptor and componentDescriptorList are optional and defaulted to the current context (.cd and .components).
  Example: 
  ```  
  imageVectorOverWrite:
  {{- generateImageOverwrite | toYaml | nindent 2 }}
  ```
  results in something like
  ```yaml
  imageVectorOverWrite:
    images:
    - name: cloud-controller-manager
      repository: eu.gcr.io/gardener-project/kubernetes/cloud-provider-aws
      tag: v1.17.15
      targetVersion: 1.17.x
  ...
  ```


##### State

Old state is provided via an additional `state` binding. New state is taken from the `state` node of the rendered template, if it exists.

**Example**
```yaml
- name: my-go-template
  type: GoTemplate
  template: |
    state: {{if .state}}{{add .state 1}}{{else}}1{{end}}
    deployitems:
    - name: my-second-deploy-item
      type: landscaper.gardener.cloud/mock
      config: {{ .imports.config }}
```



#### Spiff

The execution type to use for spiff templates is `Spiff`. The template is provided as YAML.

**Example**
```yaml
- name: my-spiff-template
  type: Spiff
  template:
    deployItems:
    - name: my-first-deploy-item
      type: landscaper.gardener.cloud/mock
      config: (( .imports.config ))
```

##### Additional Functions

- **`getResource(ComponentDescriptor, keyValuePairs ...string): Resource`**
  searches a resource in the given component descriptors that matches the specified selector. The selector are key-value pairs that describe the resource's identity.
  e.g. `getResource .cd "name" "myResource"` -> returns the resource with the name `myResource`
- **`getComponent(componentDescriptor, keyValuePairs ...string): ComponentDescriptor`**
  searches a component in the given component descriptors that matches the specified selector. The selector are key-value pairs that describe the component reference's identity.
  e.g. `getComponent .cd "name" "myComp"` -> seraches in the component descriptor for a component reference with the name `myComp` and returns the referenced component descriptor.
- **`parseOCIRef(ref string): [2]string`**
  parses an oci reference and returns the repository and the version.
  e.g. `host:5000/myrepo/myimage:1.0.0` -> `["host:5000/myrepo/myimage", "1.0.0"]`
- **`ociRefRepo(ref string): string`**
  parses an oci reference and returns the repository.
  e.g. `host:5000/myrepo/myimage:1.0.0` -> `"host:5000/myrepo/myimage"`
- **`ociRefVersion(ref string): string`**
  parses an oci reference and returns the version.
  e.g. `host:5000/myrepo/myimage:1.0.0` -> `"1.0.0"`
- **`generateImageOverwrite(componentDescriptor,componentDescriptorList): ImageVector`**
  returns the image vector overwrite for the component descriptor and the componentDescriptorList. The arguments componentDescriptor and componentDescriptorList are optional and defaulted to the current context (.cd and .components).
  Example: 
  ```yaml
  imageVectorOverWrite:
    (( generateImageOverwrite() ))
  ```
  results in something like
  ```yaml
  imageVectorOverWrite:
    images:
    - name: cloud-controller-manager
      repository: eu.gcr.io/gardener-project/kubernetes/cloud-provider-aws
      tag: v1.17.15
      targetVersion: 1.17.x
  ...
  ```

##### State

Spiff already has state handling implemented, see [here](https://github.com/mandelsoft/spiff#-state-) for details.

# Template Executors

This page contains all available template executors that can be used in the deploy and export executions of blueprints.

For detailed information of blueprints see the [Blueprint Docs](./Blueprints.md).

- [Template Executors](#template-executors)
    - [GoTemplate](#gotemplate)
      - [State handling](#state-handling)
    - [Spiff](#spiff)

### GoTemplate
__Type__: `GoTemplate`

The `GoTemplate` executor simply is standard [go tempalte](https://golang.org/pkg/text/template/) 
enhanced with [sprig](http://masterminds.github.io/sprig/) functions.

In addition to the `sprig` functions, landscaper specific functions are offered:

- __readFile(path string): []byte__: reads a file from the blueprints filesystem
- __readDir(path string): []FileInfo__: returns all files and directories in the given directory of the blueprint's filesystem.
- __toYaml(interface{}): string__: converts the given object to valid yaml
- __getResource(ComponentDescriptor, keyValuePairs ...string): Resource__: searches a resource in the given component descriptors that matches the specified selector. The selector are key-value pairs that describe the resource's identity.
  `e.g. getResource .cd "name" "myResource"` -> returns the resource with the name `myResource`
- __getComponent(componentDescriptor, keyValuePairs ...string): ComponentDescriptor__: searches a component in the given component descriptors that matches the specified selector. The selector are key-value pairs that describe the component reference's identity.
  `e.g. getComponent .cd "name" "myComp"` -> seraches in the component descriptor for a component reference with the name `myComp` and returns the referenced component descriptor.
- __getRepositoryContext(componentDescriptor): RepositoryContext__: returns the effective repository context of the given component descriptor
- __parseOCIRef(ref string): [2]string__: parses an oci reference and returns the repository and the version.
  `e.g. host:5000/myrepo/myimage:1.0.0 -> ["host:5000/myrepo/myimage", "1.0.0"]`
- __ociRefRepo(ref string): string__: parses an oci reference and returns the repository.
  `e.g. host:5000/myrepo/myimage:1.0.0 -> "host:5000/myrepo/myimage"`
- __ociRefVersion(ref string): string__: parses an oci reference and returns the version.
  `e.g. host:5000/myrepo/myimage:1.0.0 -> "1.0.0"`
- __resolve(access Access): []byte__: resolves an artifact defined by a typed access definition.<br>
   The resolve function is currently able to handle artifacts of type `ociRegistry`, others may be added in the future.
   The function always returns a byte array of the artifact response<br>
   ```
   # e.g for a oci registry artifact
   type: ociRegistry
   imageReference: host:5000/myrepo/myimage:1.0.0
   ```
- __generateImageOverwrite(componentDescriptor,componentDescriptorList): ImageVector__: returns the image vector overwrite
  for the component descriptor and the componentDescriptorList. The arguments componentDescriptor and 
  componentDescriptorList are optional and defaulted to the current context (.cd and .components).
  ```
  Example: 
  
  imageVectorOverWrite:
  {{- generateImageOverwrite | toYaml | nindent 2 }}
  
  results in something like
  
  imageVectorOverWrite:
    images:
    - name: cloud-controller-manager
      repository: eu.gcr.io/gardener-project/kubernetes/cloud-provider-aws
      tag: v1.17.15
      targetVersion: 1.17.x
  ...
  ```

:warning: Note that OS functions are not available for security reasons.

The template can be either defined inline as string or a file can be referenced.
```yaml
- type: GoTemplate
   template: |
     abc: {{ my template }}

- type: GoTemplate
  file: /file/path
```

#### State handling

The GoTemplate executor also offers the possibility to write and read from a state.
The state is read before the templating and can be accessed in addition to all other input values with:
```yaml
otherinputs: 
otherinputs2:

state:
  mystate:
```

Values in this state can be stored by provding an additional output in the executor.
```yaml
myexports:

state: 
  mystate:
```

**Example**
```yaml
# read and write to state in the deploy executor
{{ $myval := {{ default .state.stateval1 (genPrivateKey rsa) }} }}
deployItems:
- myitems: {{ $myval }}
state:
  stateval1: {{ $myval }}
```

### Spiff
__Type__: `Spiff`

The `Spiff` executor is teh default [spiff++](https://github.com/mandelsoft/spiff) executor that is restricted to the blueprint's filesystem.

The root yaml template can be either defined inline as yaml/json or a file can be referenced.
```yaml
- type: Spiff
   template:
     abc: (( my template ))

- type: Spiff
  file: /file/path
```

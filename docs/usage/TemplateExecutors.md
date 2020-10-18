# Template Executors

This page contains all available template executors that can be used in the deploy and export executions of blueprints.

For detailed information of blueprints see the [Blueprint Docs](./Blueprints.md).

- [GoTemplate](#gotemplate)
- [Spiff](#spiff)

### GoTemplate
__Type__: `GoTemplate`

The `GoTemplate` executor simply is standard [go tempalte](https://golang.org/pkg/text/template/) 
enhanced with [sprig](http://masterminds.github.io/sprig/) functions.

In addition to the `sprig` functions, landscaper specific functions are offered:

- __readFile(path string)__: reads a file from the blueprints filesystem
- __readDir(path string)__: returns all files and directories in the given directory of the blueprint's filesystem.
- __toYaml(interface{})__: converts the given object to valid yaml

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

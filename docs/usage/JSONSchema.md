# JSONSchema

Data imports and exports of Blueprints are defined using JSONSchema.
JSONSchema is a definition language to describe the structure of json or yaml data.

See the official [JSONSchema documentation](http://json-schema.org/understanding-json-schema/index.html) for a detailed description of the definition.

JSONSchema describes a mechanism to reference jsonschema in a jsonschema.
In the Landscaper context the default jsonschema `$ref` property is extended by 3 additional protocols:
- `local://` - read from the Blueprints local attribute
- `blueprint://` - read from a file in the blueprint
- `cd://` - Component Descriptor

### Local

In a blueprint it is possible to define jsonschema in a property called `localTypes`.
This makes it possible to define a blueprint wide type that can be used across imports and exports in the blueprint.

The type can be used in other schemas by using the `$ref` property with `local://<name of the type>`.

:warning: the localtypes cannot be used by other blueprints

_Example_:

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

localTypes:
  "myType": 
    type: number
    
imports:
- name: my-import
  type: data
  schema:
    $ref: "local://myType"
- name: my-other-import
  type: data
  schema:
    type: object
    properties:
      a:
        type: string
      b:
        $ref: "local://myType"
```

### Blueprint

The blueprint type is similar to the [local type](#local) with the difference that the jsonschema can be defined in a file that is packaged with the blueprint.

This feature is useful when a definition is too large for a single file or should be split into different files for readability.

The type can be used with the `$ref` property and `blueprint://path/to/schema.json`.
The root and the pwd path are always considered to be the blueprint's directory (where the `blueprint.yaml` is located).

_Example_:

File structure
```
blueprintFolder
├── definitions
│   └── my-type.json
└── blueprint.yaml
```

```json
# definitions/my-type.json
{
  "type": "number"
}
```

```
# blueprint.yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint
    
imports:
- name: my-import
  type: data
  schema:
    $ref: "blueprint://definitions/my-type.json"
```

### Component Descriptor

The component descriptor makes it possible to reuse jsonschema definition from other components.

A jsonschema definition can be defined in component descriptor as resource.

```
# component descriptor with jsonschema resource
...
resources:
- name: my-definition
  type: landscaper.gardener.cloud/jsonschema
  ...
```

This resource can be referenced in a jsonschema with ``cd://<resource locator>`` 
The resource locator is an uri defined as key value pairs (`cd://<keyword>/<value>/<keyword>/<value>...`) that is used to traverse through component descriptors and get the resource.

The uri consists of 2 keywords whereas the last key _MUST_ be a resource
- __componentReferences__: search in the component descriptor that is referenced
- __resources__: use the resource with the name given by the value

As the names of the keywords suggests they are selecting other components descriptor by the componentReferences or resources in a component descriptor.

Some Examples:
```
# Use the jsonschema that is defined as resource in the component descriptor of the blueprint
# "cd://resources/my-def"
meta:
  schemaVersion: v2
component:
  name: my-blueprint-component-descriptor
  version: v0.1.0

  resources:
  - name: my-def
    type: landscaper.gardener.cloud/jsonschema
```

```
# Use the jsonschema that is defined in a component that is referenced by the component descriptor of the blueprint
# "cd://componentReferences/the-other-comp/resources/my-def"
---
meta:
  schemaVersion: v2
component:
  name: some-other-component-descriptor
  version: v0.1.0

  resources:
  - name: my-def
    type: landscaper.gardener.cloud/jsonschema
---
meta:
  schemaVersion: v2
component:
  name: my-blueprint-component-descriptor
  version: v0.1.0

  componentReferences:
  - name: the-other-comp
    componentName: some-other-component-descriptor
    version: v0.1.0
```

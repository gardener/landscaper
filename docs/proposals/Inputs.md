
- dataobject validation

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation

imports:
  A:
    name: A
    version: v1
  apiServerDomain: 
    name: "virtualapiserverdomain"
    version: v1 # optional, defaults to v1
  dev-aws-credentials: 
    name: "aws-credentials"
    version: v1 # optional; defaults to v1

# 
importMapping: # -> map[string]interface{} : input[import.name]
    A: (( imports.A ))
    B:
        A: (( importMapping.A ))
        B: “text”
        C:
        - E1
        - E2
    C:
    - "demo.gardener.cloud"
    - (( dataObjects.apiServerDomain ))
    D:
        F1: bla
        F2: (( dataObjects.credentials.field1 ))
        F3: (( inputs.A + 1 ))

exportMapping:
  dataObjectA: (( export.A.v1 ))
  apiderverUrl: myapiserverURl

exports:
  dataObjectA:
    name: B
    version: v1
```

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint


jsonSchema: "https://json-schema.org/draft/2019-09/schema" # required
localTypes:
    aws-credential: # inline
      type: number

imports:
- name: A
  type: data
  schema:
    "$schema": "https://json-schema.org/draft/2019-09/schema" # optional, defaulted to .jsonSchema
    type: string
  optional: false
- name: B
  type: data
  schema:
    $ref: "blueprint://types/gcp-credentials" # read file from blueprint content
  optional: false
- name: C 
  type: data
  schema:
    $ref: "local://aws-credentials"
  optional: false
- name: D
  type: data
  schema:
    $ref: "cd:///componentReferences/etcd/resources/my-type" # path in component descriptor
  optional: false

exports:
- name: key1
  type: data
  version: v1 # optional; defaults to "v1"
  schema:
    "$schema": "https://json-schema.org/draft/2019-09/schema" # optional, defaulted to .jsonSchema
    type: string
```

*Ideas*:
- versioned exports and versioned dataobject imports


```yaml
kind: Installation

dataObjects:
  apiServerDomain: "virtualapiserverdomain"
  credentials: "aws-credentials"

inputs:
    A: 5
    B:
        A: (( inputs.A ))
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
```

```yaml
kind: Blueprint


jsonSchema: "https://json-schema.org/draft/2019-09/schema" # required
localTypes:
    aws-credential: # inline
      type: number

imports:
    key1: 
      schema:
        "$schema": "https://json-schema.org/draft/2019-09/schema" # optional, defaulted to .jsonSchema
        type: string
      optional: false
    key2: 
      schema:
        $ref: "blueprint://types/gcp-credentials" # read file from blueprint content
      optional: false
    key3: 
      schema:
        $ref: "local://aws-credentials"
      optional: false
    key4: 
      schema:
        $ref: "cd:///componentReferences/etcd/localResources/my-type" # path in component descriptor
      optional: false

```
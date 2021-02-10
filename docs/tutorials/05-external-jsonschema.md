# External JSON Scheme

The goal of this tutorial is to combine the previously created blueprints and deploy them together.

__Prerequisites__:
- OCI compatible oci registry (e.g. GCR or Harbor)
- Kubernetes Cluster (better use two different clusters: one for the landscaper and one for the installation)
- Component-cli (see https://github.com/gardener/component-cli)

:warning: note that the repository `eu.gcr.io/gardener-project/landscaper/tutorials` is an example repository 
and has to be replaced with your own registry if you want to upload your own artifacts.
Although the artifacts are public readable so they can be used out-of-the-box without a need for your own oci registry.


### Resources

#### Resources JSONSchema


```yaml
# ./docs/tutorials/resources/external-jsonschema/definitions/component-descriptor.yaml
meta:
  schemaVersion: v2
component:
  name: github.com/gardener/landscaper/external-jsonschema/definitions
  provider: internal
  repositoryContexts:
  - baseUrl: eu.gcr.io/gardener-project/landscaper/tutorials/components
    type: ociRegistry
  resources: []
  componentReferences: []
  sources: []
```

```
# ./docs/tutorials/resources/external-jsonschema/definitions/component-descriptor
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "landscaper.gardener.cloud/tutorial/external-jsonschema/resources",
  "title": "Resources",
  "description": "Describes kubernetes resource requests and limits",
  "type": "object",
  "properties": {
    "requests": {
      "$ref": "#/definitions/resource"
    },
    "limits": {
      "$ref": "#/definitions/resource"
    }
  },
  "definitions": {
    "resource": {
      "properties": {
        "cpu": {
          "type": "string"
        },
        "memory": {
          "type": "string"
        }
      }
    }
  }
}
```

```
component-cli ca resources add ./docs/tutorials/resources/external-jsonschema/definitions -r ./docs/tutorials/resources/external-jsonschema/definitions/jsonscheme-resource.yaml -v 5
```

```
component-cli ca remote push ./docs/tutorials/resources/external-jsonschema/definitions
```

#### Echo Server Deployment

```yaml
# ./docs/tutorials/resources/external-jsonschema/echo-server/blueprint.yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

imports:
- name: cluster
  targetType: landscaper.gardener.cloud/kubernetes-cluster
- name: ingressClass
  schema:
    type: string
- name: resources
  schema:
    $ref: "cd://componentReferences/definitions/resources/resources-definition"

deployExecutions:
- name: default
  type: GoTemplate
  file: /defaultDeployExecution.yaml
```


```yaml
# ./docs/tutorials/resources/external-jsonschema/echo-server/component-descriptor.yaml
meta:
  schemaVersion: v2

component:
  name: github.com/gardener/landscaper/external-jsonschema/echo-server
  version: v0.1.0

  provider: internal

  repositoryContexts:
  - type: ociRegistry
    baseUrl: eu.gcr.io/gardener-project/landscaper/tutorials/components

  sources: []
  componentReferences:
  - name: definitions
    componentName: github.com/gardener/landscaper/external-jsonschema/definitions
    version: v0.1.0

  resources:
  - type: ociImage
    name: echo-server-image
    version: v0.2.3
    relation: external
    access:
      type: ociRegistry
      imageReference: hashicorp/http-echo:0.2.3
```

```yaml
# ./docs/tutorials/resources/external-jsonschema/echo-server/blueprint-resource.yaml
---
type: blueprint
name: echo-server-blueprint
version: v0.1.0
relation: local
input:
  type: "dir"
  path: "./blueprint"
  compress: true
  mediaType: "application/vnd.gardener.landscaper.blueprint.v1+tar+gzip"
...
```

```
component-cli ca resources add ./docs/tutorials/resources/external-jsonschema/echo-server -r ./docs/tutorials/resources/external-jsonschema/echo-server/blueprint-resource.yaml -v 5
```

```
component-cli ca remote push ./docs/tutorials/resources/external-jsonschema/echo-server
```

### Installation

Provide imports

```yaml
# ./docs/tutorials/resources/external-jsonschema/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-imports
data:
  ingressClass: "nginx"
  resources: |
    requests:
      memory: 50Mi
    limits:
      memory: 100Mi
```

```
kubectl apply -f ./docs/tutorials/resources/external-jsonschema/configmap.yaml
```

### Summary

- A blueprint has been created that includes a ingress-nginx and an echo server.
- With that blueprint is it now possible for others to reuse the aggregated blueprint and deploy the ingress together with the echo server.


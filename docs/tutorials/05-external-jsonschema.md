# External JSON Scheme

In the first tutorials a simple jsonschema was used to describe the namespace import.
```yaml
imports:
- name: namespace
  schema:
    type: string
```

When jsonschemas become more complex or are used by multiple components the requirement to declare a jsonschema centrally arises.

The goal of this tutorial is to showcase how this can be done in the context of Landscaper. Therefore, we will create a jsonschema in a 'definition' component and make it usable in another component.

__Prerequisites__:
- OCI compatible oci registry (e.g. GCR or Harbor)
- Kubernetes Cluster (better use two different clusters: one for the landscaper and one for the installation)
- Component-cli (see https://github.com/gardener/component-cli)

:warning: note that the repository `eu.gcr.io/gardener-project/landscaper/tutorials` is an example repository 
and has to be replaced with your own registry if you want to upload your own artifacts.
Although the artifacts are public readable so they can be used out-of-the-box without a need for your own oci registry.


### Resources

This tutorial uses 2 different components:
- _definitions_: component that contains the jsonschema definition
- _echo-server_: simple echo server that consumes the jsonschema.

#### Resources JSONSchema

First the shared jsonschema has to be created.
Kubernetes' definition of `resources`  is used as an example here.

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

With the jsonschema defined, it has to be linked to a new component descriptor. In this case, the jsonschema will be added as local artifact to the component descriptor.
For more detailed explanation have a look at the [second tutorial](./02-local-simple-blueprint.md)
   
Now, create the resource definition and component descriptor, add the resource with the component-cli and upload the component descriptor.

The result is a component descriptor that contains the previously created jsonschema as local resource.

<details>

```yaml
# ./docs/tutorials/resources/external-jsonschema/definitions/jsonschema-resource.yaml
---
type: landscaper.gardener.cloud/jsonschema
name: resources-definition
relation: local
input:
  type: "file"
  path: "./resources.json"
  mediaType: "application/vnd.gardener.landscaper.jsonscheme.v1+json"
...
```
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
component-cli ca resources add ./docs/tutorials/resources/external-jsonschema/definitions -r ./docs/tutorials/resources/external-jsonschema/definitions/jsonscheme-resource.yaml -v 5
```

```
component-cli ca remote push ./docs/tutorials/resources/external-jsonschema/definitions
```

</details>

#### Echo Server Deployment

With the _definitions_  component ready, let's move on and create a component which will consume those definitions.```

To keep things as simple as possible, the echo server example will be reused and enhanced with another import parameter _resources_.

Before the jsonschema can be used in the blueprint, it has to be referenced in the component descriptor so that Landscaper knows about the dependency towards the `definitions` component.
Hence, a component reference entry is added to the echo server's component descriptor (see the complete component descriptor in the details):
```yaml
componentReferences:
- name: definitions
componentName: github.com/gardener/landscaper/external-jsonschema/definitions
version: v0.1.0
```

<details>
<div id="echo-server-comp-desc"></div>

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

</details>

Based on the extended component descriptor, the resource import can be added to the blueprint.
The import also contains a valid jsonschema but uses the `$ref` attribute with a custom landscaper protocol implementation.
That custom protocol specifies that the jsonschema should be fetched from the resource `resources-definition` of the referenced component `definitions`.

For detailed explanation about available reference methods see the [blueprint jsonschema documentation](../usage/JSONSchema.md)

```yaml
imports:
- name: resources
  schema:
    $ref: "cd://componentReferences/definitions/resources/resources-definition"
```

The blueprint is defined, so it can be added to the component descriptor using the component-cli and then uploaded.
See the details section for concrete resource definitions and cli calls.

<details>

```yaml
# ./docs/tutorials/resources/external-jsonschema/echo-server/blueprint.yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

imports:
- name: cluster
  type: target
  targetType: landscaper.gardener.cloud/kubernetes-cluster
- name: ingressClass
  type: data
  schema:
    type: string
- name: resources
  type: data
  schema:
    $ref: "cd://componentReferences/definitions/resources/resources-definition"

deployExecutions:
- name: default
  type: GoTemplate
  file: /defaultDeployExecution.yaml
```

```yaml
# ./docs/tutorials/resources/external-jsonschema/echo-server/defaultDeployExecution.yaml
{{ $name :=  "echo-server" }}
{{ $namespace :=  "default" }}
deployItems:
- name: deploy
  type: landscaper.gardener.cloud/kubernetes-manifest
  target:
    name: {{ .imports.cluster.metadata.name }}
    namespace: {{ .imports.cluster.metadata.namespace }}
  config:
    apiVersion: manifest.deployer.landscaper.gardener.cloud/v1alpha2
    kind: ProviderConfiguration

    updateStrategy: patch

    manifests:
    - policy: manage
      manifest:
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: {{ $name }}
          namespace: {{ $namespace }}
        spec:
          replicas: 1
          selector:
            matchLabels:
              app: echo-server
          template:
            metadata:
              labels:
                app: echo-server
            spec:
              containers:
                - image: {{ with (getResource .cd "name" "echo-server-image") }}{{ .access.imageReference }}{{end}}
                  imagePullPolicy: IfNotPresent
                  name: echo-server
                  args:
                  - -text="hello world"
                  ports:
                    - containerPort: 5678
                  resources:
{{ toYaml .imports.resources | indent 21 }}
    - policy: manage
      manifest:
        apiVersion: v1
        kind: Service
        metadata:
          name: {{ $name }}
          namespace: {{ $namespace }}
        spec:
          selector:
            app: echo-server
          ports:
          - protocol: TCP
            port: 80
            targetPort: 5678
      - apiVersion: networking.k8s.io/v1
        kind: Ingress
        metadata:
          name: {{ $name }}
          namespace: {{ $namespace }}
          annotations:
            nginx.ingress.kubernetes.io/rewrite-target: /
            kubernetes.io/ingress.class: "{{ .imports.ingressClass }}"
        spec:
          rules:
          - http:
              paths:
              - path: /
                pathType: Prefix
                backend:
                  service:
                    name: echo-server
                    port:
                      number: 80
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

</details>

### Installation

The components with their jsonschema and blueprint artifacts are now uploaded and can be deployed.

In a first step, the target import and the data imports have to be defined and applied to the Kubernetes cluster so that Landscaper can pick them up.

```yaml
# ./docs/tutorials/resources/external-jsonschema/my-target.yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Target
metadata:
  name: my-target-cluster
spec:
  type: landscaper.gardener.cloud/kubernetes-cluster
  config:
    kubeconfig: |
      apiVersion:...
      # here goes the kubeconfig of the target cluster
```

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
kubectl apply -f ./docs/tutorials/resources/external-jsonschema/my-target.yaml
kubectl apply -f ./docs/tutorials/resources/external-jsonschema/configmap.yaml
```

The imports are now available in the system, so the installation can be applied and will be processed by Landscaper.

```
# ./docs/tutorials/resources/external-jsonschema/installation.yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: my-echo-server
spec:
  componentDescriptor:
    ref:
      repositoryContext:
        type: ociRegistry
        baseUrl: eu.gcr.io/gardener-project/landscaper/tutorials/components
      componentName: github.com/gardener/landscaper/external-jsonscheme/echo-server
      version: v0.1.0

  blueprint:
    ref:
      resourceName: echo-server-blueprint

  imports:
    targets:
    - name: cluster
      # the "#" forces the landscaper to use the target with the name "my-cluster" in the same namespace
      target: "#my-cluster"
    data:
    - name: ingressClass
      configMapRef:
        key: ingressClass
        name: my-imports
    - name: resources
      configMapRef:
        key: resources
        name: my-imports
```

### Summary

- A component has been created that contains a jsonschema
- The echo server has been enhanced by another import that uses a jsonschema defined by another component
- With the external jsonschema it is now possible to reuse and share jsonschemas across components.

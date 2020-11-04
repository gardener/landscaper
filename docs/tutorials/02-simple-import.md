# Simple data import

This tutorial is build upon the first simple blueprint, therefore the [first tutorial](01-create-simple-blueprint.md) 
should be done before.

The goal of this tutorial is to deploy a http echo server (https://github.com/hashicorp/http-echo) that is exposed via an ingress.
It consumes the exported ingressClass value of the nginx ingress installation and use that annotation in the ingress to use the previously deployed ingress controller.

__Prerequisites__:
- Helm commandline tool (see https://helm.sh/docs/intro/install/)
- OCI compatible oci registry (e.g. GCR or Harbor)
- Kubernetes Cluster (better use two different clusters: one for the landscaper and one for the installation)
- [first tutorial](01-create-simple-blueprint.md)

All example resources can be found in [docs/tutorials/resources/echo-server](./resources/echo-server).

### Resources

#### Build the Blueprint

The http echo server consists of a deployment, a service and a ingress object that are istalled using the [kubernetes manifest deployer](/docs/deployer/manifest.md)

First resource that we have to create is the blueprint.<br>
The http echo blueprint imports a cluster to deploy the kubernetes resources and it also imports a ingress class
that is used to determine the responsible ingress controller.
(See the kubernetes [ingress docs](https://kubernetes.io/docs/concepts/services-networking/ingress/) for detailed documentation)

Then the deploy items are defined.
Again GoTemplate is used as the templating engine but the go template is not defined inline in the blueprint.
This time, the template is defined in a separate file to keep the blueprint clean and readable.

The external file is defined with the `file` attribute and points to the external file in the filesystem of the blueprint.
Whereas the root is the directory of the `blueprint.yaml`.

For detailed information about the template executors see [here](/docs/usage/TemplateExecutors.md).

*blueprint.yaml*:
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

imports:
- name: cluster
  targetType: landscaper.gardener.cloud/kubernetes-cluster
- name: ingressClass
  schema:
    type: string

deployExecutions:
- name: default
  type: GoTemplate
  file: /defaultDeployExecution.yaml
```

The external file contains the template to render the deploy items.
As the kubernetes manifest deployer is used to deploy the kubernetes object, one deploy item of type `landscaper.gardener.cloud/kubernetes-manifest` is defined.

It contains all the 3 resources that are needed for the echo server deployment.<br>
The imported `ingressClass` is used in the ingress resource to define the class annotation: `kubernetes.io/ingress.class: "{{ .imports.ingressClass }}"`.<br>
Also the http echo server oci image is taken from the component descriptor as external resource: `image: {{ index .cd.component.resources "echo-server-image" "access" "imageReference" }}`.

*defaultDeployExecution.yaml*:
```helmyaml
{{ $name :=  "echo-server" }}
{{ $namespace :=  "default" }}
deployItems:
- name: deploy
  type: landscaper.gardener.cloud/kubernetes-manifest
  target:
    name: {{ .imports.cluster.metadata.name }}
    namespace: {{ .imports.cluster.metadata.namespace }}
  config:
    apiVersion: manifest.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderConfiguration

    updateStrategy: patch

    manifests:
      - apiVersion: apps/v1
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
                - image: {{ index .cd.component.resources "echo-server-image" "access" "imageReference" }}
                  imagePullPolicy: IfNotPresent
                  name: echo-server
                  args:
                  - -text="hello world"
                  ports:
                    - containerPort: 5678
      - apiVersion: v1
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
      - apiVersion: networking.k8s.io/v1beta1
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
                  serviceName: echo-server
                  servicePort: 80
```

Upload the blueprint into the oci registry.
```shell script
landscaper-cli blueprints push myregistry/mypath/echo-server:v0.1.0 docs/tutorials/resources/echo-server/blueprint
```

#### Build the Component Descriptor

The blueprint is now build and uploaded.
Then the corresponding component descriptor has to be created.

It contains the blueprint as local resource and the http echo server image as external resource.
The echo server is specified as external image because the image is consumed form the open source.<br>
For more information about the component descriptor and the usage of the different fields see the [component descriptor docs](https://github.com/gardener/component-spec).

```yaml
meta:
  schemaVersion: v2

component:
  name: github.com/gardener/landscaper/echo-server
  version: v0.1.0

  provider: internal

  repositoryContexts:
  - type: ociRegistry
    baseUrl: eu.gcr.io/myproject

  resources:
  - type: blueprint
    name: echo-server-blueprint
    relation: local
    access:
      type: ociRegistry
      imageReference: myregistry/mypath/echo-server:v0.1.0
  - type: ociImage
    name: echo-server-image
    version: v0.2.3
    relation: external
    access:
      type: ociRegistry
      imageReference: hashicorp/http-echo:0.2.3
```

```shell script
landscaper-cli cd push docs/tutorials/resources/echo-server/component-descriptor.yaml
```

### Installation

The same target as in the first tutorial is used as the resources have to be deployed into the same kubernetes cluster.
The only resource that has to be defined is a Installation for the echo-server blueprint.

The echo-server installation is the same as it was previously created for the nginx ingress blueprint.

In addition, the `ingressClass` import has to be defined.
The nginx installation exports ìts ingressClass to `myIngressClass`, so this dataobject has to be used as import for the echo server.

```yaml
imports:
  data:
  - name: ingressClass
    dataRef: "myIngressClass"
```

*Complete Installation*:
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: my-echo-server
spec:
  blueprint:
    ref:
      repositoryContext:
        type: ociRegistry
        baseUrl: eu.gcr.io/my-project/comp
      componentName: github.com/gardener/landscaper/echo-server
      version: v0.1.0
      kind: localResource
      resourceName: echo-server-blueprint

  imports:
    targets:
    - name: cluster
      # the "#" forces the landscaper to use the target with the name "my-cluster" in the same namespace
      target: "#my-cluster"
    data:
    - name: ingressClass
      dataRef: "myIngressClass"
```

The echo-server can now be installed by applying the installation to the landscaper cluster.
```shell script
kubectl create -f docs/tutorials/resources/echo-server/installation.yaml
```

### Summary

- A blueprint that describes the deployment of an echo server deployment and imports data from another blueprint has been development

### Up next

In the [next tutorial](./03-aggregated-blueprint.md), a aggregated blueprint will be developed.

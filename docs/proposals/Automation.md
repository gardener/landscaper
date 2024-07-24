

Todo:
- Values
- multiple items
- Import/Export
- Create Targets (only provide path to kubeconfig)
- image pull secrets (use some tags in yaml)
- secrets for chart registry etc.
- images (use some tag such that it is added to the component descriptor)
- add other features
- with and without components 
- think about storage of all artefacts in k8s cluster


Create all artefacts/scripts for creating/uploading a component with a helm chart and an Installation/Target/Secrets
which get additional values as input as well as the namespace (and name) of the helm chart. 

```yaml
component:
  name: someName
  version: someVersion
items:
  - name: myName1
    type: helm
    createNamespace: true
    chart:
      access:
        type: ociArtifact
        imageReference: eu.gcr.io/gardener-project/landscaper/examples/charts/guided-tour/echo-server:1.0.0
    images:
      - name: echo-server-image
        type: ociImage
        version: v0.2.3
        access:
          type: ociArtifact
          imageReference: hashicorp/http-echo:0.2.3
    predefinedValues:
      valName1: ttt
      sub1:
        sub2: (($images.echo-server-image))
```
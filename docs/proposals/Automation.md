

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

Questions:
- should we always create subinsts such that later data exchange is possible


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
      type: helmChart
      version: 1.0.0
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
        sub2: (($images.echo-server-image.access.imageReference))
```

ls-cli component create path-to-config-yaml path-to-output-dir:
  - creates the files required to add and upload a component
  - does this include some scripts

ls-cli component upload path-to-input-dir path-to-settings (repo-base-url) 
  - add and upload component 
  - perhaps use just the two ocm commands

ls-cli component installation path-to-input-dir path-to-values path-to-config (instname, instnamespace, targetname)
  - creates installation yaml


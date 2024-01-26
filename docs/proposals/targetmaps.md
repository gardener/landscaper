# Target Maps

## Syntax

### Declaration of a TargetMap Import in a Blueprint 

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

imports:
  - name: clusters
    type: targetMap
    targetType: landscaper.gardener.cloud/kubernetes-cluster
```

### Providing a TargetMap Value in an Installation

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation

spec:
  imports:
    targets:
      - name: clusters
        targetMap:
          red: red-cluster
          green: green-cluster
          blue: blue-cluster
```

### Providing a TargetMap Value in an Installation by Reference

Suppose two blueprints A and B each have a TargetMap import parameter with name `targetMapA`, resp. `targetMapB`.
Suppose blueprint A has a subinstallation template which calls blueprint B.
A can forward its own TargetMap to B in the following way:

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: InstallationTemplate

imports:
  targets:
    - name: targetMapB          # name of the import of the called blueprint B
      targetMapRef: targetMapA  # value for the import parameter targetMapB, 
                                # namely the forwarded import of the calling blueprint A.
```

### Using a TargetMap Inside a Blueprint's DeployExecution

```yaml
deployItems:
{{ range $key, $target := .imports.clusters }}
  - name: item-{{ $key }}
    type: landscaper.gardener.cloud/kubernetes-manifest

    target:
      import: clusters
      key: {{ $key }}

    config:
      ...
{{ end }}
```

## Implementation

### Reading a TargetMap from the Upper Context

During the Init phase of an Installation, the controller reads the imported TargetMaps from the "upper context", i.e.
the context of the parent installation, or in case of a root installation from the root context.  
See: [LoadImports](../../pkg/landscaper/installations/imports/constructor.go)

### Writing a TargetMap into the Lower Context

Later during the Init phase, the controller writes the imported TargetMaps into the "lower context", i.e. the context of
the present Installation.  
See: [createOrUpdateImports](../../pkg/landscaper/installations/operation.go)

Each Target gets the following labels:

- `DataObjectContextLabel`: the (lower) context of the Installation "Inst.<...>", i.e. the context into which the Target is written.
- `DataObjectKeyLabel`: name of the TargetMap import parameter of the blueprint
- `DataObjectSourceTypeLabel`: always import, because we do not support the export of TargetMaps
- `DataObjectSourceLabel`: 
- `DataObjectTargetMapKeyLabel`: the key of the Target in the TargetMap
- `DataObjectJobIDLabel`: the job ID of the Installation

Annotations:

- `DataObjectHashAnnotation`

### Exporting TargetMaps

Not supported

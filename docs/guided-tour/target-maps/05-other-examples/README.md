# Other Target Map Examples

## Target Map on Subinstallation Level

The example defined [here](../04-forward-map) deploys the same config maps on the target cluster as the others. The
difference is that it imports 3 targets as standard targets and in this 
[Subinstallation](../04-forward-map/component/blueprint/sub/subinst.yaml) it converts these into a target map.

## Iterate over Config

In all examples about target maps so far, the iteration was done over the targets. Of course, it would have been also
possible to iterate over the configuration data of the imported data object `config`.

## Importing Target with a Target Map from Siblings

It is also possible that different Targets from sibling Subinstallations can be imported by a Subinstallation
using a Target Map. 

The following example demonstrates the principal syntax. 

```
subinstallations:
{{ $rootconfig := .imports.rootconfig }}
{{ range $key, $instanceConfig := $rootconfig }}
  - apiVersion: landscaper.gardener.cloud/v1alpha1
    kind: InstallationTemplate

    name: compose-exports-exporter-{{ $key }}

    blueprint:
      ref: cd://resources/blueprint-exporter

    imports:
      targets:
        - name: clusterIn
          target: rootcluster

    exports:
      targets:
        - name: clusterOut
          target: cluster-{{ $key }}
{{ end }}

  - apiVersion: landscaper.gardener.cloud/v1alpha1
    kind: InstallationTemplate

    name: compose-exports-importer

    blueprint:
      ref: cd://resources/blueprint-importer

    imports:
      targets:
        - name: clusters
          targetMap:
            {{ range $key, $instanceConfig := $rootconfig }}
              {{ $key }}: cluster-{{ $key }}
            {{- end }}
      data:
        - name: config
          dataRef: rootconfig
```

In the first loop over `$rootconfig` a Subinstallation `compose-exports-exporter-{{ $key }}` is created for every 
entry. Each of these Subinstallations exports a target `cluster-{{ $key }}`. 

The Subinstallation `compose-exports-importer` imports all exported targets as a targetMap:

```
    imports:
      targets:
        - name: clusters
          targetMap:
            {{ range $key, $instanceConfig := $rootconfig }}
              {{ $key }}: cluster-{{ $key }}
            {{- end }}
```
# Installations

Installations are instances of `Definitions` inside a cluster with a specific context.
This context consists of the import/export mapping and optional static configuration.


Basic structure:
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: my-installation
spec:

  definitionRef: my-def:1.0.0

  imports: # generated from aggregated definition or default from definition with from = to
  - from: current-context.namespace
    to: namespace

  exports:
  - from: ingressclass
    to: current-context.ingressClass

  staticData:
  - value: # value must contain a map of values
      current-context: 
        namespace: my-ns

status:
  phase: Progressing | Pending | Deleting | Completed | Failed

  imports:
  - from: current-context.namespace
    to: namespace
    configGeneration: 0
  
  # Reference to the execution of the installation which is templated
  # based on the ComponentDefinition (.spec.definitionRef).
  executionRef:  
    name: my-execution
    namespace: default
  
  # References to subinstallations that were automatically created 
  # based on the ComponentDefinition (.spec.definitionRef).
  installationRefs: 
  - name: my-sub-component
    ref:
      name: my-sub-component
      namespace: default

  configGeneration: 0
  exportRef: 
    name: my-exports
    namespace: default
  observedGeneration: 0

```


### Specify static data

Imports of installation can be satisfied if they are either
- exported by sibling installations or
- statically specified in `.spec.staticData`.

Static data can be specified by providing multiple data sources as described below.
All configured data sources are loaded and merged during runtime. <br>
The resulting merged data is then used to satisfy the imports by using the import key as a jsonpath to this merged data.
The resulting value is then validated against the specified DataType of the key.

Static data that satisfies an import is always preferred over exported data from other installations.

```yaml
  staticData:
  - value: # value must contain a map of values
      configkey1: val1
  - valueFrom:
      secretKeyRef:
        name: mysecret
        key: key1 # default to "config"; the value must contain a map of values
  - valueFrom:
      secretLabelSelector:
        selector:
          my-app: mysecret-label
        key: key1 # default to "config"; the value must contain a map of values
```
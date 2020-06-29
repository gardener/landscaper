# Component Definitions

A Component Definition describes the steps that are necessary to deploy a component. These steps are described by either Executors or references to other Component Definitions.

## Executors and Aggregations

### Executors

A Definition may contain any number of Executors. They are provided as one single string in `.executors`. While processing the Definition, the string is templated, after which it should be a valid YAML list, and then stored in the cluster as an `Execution` CR. Executors are processed in the given order, but in parallel with referenced Definitions. See the [Executor documentation](Executors.md) for further information.

```yaml
executors: |
- name: deploy-chart
  type: helm
  config:
    chartRepository: my-repo
    version: 1.0.0
    values: {{ .exports.mykey.x }}
    valueRef:
      secretRef: abc
```
*Example*


### Aggregations

A Definition can aggregate any number of other Definitions by referencing them in `.definitionsRefs`.
To map the imports of the surrounding Definition to their inner definitions, a mapping is needed.
The mapping can be defined for each component for their imports and exports.

```yaml
definitionRefs:
- ref: my-sub-component:1.0.0
  imports:
  - from: abc
    to: yxz
  exports:
  - from: abc
    to: yxz
```

*Example*


## Installations

Component Definitions are not deployed into the cluster. Instead, a Component Installation is deployed which references the corresponding Definition. If the referenced Definition aggregates other definitions, their corresponding Installations will be created automatically, the user only needs to deploy the top-level Installation(s). See the [documentation on Installations](Installations.md) for details.
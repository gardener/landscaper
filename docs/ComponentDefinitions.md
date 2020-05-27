# Component Definitions

### Aggregations

Aggregations can be created by referencing other components in `.spec.definitionsRefs`.
To map the imports of the surrounding Definition to their inner definitions, a mapping is needed.
The mapping can be defined for each component for their imports and exports.

```yaml
- ref: my-sub-component:1.0.0
  imports:
  - from: abc
    to: yxz
  exports:
  - from: abc
    to: yxz
```

*Example*


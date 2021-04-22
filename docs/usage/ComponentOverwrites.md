# Component Overwrites

When deploying applications using aggregated blueprints, their component descriptors reference multiple other components with blueprints.
In order to test new versions of transitive dependencies one would have to release new versions of all including component descriptors which could, depending on the depth, result in a lot of releases and therefore repetitive overhead.

To support simpler testing of development versions in aggregated components it should be possible to override specific blueprints or components manually in a live system.
Therefore a new clusterwide resource is introduced called `ComponentOverwrites`.

```yaml
apiVersion: landscaper.gardener.com/v1alpha1
kind: ComponentOverwrite
metadata:
  name: my-overwrite

overwrites:
- component:
    repositoryContext: # optional
      type: ociRegistry
      baseUrl: "example.com"
    componentName: ""
    version: "" # optional
  target:
    repositoryContext: # optional
      type: ociRegistry
      baseUrl: "example.com"
    componentName: "" # optional
    version: ""
```

# Optimization

This chapter contains some hints to improve the performance of Landscaper instances.

- Do not create too many Installations, Executions, DeployItems, Targets etc. in one namespace watched by your 
  Landscaper instance. A reasonable upper bound is about 500 objects for every object type. If you have more
  objects, spread them over more than one namespace.

- If you know that an installation does not import/exports data from/to sibling installations or has no 
  siblings at all, you could specify this in the `spec` of an installation as follows. If nothing set, the default
  value `false` is assumed. This hint prevents the need for complex dependency computation and speads up processing. 

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  ...
spec:
  ...
  optimization:
    # set this on true if the installation does not import data from its siblings or has no siblings at all
    hasNoSiblingImports: true/false
    # set this on true if the installation does not export data to its siblings or has no siblings at all
    hasNoSiblingExports: true/false 
  ...
```
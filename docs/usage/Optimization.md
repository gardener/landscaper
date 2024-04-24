---
title: Landscaper Optimization
sidebar_position: 18
---

# Optimization

This chapter contains some hints to improve the performance of Landscaper instances.

- Do not create too many Installations, Executions, DeployItems, Targets etc. in one namespace watched by your 
  Landscaper instance. A reasonable upper bound is about 200 objects for every object type. If you have more
  objects, spread them over more than one namespace.

- It is possible to cache helm chart with the annotation `landscaper.gardener.cloud/cache-helm-charts: "true"`
  ([see](./Annotations.md#cache-helm-charts-annotation))

- If you know that an installation does not import/exports data from/to sibling installations or has no 
  siblings at all, you could specify this in the `spec` of an installation as follows. If nothing set, the default
  value `false` is assumed. This hint prevents the need for complex dependency computation and speeds up processing. 
  Only use this feature, if you are sure about the data exchange of your Installations because if this is enabled and 
  siblings are exchanging data, this might produce erratic results. If you could enable this feature for all of your 
  installations in a namespace, a reasonable upper limit for the number of objects of this namespace is 500 for every 
  object type.

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

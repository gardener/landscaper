# Component Overwrites

When deploying applications using aggregated blueprints, their component descriptors reference multiple other components with blueprints.
In order to test new versions of transitive dependencies one would have to release new versions of all including component descriptors which could, depending on the depth, result in a lot of releases and therefore repetitive overhead.

To support simpler testing of development versions in aggregated components it should be possible to override specific blueprints or components manually in a live system.


Therefore, two new resources are introduced:
- [`ComponentVersionOverwrites`](#componentversionoverwrites)
- [`ComponentOverwrites`](#componentoverwrites) *(deprecated)*

Whenever the component reference of an Installation, regardless whether used directly or indirectly, it will be matter of substitution for described component overwrites. This means that not only the component descriptor reference used in an Installation is replaces, but also all of the references within the - potentially replaced - component descriptor.

## Overwrite Evaluation

Every [context](./Context.md) has an assigned component overwrite list where each entry describes a component version to overwrite and an overwrite target. Both elements feature the following structure:
- **`componentName`** *string* (optional)
- **`version`** *string* (optional)
- **`repositoryContext`** *[structure](./RepositoryContext.md)* (optional)

If used to match a replacement source, unspecified attributes match any value and specified attributes describe a concrete match.
If used to describe a replacement target, specified attributes replace the respective attribute in the source, while unspecified attributes are left unchanged.

A list of overwrite specifications is evaluated in the given order. If a source specification matches and none of the given substitution attributes have already been substituted by an earlier match, the substitution is executed.

This means that more specific overwrite specifications should be placed before more generic ones and if a substitution already replaced an attribute, more general substitutions later on, affecting the same field, will be ignored.

## ComponentVersionOverwrites

In the namespace, a [Context](./Context.md) object might be accompied by a component version overwrite object, describing desired overwrites applicable for this context. The component version overwrite object must have the same name as the Context object.

If an overwrite object is described for a context, the cluster-wide component overwrites are ignored.

**Example**
```yaml
apiVersion: landscaper.gardener.com/v1alpha1
kind: ComponentVersionOverwrite
metadata:
  name: my-context
  namespace: my-namespace

overwrites:
- source:
    repositoryContext:
      type: ociRegistry
      baseUrl: "example.com"
    componentName: ""
    version: ""
  substitution:
    repositoryContext:
      type: ociRegistry
      baseUrl: "example.com"
    componentName: ""
    version: ""
```

## ComponentOverwrites

A component overwrite object is a cluster-wide specification of component version overwrites. There might be any number of such overwrite objects. The described overwrite specifications are concatenated in the inversed order of the creation timestamps of the resources. This means that the most recent overwrite specifications will be executed before the older ones.

**Example**
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

# Component Overwrites

When deploying applications using aggregated blueprints, their component descriptors reference multiple other components with blueprints.
In order to test new versions of transitive dependencies one would have to release new versions of all included component descriptors which could, depending on the depth, result in a lot of releases and therefore repetitive overhead.

To support simpler testing of development versions in aggregated components it should be possible to override specific blueprints or components manually in a live system.


Therefore, a new resource is introduced: [`ComponentVersionOverwrites`](#componentversionoverwrites)

Whenever the component references an Installation, regardless of whether it is used directly or indirectly, it will be matter of substitution for described component overwrites. This means that not only the component descriptor reference used in an Installation is replaced, but also all of the references within the - potentially replaced - component descriptor.

## Overwrite Evaluation

Every [context](./Context.md) has an assigned component overwrite list where each entry describes a component version to overwrite and an overwrite target. Both elements feature the following structure:
- **`componentName`** *string* (optional)
- **`version`** *string* (optional)
- **`repositoryContext`** *[structure](./RepositoryContext.md)* (optional)

If used to match a replacement source, unspecified attributes match any value and specified attributes describe a concrete match.
If used to describe a replacement target, specified attributes replace the respective attribute in the source, while unspecified attributes are left unchanged.

A list of overwrite specifications is evaluated in the given order. If a source specification matches and none of the given substitution attributes have already been substituted by an earlier match, the substitution is executed. No attribute will ever be overwritten twice and overwrites are applied either whole or not at all, but not partially.

This means that more specific overwrite specifications should be placed before more generic ones and if a substitution already replaced an attribute, more general substitutions later on, affecting the same field, will be ignored.

## ComponentVersionOverwrites

In the namespace, a [Context](./Context.md) object might be accompied by a `ComponentVersionOverwrite` object, describing desired overwrites applicable for this context. For a `ComponentVersionOverwrite` object to take effect, it has to be referenced in the `Context` which is used for the installation.

**Example**

Below is part of an Installation resource which belongs to the echo-server from the tutorial. It uses the `default` context (which is the default, if not specified otherwise in the Installation spec).

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: server
  namespace: my-namespace
spec:
  blueprint:
    ref:
      resourceName: echo-server-blueprint
  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper/echo-server
      repositoryContext:
        baseUrl: eu.gcr.io/gardener-project/landscaper/tutorials/components
        type: ociRegistry
      version: v0.2.0
  context: default
  ...
```

The `default` context references a `ComponentVersionOverwrites` object named `my-overwrites`. This reference is optional and no overwrites will happen if it is missing.

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Context
metadata:
  name: default
  namespace: my-namespace
componentVersionOverwrites: my-overwrites
```

The below `ComponentVersionOverwrites` object defines two overwrites.

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: ComponentVersionOverwrites
metadata:
  name: my-overwrites
  namespace: my-namespace
overwrites:
- source:
    repositoryContext:
      baseUrl: eu.gcr.io/gardener-project/landscaper/tutorials/components
      type: ociRegistry
    componentName: github.com/gardener/landscaper/echo-server
  substitution:
    repositoryContext:
      baseUrl: "example.org/my-own-registry/components"
      type: ociRegistry
    componentName: my-own-echo-server
- source:
    componentName: github.com/gardener/landscaper/echo-server
  substitution:
    componentName: another-echo-server
    version: v1.2.3
```

The first overwrite matches all components named `github.com/gardener/landscaper/echo-server` from the OCI repository at `eu.gcr.io/gardener-project/landscaper/tutorials/components` and replaces the repository as well as the name. The second overwrite matches all components named `github.com/gardener/landscaper/echo-server` and overwrites their name and version, without changing the repository context.

This combination of resources results in the following condition on the Installation:
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: server
  namespace: my-namespace
spec:
  blueprint:
    ref:
      resourceName: echo-server-blueprint
  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper/echo-server
      repositoryContext:
        baseUrl: eu.gcr.io/gardener-project/landscaper/tutorials/components
        type: ociRegistry
      version: v0.2.0
  context: default
  ...
status:
  conditions:
  - lastTransitionTime: ...
    lastUpdateTime: ...
    message: |-
      Component reference has been overwritten:
      eu.gcr.io/gardener-project/landscaper/tutorials/components () -> example.org/my-own-registry/components ()
      github.com/gardener/landscaper/echo-server -> my-own-echo-server
      Version has not been overwritten
    reason: FoundOverwrite
    status: "True"
    type: ComponentReferenceOverwrite
  ...
```

While the component descriptor reference in the Installation spec still shows the original reference, the status shows that it has been overwritten and the Landscaper will actually use the overwritten component reference.

Note that the version has not been overwritten, despite the second overwrite matching the name of the component. The reason for this is that the second overwrite overwrites the name and the version, but the name has already been overwritten by the first overwrite. Therefore, the second overwrite is ignored. Had it only changed the version and not the name, then it would have taken effect.
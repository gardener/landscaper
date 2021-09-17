# Deployer Extensions

This article is meant to list already implemented extensions which can be integrated in deployers. For more information on how deployer extensions work, read about the [Deployer Library Extension Hooks](./dep-lib-extension-hooks.md).

#### Index
- [Continuous Reconcile Extension](#continuous-reconcile-extension)


### Continuous Reconcile Extension

##### What It Does
The continuous reconcile extension allows to reconcile a deploy item regularly at specific points in time or at a fixed interval. This is for example useful if one wants to re-deploy a deploy item every hour to overwrite any changes a user has potentially done to the resources created by the deploy item. While this feature might be desired for some types of deploy items, it might not make any sense for others, therefore it is not implemented directly in the deployer library, but in an extension instead.

##### Package
github.com/gardener/landscaper/pkg/deployer/lib/continuousreconcile

##### The Hook
The continuous reconcile extension is a hook of type `ShouldReconcile`. To get the complete hook setup, call `continuousreconcile.ContinuousReconcileExtensionSetup`, to get only the hook `continuousreconcile.ContinuousReconcileExtension` can be used. 

##### Required Implementation
Both functions mentioned above take a function with the signature `func(context.Context, time.Time, *lsv1alpha1.DeployItem) (*time.Time, error)` as argument. The function takes - apart from the context - a time and a pointer to a deploy item and is expected to return the next point in time after the given time when the deploy item should be reconciled again. If there is no such point in time - e.g. because continuous reconciliation is disabled for that deploy item - the function can return `nil`.
How exactly the function works depends on the deployer. In the mock deployer, it reads the configuration for continuous reconciliation from the given deploy item and then returns the next time that matches the specification, or nil, if this configuration is missing or empty in the deploy item. This is probably the most straight-forward implementation when per-deploy-item configuration is desired.
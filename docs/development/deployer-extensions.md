# Deployer Extensions

After switching to the new reconcile logic the continuous reconcile extension is currently deactivated. It is open if, how and
when it is activated again.

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

##### Usage
The continuous reconcile extension has been added to the `mock`, `helm`, `manifest`, and `container` deployers.
The configuration is the same for all deployers: add this node to the config of your deploy item
```yaml
continuousReconcile:
  every: "1h" # OR
  cron: "* */1 * * *"
```
Example:
This deployItem is configured in such a way, that reconciliation is triggered every 1 minutes. If a user deletes what is deployed by the deployItem, e.g. by manually deleting it via kubectl delete, then after 1m the deployItem is retriggered and therefor restores what has been removed.
```yaml
deployItems:
  - name: default-deploy-item
    type: landscaper.gardener.cloud/kubernetes-manifest
    target:
      name: {{ .imports.cluster.metadata.name }}
      namespace: {{ .imports.cluster.metadata.namespace }}
    config:
      apiVersion: manifest.deployer.landscaper.gardener.cloud/v1alpha2
      kind: ProviderConfiguration
      updateStrategy: update
      continuousReconcile:
        every: "1m" 
      manifests:
        - policy: manage
          manifest:
            apiVersion: v1
            kind: ConfigMap
            metadata:
              name: test
              namespace: example
            data:
              foo: bar

      exports:
        defaultTimeout: 2m
        exports:
          - key: test
            jsonPath: .data
            fromResource:
              apiVersion: v1
              kind: ConfigMap
              name: test
              namespace: example
```

In order to enable continuous reconciliation for a deploy item, you have to configure either an interval or a cron schedule.

The first one is done by setting `continuousReconcile.every` to the desired interval. The value is expected to be parseable by `time.ParseDuration`. This method uses the `lastReconcileTime` information from the deploy item status to determine when the last reconciliation happened and requeues the deploy item when this matches or exceeds the configured interval. Please note that `lastReconcileTime` is set at the beginning of the reconciliation process and it is only updated if the reconciliation is not being aborted (which usually happens if nothing has changed since the last one). This means that the extension actually enforces 'full' reconciliations at least once per configured interval, but it will only do so if there hasn't been any regular 'full' reconciliation within the last interval (happens usually if the deploy item or its imports change). Furthermore, if you have long-running deploy items with too short delays between the reconciliations, a deploy item might be re-triggered before having handled the last activation, which could lead to unexpected behaviour. It is recommended to choose the delay large enough so this doesn't happen.

For better control on when and how often the deploy item is reconciled, it is also possible to provide a cron schedule via `continuousReconcile.cron`. Apart from standard cron specification, some keywords like `@daily` are also supported. Please note that the cron schedule specifies reconciliation at certain times rather than after certain intervals, so unlike for the `every` option, this could cause additional reconciliations directly after regular ones (for example: a cron schedule which will cause a reconciliation every day at 8am will trigger it even if the deploy item has been changed - and thus reconciled - at 7:59am).

Specifying both `every` and `cron` will lead to a validation error (except for the `mock` deploy item, where the config is not validated, here the `cron` will take precedence).

To temporarily disable continuous reconciliation without changing the spec of the deploy item, the annotation `continuousreconcile.extensions.landscaper.gardener.cloud/active: "false"` can be used.

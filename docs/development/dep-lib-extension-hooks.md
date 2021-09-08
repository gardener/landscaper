# Deployer Library Extension Hooks

No matter what exactly a deployer is meant to do, the landscaper has some expectations regarding the handling of deploy items - the deployer is expected to remove certain annotations and react on them, reconcile only deploy items which it is responsible for, [and so on](../technical/deployer_contract.md). To avoid lots of duplicate code and reduce the risk of errors in the implementation of this contract, there is a deployer library which implements most of the boilerplate code and basically simplifies the process of writing a deployer to a little more than implementing an interface.

However, in some cases, the default reconciliation flow implemented by the deployer library is too restrictive for what a deployer wants to do.
Let's have a look at an example scenario:
I have a kubernetes manifest deploy item which creates a deployment in my cluster for, let's say, logging. After creating the deploy item, the manifest deployer will apply the contained manifest and create my logging deployment - everything is fine. Now a user edits the replicaset belonging to the deployment and scales it to zero - all my logging pods are gone. And since the replicaset is only created indirectly, the manifest deployer would not notice this change, even if it was watching the resources it had created (which it doesn't do). The deployment will only be redeployed if the deploy item is either changed or triggered manually - and until then, my logging doesn't work.

In this scenario, it would be desirable to re-apply the manifest after a fixed timespan in order to overwrite any manual changes to the deployed resources, but the default reconciliation logic of the deployer library is to abort the reconcile if nothing relevant has changed in the deploy item.

To be able to still use the library in cases like these, the concept of extension hooks was introduced.


## Extension Hooks and Hook Results

The idea behind the extension hooks is to enable modification of the deployer library's default reconciliation logic by injecting custom code.
This custom code comes in form of a 'hook', which is basically just a function with a specific signature.
```golang
// ReconcileExtensionHook represents a function which will be called when the hook is executed.
type ReconcileExtensionHook func(context.Context, logr.Logger, *lsv1alpha1.DeployItem, *lsv1alpha1.Target, HookType) (*HookResult, error)
```

The deployer library's `Reconcile` method offers several places where such hooks can be executed. In addition to the effects of the executed code, a hook may modify the reconciliation flow by returning a `HookResult`. It can abort the reconciliation as well as modify the result returned by it, the latter of which enables it to requeue the deploy item for another reconciliation, either immediately or after a given time.
```golang
// HookResult represents the result of a reconciliation extension hook.
type HookResult struct {
	// ReconcileResult will be returned by the reconcile function if no error occurs.
	ReconcileResult reconcile.Result

	// If set to true, reconciliation will be aborted with returning ReconcileResult after the current execution.
	AbortReconcile bool
}
```

In case of an error, the hook result will be ignored and an empty `reconcile.Result` will be returned together with the error.

### Aggregating Hook Results

The hook result determines the result of the `Reconcile` function, as well as whether to continue or abort the reconciliation. However, each hook function produces its own hook result, so multiple hook results need to be aggregated into a single one. The reconcile method will start with a 'default' hook result and every time hooks are executed, they will return a single aggregated hook result which is then aggregated to the current hook result.
These aggregations happen according to the following rules:
- If all hook results are `nil`, the result will be `nil`.
- If exactly one hook result is non-nil, the result will be a copy of this hook result.
- Otherwise:
  - `AbortReconcile` is ORed and will thus result to `true` if any of the hook results has it set to this.
  - `ReconcileResult.Requeue` will also be ORed.
  - `ReconcileResult.RequeueAfter` will be set to the smallest value greater than zero which is found among the hook results.
    - If `ReconcileResult.Requeue` is `true`, it will be set to zero instead to enforce an immediate requeue.


## Default Reconciliation Flow

In order to understand how to modify the default reconciliation flow, let's have a look at what the deployer library's `Reconcile` method does:

1. **Fetch the deploy item from the cluster** - The method only gets name and namespace as arguments, so it first needs to fetch the corresponding deploy item from the cluster.
2. **Check responsibility** - Each deployer watches all deploy items, but it is only responsible for those which have its type and are in the environment which it is supposed to handle. For example, the helm deployer should not react on container deploy items - this is the job of the container deployer. It is therefore checked whether the deployer is actually responsible for the deploy item and the reconcile is aborted if not.
3. **Handle deployer contract** - This includes removing certain annotations and checking whether the deploy item has been changed since the last reconcile. Notably, if the deploy item has been changed, its `LastReconcileTime` timestamp will be set and its `Phase` will be set to `Init` at this point.
4. **Should reconcile check** - This check evaluates the outcome of step 3 - if the deploy item is in a final state (`Succeeded` or `Failed`), the reconcile will be aborted.
5. Exactly one of the following will be executed, depending on annotations and the state of the deploy item. All of these behave similarly in that they basically just call the respective deployer-specific method.
5.a **Abort** - If the deploy item has the abort annotation, it will be aborted. Please note that this differs from _aborting the reconciliation_ as mentioned above, because in this case, not the reconciliation is aborted, but the _processing of the deploy item_.
5.b **Force-Reconcile** - If the deploy item has the force-reconcile annotation, the corresponding method of the deployer will be called.
5.c **Deletion** - If the deletion timestamp on the deploy item exists, the deployer's `Delete` method will be called.
5.d **Reconcile** - Otherwise, the deployer's `Reconcile` method will be called.


## Hook Types

Extension hooks can be registered at several points during the aforementioned reconciliation flow. When to execute a hook is codified as 'hook type' - when a hook function is registered at a deployer, it will be given one or more of these types to determine when to be called.

**Start** hooks will be executed at the very beginning of the reconciliation flow. The deploy item and target which are given to the hook function will always be nil for these hooks.

**DuringResponsibilityCheck** hooks can influence the outcome of the 'check responsibility' step. If a non-nil hook result is returned, its `AbortReconcile` value will overwrite the result of the responsibility check - if it is `false`, the deployer will be considered responsible and continue the reconciliation, if it is `true` the reconciliation will be aborted independently of the result of the responsibility check. Using this hook in the wrong way can lead to unexpected behaviour among _all_ deploy items, so use with caution.

**AfterResponsibilityCheck** hooks are executed after it has been determined that the deployer is responsible for the deploy item (otherwise the reconciliation would have been aborted by now).

**ShouldReconcile** hooks behave similarly to the `DuringResponsibilityCheck` hooks in that they can influence the result of a check. A non-nil hook result with `AbortReconcile` set to `false` will enforce a reconcile, even if it is not required by the default logic. Please note that, apart from the other hooks, setting `AbortReconcile` to `true` will not always abort the reconciliation: if the default logic deems a reconcile necessary (e.g. due to changes in the deploy item), reconciliation cannot be stopped at this point. Since step 3 mentioned above will already have set the `Phase` and `LastReconcileTime`, aborting the reconciliation now would not only leave the deploy item in an inconsistent state, but also heavily meddle with the landscaper logic.

**BeforeAnyReconcile** hooks are executed before any of the mentioned `Abort`, `Force-Reconcile`, `Delete`, or `Reconcile` steps, while **BeforeAbort**, **BeforeForceReconcile**, **BeforeDelete**, and **BeforeReconcile** are only executed before their respective step (meaning only one of these hook types is executed each time). While it is possible to abort the reconciliation by setting the hook result's `AbortReconcile` to `true`, this will have the same consequences as mentioned above and should only be done with extreme caution.

Hooks with type **End** are executed at the end of the reconciliation, after the deployer-specific code has returned.


## Registering Hooks and Implementing the Interface

To enable the extension hooks, an additional method has been added to the `Deployer` interface.
```golang
// ExtensionHooks returns all registered extension hooks.
ExtensionHooks() extension.ReconcileExtensionHooks
```

This method has to be implemented so that it returns all registered hooks as a mapping from hook types to lists of hook functions.
```golang
// ReconcileExtensionHooks maps hook types to a list of hook functions.
type ReconcileExtensionHooks map[HookType][]ReconcileExtensionHook
```

The easiest way of implementing this is just adding said map to the deployer struct and returning it in the `ExtensionHooks` method, see e.g. the mock deployer for an example:
```golang
type deployer struct {
	log        logr.Logger
	lsClient   client.Client
	hostClient client.Client
	config     mockv1alpha1.Configuration
	hooks      extension.ReconcileExtensionHooks
}

func (d *deployer) ExtensionHooks() extension.ReconcileExtensionHooks {
	return d.hooks
}
```

The extension library offers a helper method to register a new hook:
```golang
// RegisterHook appends the given hook function to the list of hook functions for all given hook types.
// It returns the ReconcileExtensionHooks object it is called on for chaining.
func (hooks ReconcileExtensionHooks) RegisterHook(hook ReconcileExtensionHook, hypes ...HookType) ReconcileExtensionHooks
```
The given hook will be registered for all provided hook types. If the method is called without hook types, it won't have any effect.

> Hooks are meant to be registered during the creation of the deployer. Registering new hooks to an already running deployer is not supported and might or might not work.


### Hook Setups

With the auxiliary method mentioned above, registering a hook might look like this:
```golang
myDeployer.hooks.RegisterHook(myHookFunction, extension.StartHoo, extension.EndHook)
```

There is a small disadvantage though: when registering the hook, one has to know which hook types it was designed for.
While some hooks - e.g. a hook which just adds more logging - can probably be used in combination with any hook types, others are specifically designed for certain hook types and won't work when called at any other point in the reconciliation flow.

To simplify the registration, whoever writes a hook function has the possibility of bundling the function together with the fitting hook types into a hook setup struct.
```golang
// ReconcileExtensionHookSetup can be used to couple a hook function with the hooks it is meant for.
type ReconcileExtensionHookSetup struct {
	Hook      ReconcileExtensionHook
	HookTypes []HookType
}
```

This hook setup can given to the `RegisterHookSetup` helper method, which will register the hook for all given types.
```golang
// RegisterHookSetup is a wrapper for RegisterHook which uses a ReconcileExtensionHookSetup object instead of a hook function and types.
// It returns the ReconcileExtensionHooks object it is called on for chaining.
func (hooks ReconcileExtensionHooks) RegisterHookSetup(hookSetup ReconcileExtensionHookSetup) ReconcileExtensionHooks
```

Using hook setups, the above example of registering a hook turns into this:
```golang
myDeployer.hooks.RegisterHookSetup(myHookSetup)
```
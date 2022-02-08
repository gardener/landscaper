# Controlling the Landscaper via Annotations

There are a few annotations which can be set on landscaper objects to influence the reconciliation flow.

Please note that the effects which an annotation has on a deployitem depend on the implementation of the deployer responsible for that deployitem. Depending on the functionality of the deployer, developers might decide to deviate from the expected behavior (e.g. if deployitems of that type cannot be aborted). The 'effects on deployitems' described below therefore describe the default case, as it is implemented by the deployer library (as far as possible). 

## Operation Annotation

**Annotation:** `landscaper.gardener.cloud/operation`
**Accepted values:**
  - `reconcile`
  - `force-reconcile`
  - `abort`

#### Effect on Installations

**reconcile**
The installation will be queued for reconciliation. This is the standard way of triggering an installation without changing its spec. Note that the landscaper checks whether the installation is up-to-date, so setting this annotation will not necessarily result in redeploying the executions and subinstallations. 
The operation annotation is removed during the reconciliation.

**force-reconcile**
This enforces a redeployment of executions and subinstallations. The checks, whether any of them is still progressing or the installation's imports are outdated, are skipped.
In order to fix potentially broken executions of subinstallations, the force-reconcile annotation will be propagated to the subinstallations.
The operation annotation is removed during the force-reconciliation.

**abort**
If the abort operation annotation is set, a reconcile will be stopped before checking whether the installation needs to be updated.
The abort operation annotation is not removed automatically.


#### Effect on Executions

**reconcile**
TODO

**force-reconcile**
TODO

**abort**
TODO


#### Effect on DeployItems

**reconcile**
TODO

**force-reconcile**
TODO

**abort**
TODO


## Ignore Annotation

**Annotation:** `landscaper.gardener.cloud/ignore`
**Accepted values:**
  - `true`

#### Effect on Installations/Executions/DeployItems

The effect of this annotation is the same for all landscaper resources: the respective resource will not be reconciled by the landscaper, even if its spec changed or the operation annotation says otherwise. Only resources in a final phase are affected, to interrupt a running installation/execution/deployitem, the `landscaper.gardener.cloud/operation=abort` annotation has to be used.
Please note that as long as an update of a resource is blocked from reconciliation by this annotation, all other landscaper resources which are waiting for the update (because they depend on the resource) won't be able to be reconciled either and will be stuck in the `PendingDependencies` phase.
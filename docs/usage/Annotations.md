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
The installation will be queued for reconciliation just as if someone had changed its `spec`. The operation annotation will be removed during the reconciliation.

**force-reconcile**
TODO

**abort**
TODO


#### Effect on Executions

**reconcile**
Equivalent to the effect on installations.

**force-reconcile**
TODO

**abort**
TODO


#### Effect on DeployItems

**reconcile**
Equivalent to the effect on installations.

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
---
title: Controlling the Landscaper via Annotations
sidebar_position: 13
---

# Controlling the Landscaper via Annotations

There are a few annotations which can be set on landscaper objects to influence the reconciliation flow.

Please note that the effects which an annotation has on a deploy item might depend on the implementation of the deployer 
responsible for that deploy item. Depending on the functionality of the deployer, developers might decide to deviate from 
the expected behavior. The 'effects on deployitems' described below therefore describe the default case, as it is 
implemented by the deployer library (as far as possible). 

## Reconcile Annotation

**Annotation:** `landscaper.gardener.cloud/operation: reconcile`

With this annotation the processing of installations are started.

This value has only an effect on root installations, i.e. installations with no parent installation. If set, it initiates 
a new reconcile loop of the Landscaper for the installation. Without such an annotation the Landscaper does not process 
a root installation. This allows you to deploy all of your root installations and their input data first and then start 
the overall deployment by setting the reconcile annotation at the initial root installations (i.e. root installations 
not depending on other root installations) afterwards. The Landscaper starts processing these annotated root installations 
first. If a root installation was processed successfully the Landscaper triggers dependent root installations by setting 
this annotation automatically. A dependent annotated root installation is processed if all predecessors have finished their work.

This annotation also triggers another reconcile loop when the deletion of a root installation failed.

If this annotation is set at a sub installation the annotation is removed without any consequences.

This annotation has no effect at executions and deploy items.

## Interrupt Annotation

**Annotation:** `landscaper.gardener.cloud/operation: interrupt`

With this annotation currently running deployments could be interrupted. As it is processed in parallel to a currently 
running deployment, it might be that further deploy items might be created during propagating this annotation. 
Therefore, it could be required to set this annotation more than once to really stop a deployment.

If set at an installation, the Landscaper forwards it to all of its sub installations and execution. When forwarded 
the annotation is removed.

If set at an execution the Landscaper sets all existing deploy items which have not been finished processing so far
on failed and finished, i.e. it sets in their status as follows:
- `deployItemPhase` and `Phase` are set on `Failed`, indicating the deployment failed
- `jobIDFinished` and `jobId` are set on the job ID of the execution indicating that processing of the deploy item is 
  finished.

Afterwards the annotation is removed from the execution. 

Setting this annotation at a deploy item has no effect.

## Test Reconcile Annotation

**Annotation:** `landscaper.gardener.cloud/operation: test-reconcile`

With this annotation the processing of a deploy item could be started. This annotation must be used only in test
scenarios because it breaks the overall logic of the processing of installations and their sub-installations, executions
and deploy items.

This annotation has no effect at installations and executions.

## Delete-Without-Uninstall Annotation

**Annotation:** `landscaper.gardener.cloud/delete-without-uninstall: true`

If the annotation `landscaper.gardener.cloud/delete-without-uninstall: "true"` has been added to an installation, then
afterwards a deletion of the installation has the following effect:

- The installation, its sub installations, executions, and deploy items will be deleted,
- The installed artifacts on the target clusters will not be deleted. The deployers will only remove the finalizers at the 
  deploy items such that they could be deleted.

Note that you have to add the annotation **before** you delete the installation.

## Reconcile-If-Changed Annotation

See [here](https://github.com/gardener/landscaper/blob/master/docs/usage/Installations.md#automatic-reconciliationprocessing-of-installations-if-spec-was-changed).

## Cache-Helm-Charts Annotation

If the annotation `landscaper.gardener.cloud/cache-helm-charts: "true"` has been added to a root Installation,
all HelmCharts of deploy items of this Installation are fetched only once and cached locally. The default maximal 
size of the cache is 100 MB in the main memory. If more memory is required for new helm charts, the oldest entries are 
removed. Furthermore, by default all entries not used for more than one day, are also deleted.


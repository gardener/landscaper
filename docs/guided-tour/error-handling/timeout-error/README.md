---
title: Timeouts
sidebar_position: 2
---

# Timeouts

In this example, we will again deploy the Helm chart of the previous hello-world example. In order to demonstrate 
another error situation, we have slightly changed the [Installation](./installation/installation.yaml): 
It references chart version `0.0.5`, which does not exist.

For prerequisites, see [here](../../README.md).

## Procedure

1. In the [settings](commands/settings) file, adjust the variables `RESOURCE_CLUSTER_KUBECONFIG_PATH`
   and `TARGET_CLUSTER_KUBECONFIG_PATH`.

2. On the Landscaper resource cluster, create a namespace `cu-example`.

3. Run script [commands/deploy-k8s-resources.sh](commands/deploy-k8s-resources.sh).
   It creates a Target and an Installation on the resource cluster.


## Inspect the Result

When the Landscaper processes the Installation, it does not find the Helm chart version `0.0.5` which is referenced 
in the Installation. Landscaper considers this as a recoverable error situation. Therefore, the Installation remains in 
phase `Progressing`. Landscaper will retry the processing in intervals that become increasingly larger.

Status of the Installation:

```yaml
status:
   lastError:
      message: execution example / hello-world is not finished yet
   phase: Progressing
```

> Note: Whenever the state of an installation shows a `lastError`, and the phase is `Progressing`, the Landscaper will 
> try to reconcile the installation again after a certain, steadily increasing amount of time. This is done until a 
> timeout is reached. When this happens, the phase will change to `Failed` and Landscaper stops reconciliation.

Starting from the Installation, Landscaper creates further custom resources, namely DeployItems. In this concrete case, 
there will be only one DeployItem, which describes the Helm deployment of the hello-world chart. In the status section 
of the DeployItem, we can find further information about the error:

```shell
# Find the name of the DeployItem
▶ kubectl get di -n example
NAME                                    TYPE                             PHASE
hello-world-default-deploy-item-tslq8   landscaper.gardener.cloud/helm   Progressing

# Display the DeployItem
▶ kubectl get di -n example hello-world-default-deploy-item-tslq8 -o yaml
...
status:
  lastError:
    message: 'Op: TemplateChart - Reason: GetHelmChart - Message: unable to get manifest:
      eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:0.0.5: not
      found'
  phase: Progressing
```

The Landscaper CLI command `landscaper-cli inst inspect` prints an object tree consisting of the Installation, 
and DeployItems, together with status information:

```shell
▶ landscaper-cli inst inspect -n example hello-world
[🏗️ Progressing] Installation hello-world
    Last error: execution example / hello-world is not finished yet
    └── [🏗️ Progressing] DeployItem hello-world-default-deploy-item-tslq8
        Last error: Op: TemplateChart - Reason: GetHelmChart - Message: unable to get manifest: 
        eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:0.0.5: not found
```

After a few minutes, the DeployItem and the Installation will fail due to a timeout. In this example, we have set the 
length of the timeout interval to 2 minutes &mdash; see section 
[Configuring a Timeout for a DeployItem](#configuring-a-timeout-for-a-deployitem) below.

```shell
▶ landscaper-cli inst inspect -n example hello-world
[❌ Failed] Installation hello-world
    └── [❌ Failed] DeployItem hello-world-default-deploy-item-tslq8
        Last error: deployer has not finished this deploy item within 120 seconds
```

As a consequence of the failure of the DeployItem, the Installation also goes into a failure state.


## Resolving the Error

Let's resolve the error by fixing the Helm chart version in the Installation (you can find the corrected Installation 
here: [installation/installation-fixed.yaml](./installation/installation-fixed.yaml)), but we have to distinguish 
between two cases:

**Case 1:** The Installation has already failed due to the timeout described above. In this case, we can simply apply 
the Installation with the fixed Helm chart version. As usual, make sure that the Installation has the 
annotation `landscaper.gardener.cloud/operation: reconcile`, otherwise Landscaper will not start processing it. 
The [installation/installation-fixed.yaml](./installation/installation-fixed.yaml) already contains this annotation.

**Case 2:** The Installation has not yet failed, and is still in an unfinished phase like `Progressing`. 
As long as a deployment is still running, Landscaper does not take any changes of the corresponding Installation into 
account, since it is unpredictable what might happen. Therefore, in such unfinished phases, applying a changed 
Installation will not have any effect until the timeout has occurred and phase `Failed` has been reached 
(or the installation was `Succeeded`). However, if you do not want to wait until the timeout has occurred, 
you can **interrupt** the ongoing deployment as described below.

### Interrupting a Deployment

To interrupt an ongoing deployment, add the annotation `landscaper.gardener.cloud/operation: interrupt` to the 
Installation:

```shell
kubectl annotate inst -n cu-example hello-world landscaper.gardener.cloud/operation=interrupt
```

Alternatively, you can use the following command of the Landscaper CLI to add this annotation:

```shell
landscaper-cli inst interrupt -n cu-example hello-world
```

> **Warning:** Be aware that the interruption just _forces_ the Installation and its DeployItems into a final phase 
> (`Succeeded`, `Failed`, or `DeleteFailed`). The behaviour of for example a Helm installation, which might currently 
> run, is not defined. Therefore, you should interrupt a running deployment only if you are sure that nothing bad can 
> happen or in development scenarios. It is **not advised** to use this annotation in a productive environment.

## Deploy the fixed Installation

Once the Installation reaches phase `Failed`, run script 
[commands/deploy-fixed-installation.sh](commands/deploy-fixed-installation.sh)
to apply the corrected one ([installation/installation-fixed.yaml](./installation/installation-fixed.yaml)) 
with the fixed Helm chart version.

> Note that this fixed version already contains the annotation `landscaper.gardener.cloud/operation: reconcile`, 
> so that Landscaper will start processing it.

After some time, the phase of the Installation should be `Succeeded` and the ConfigMap deployed by the Helm chart should 
exist on the target cluster.


## Cleanup

You can remove the Installation with the
[delete-installation script](commands/delete-installation.sh).

Note: if the Installation is not yet in a final phase, the deletion process will not start directly. 
Rather it will wait until the current deployment process has finished. However, if you do not want
to wait for this, you can **interrupt** the ongoing deployment as described [above](#interrupting-a-deployment).

When the Installation is gone, you can delete the Target with the
[delete-other-k8s-resources script](commands/delete-other-k8s-resources.sh).


## Configuring a Timeout for a DeployItem

In this example we have specified a progressing timeout for a DeployItem. This is done in the DeployItem template of the 
Blueprint (here, inline in the [Installation](./installation/installation.yaml).)

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

deployExecutions:
  - name: default
    type: GoTemplate
    template: |
      deployItems:
        - name: default-deploy-item
          type: landscaper.gardener.cloud/helm
      
          timeout: 2m
```

If you do not specify a timeout, the default of 10 minutes is used.

For more details, see [DeployItem Timeouts](../../../usage/DeployItemTimeouts.md).

---
title: Troubleshooting Landscaper Issues
sidebar_position: 1
---

# Troubleshooting Landscaper Issues

Here you find a few practical tips on how to track down issues during landscaper deployments.


## Use the "landscaper-cli installations inspect" command

It is cumbersome to manually search through all the dependencies of a Landscaper Installation using kubectl describe. 
The [Landscaper CLI inspect command][1] prints an object tree consisting of the Installations, Executions, and DeployItems,
and it assists you in getting an overview of the current deployment status.

```shell
❯ landscaper-cli installations inspect -n <NAMESPACE> <INSTALLATION_NAME>

# Short form
❯ landscaper-cli inst inspect -n <NAMESPACE> <INSTALLATION_NAME>
```

If you skip the installation name, the inspect command displays all installations in the namespace.

#### Example: Installation that succeeded

In the simplest case, the object tree consists of an Installation, an Execution, and a DeployItem.
If all of them succeed, the inspect command shows this:

```shell
❯ landscaper-cli inst inspect -n cu-example echo-server
[✅ Succeeded] Installation echo-server
    └── [✅ Succeeded] Execution echo-server
        └── [✅ Succeeded] DeployItem echo-server-default-deploy-item-cglkb
```

#### Example: Installation that failed during create

Suppose the Installation fails before its Execution and DeployItem could be created. Then the object tree 
consists of the Installation only:

```shell
❯ landscaper-cli inst inspect -n cu-example echo-server
[❌ Failed] Installation echo-server
    Last error: unable to ...
```

The inspect command shows the error from the status of the Installation.

#### Example: Installation that failed during update

Suppose an Installation was succeeded, was then updated, and the update failed. In this case, the inspect command
shows this:

```shell
❯ landscaper-cli inst inspect -n cu-example echo-server
[❌ Failed] Installation echo-server
    Last error: unable to ...
    └── [✅ Succeeded (outdated)] Execution echo-server
        └── [✅ Succeeded (outdated)] DeployItem echo-server-default-deploy-item-cglkb
```

The Installation failed before it could update the Execution and DeployItem. Nevertheless, the Execution and DeployItem
exist and are still succeeded from the previous precessing. Therefore, their status are marked as "outdated".

Again, it makes sense to have a look at the Installation itself with command: 
`kubectl get installation -n <NAMESPACE> <INSTALLATION_NAME>`.

#### Example: Failed DeployItem

If a DeployItem fails, the inspect command returns this:

```shell
❯ landscaper-cli inst inspect -n cu-example echo-server
[❌ Failed] Installation echo-server
    └── [❌ Failed] Execution echo-server
        Last error: has failed or missing deploy items
        └── [❌ Failed] DeployItem echo-server-default-deploy-item-c9pdz
            Last error: unable to ...
```

If an object fails, the other objects which are not yet in a final phase are also set to `Failed`.

Inspect the status of the DeployItem with 
`kubectl get deployitem -n <NAMESPACE> <DEPLOYITEM_NAME>`.

#### Example: Progressing DeployItem

While the objects are being processed, the inspect command can return for example this:

```shell
❯ landscaper-cli inst inspect -n cu-example echo-server
[🏗️ Progressing] Installation echo-server
    Last error: execution cu-example / echo-server is not finished yet
    └── [🏗️ Progressing] Execution echo-server
        Last error: some running items
        └── [🏗️ Progressing] DeployItem echo-server-default-deploy-item-c9pdz
```

Note that there are more phases than `Progressing`, for example `Init` and `Completing`.

If this state persists longer than you expect, check the status of the DeployItem for errors. 
Certain errors are considered as retryable, so that the DeployItem does not switch to phase `Failed`. It then remains
`Progressing` until a timeout occurs.


## Error messages in the status

The inspect command helps to localize in which Installation or DeployItem an error might have error occurred. 
To further drill down, check the status of the involved Installation, Execution, or DeployItem
with commands like `kubectl get installation ...` or `kubectl get execution ...` or `kubectl get deployitem ...`.

Chapter [Common Issues](common-issues.md) explains the most important error messages.

### DeployItem Status

In case of an error, the status of a DeployItem does not only contain the `lastError`, but also a list of the
`lastErrors` (maximum 5), as well as the `firstError`. For example, it can happen that the last error is just a timeout, 
whereas the previous error gives a hint what really went wrong.

```yaml
status:
  firstError:
    ...
  lastError:
    ...
  lastErrors:
    ...
    - lastTransitionTime: "2024-07-24T14:10:40Z"
      lastUpdateTime: "2024-07-24T14:10:40Z"
      message: 'Op: TemplateChart - Reason: GetHelmChart - Message: downloading helm
      chart "eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:0.0.5":
      oci artifact "eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:0.0.5"
      not found in gardener-project/landscaper/examples/charts/hello-world'
      operation: Reconcile
      reason: Template
    - codes:
        - ERR_TIMEOUT
      lastTransitionTime: "2024-07-24T14:12:41Z"
      lastUpdateTime: "2024-07-24T14:12:41Z"
      message: 'timeout at: "helm deployer: start reconcile"'
      operation: StandardTimeoutChecker.TimeoutExceeded
      reason: ProgressingTimeout
```


## Trigger reconciliation of Installations

It might be necessary to trigger a reconciliation operation on an Installation resource. This can be achieved by 
applying the [landscaper.gardener.cloud/operation=reconcile][2] annotation on the root installation.

Example:

```shell
kubectl annotate installation -n <NAMESPACE> <INSTALLATION_NAME> landscaper.gardener.cloud/operation=reconcile

# Example
kubectl annotate installation -n cu-example echo-server landscaper.gardener.cloud/operation=reconcile
```

Note that the new reconciliation does not start before the Installation has reached a finished phase 
(`Succeeded`, `Failed`, or `DeleteFailed`).


<!-- References -->

[1]: https://github.com/gardener/landscapercli/blob/master/docs/reference/landscaper-cli_installations_inspect.md

[2]: https://github.com/gardener/landscaper/blob/master/docs/usage/Annotations.md#reconcile-annotation
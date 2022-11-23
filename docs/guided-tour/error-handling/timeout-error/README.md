# Handling a Timeout Error

For prerequisites see [here](../../README.md#prerequisites-and-basic-definitions).

In this example, we try again to deploy the Helm chart of the hello-world example.
To demonstrate another error situation, we have manipulated the [Installation](./installation/installation.yaml). 
It references chart version `0.0.9`, which does not exist.


## Procedure

1. Insert in file [target.yaml](installation/target.yaml) the kubeconfig of your target cluster.

2. On the Landscaper resource cluster, create namespace `example` and apply 
   the [target.yaml](installation/target.yaml) and the [installation.yaml](installation/installation.yaml):
   
   ```shell
   kubectl create ns example
   kubectl apply -f <path to target.yaml>
   kubectl apply -f <path to installation.yaml>
   ```

## Inspect the Result

When the Landscaper processes the Installation, it does not find the Helm chart version `0.0.9` that is referenced 
in the Installation. Landscaper considers this as a recoverable error situation. Therefore, the Installation remains in 
phase `Progressing`. Landscaper will retry the processing in intervals that become increasingly larger.

Status of the Installation:

```yaml
status:
   lastError:
      message: execution example / hello-world is not finished yet
   phase: Progressing
```

Starting from the Installation, Landscaper creates further custom resources, namely DeployItems. In the present case 
there will be only one DeployItem, that describes the Helm deployment of the hello-world chart. In the status of the 
DeployItem we find further information about the error:

```shell
# Find the name of the DeployItem
▶ kubectl get di -n example
NAME                                    TYPE                             PHASE
hello-world-default-deploy-item-tslq8   landscaper.gardener.cloud/helm   Progressing

# Display the DeployItem
▶ k get di -n example hello-world-default-deploy-item-tslq8 -o yaml
...
status:
  lastError:
    message: 'Op: TemplateChart - Reason: GetHelmChart - Message: unable to resolve
      chart from "eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:0.0.9":
      unable to get manifest: eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:0.0.9:
      not found'
  phase: Progressing
```

The Landscaper CLI command `landscaper-cli inst inspect` prints an object tree consisting of the Installation, 
and DeployItems, together with status information:

```shell
▶ landscaper-cli inst inspect -n example hello-world
[🏗️ Progressing] Installation hello-world
    Last error: execution example / hello-world is not finished yet
    └── [🏗️ Progressing] DeployItem hello-world-default-deploy-item-tslq8
        Last error: Op: TemplateChart - Reason: GetHelmChart - Message: unable to resolve chart from "eu.gcr.io/gardener-project
        /landscaper/examples/charts/hello-world:0.0.9": unable to get manifest: 
        eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:0.0.9: not found
```

After a few minutes, the DeployItem and the Installation will fail due to a timeout:

```shell
▶ landscaper-cli inst inspect -n example hello-world
[❌ Failed] Installation hello-world
    └── [❌ Failed] DeployItem hello-world-default-deploy-item-tslq8
        Last error: deployer has not aborted progressing this deploy item within 300 seconds
```

<details>
Actually there are two timeouts. After the first timeout, the "progressing timeout", the DeployItem is being told to 
abort the deployment. If it does not do that before the second timeout, the "abort timeout", it fails.
</details>

As a consequence of the failure of the DeployItem, the Installation also goes into a failure state.


## Resolving the Error

Let's resolve the error situation by fixing the Helm chart version in the Installation. There are two cases.

**Case 1:** The Installation has already failed due to the timeout described above. In this case we can simply apply
the Installation with the fixed Helm chart version. Make sure that the Installation has the annotation
`landscaper.gardener.cloud/operation: reconcile`, otherwise Landscaper will not start processing it.

**Case 2:** The Installation has not yet failed, but is still in an unfinished phase like `Progressing`.
The point is that Landscaper does not take any change of the Installation spec into account as long as a deployment is 
still running. It is incalculable what might happen when one would change an ongoing deployment in the middle of 
the processing. Therefore, it is possible to change the Installation spec, but this change will have no effect 
before the timeout has occurred. However, if you do not want to wait until the timeout has occurred, 
you can **interrupt** the ongoing deployment as described below.


## Interrupting a Deployment

To interrupt the ongoing deployment, add the annotation `landscaper.gardener.cloud/operation: interrupt` to the
Installation:

```shell
kubectl annotate inst -n example hello-world landscaper.gardener.cloud/operation=interrupt
```

Alternatively, you can use the following command of the Landscaper CLI to add this annotation:

```shell
landscaper-cli inst interrupt -n example hello-world
```

**Warning:** Be aware that the interruption just forces the Installation and its DeployItems into a 
final phase (`Succeeded`, `Failed`, or `DeleteFailed`). The behaviour concerning for example a Helm installation that 
might run at that moment, is not defined. Therefore, you should interrupt a running deployment only if you are sure 
that nothing bad can happen. It is not recommended to use this annotation in a productive environment.

## Deploy the fixed Installation

Once the Installation is in phase `Failed`, apply the Installation
[installation/installation-fixed.yaml](./installation/installation-fixed.yaml) with the fixed Helm chart version:

```shell
kubectl apply -f <path to installation-fixed.yaml>
```

Note that this fixed version already contains the annotation `landscaper.gardener.cloud/operation: reconcile`, so
that Landscaper will start processing it. After some time, the phase of the Installation should be `Succeeded` and
the ConfigMap deployed by the Helm chart should exist on the target cluster.

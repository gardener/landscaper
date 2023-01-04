# Handling a Delete Error

For prerequisites, see [here](../../README.md#prerequisites-and-basic-definitions).

In this example, we will again deploy the Helm chart of the hello-world example and then delete it again. We will provoke an error during the deletion by removing the target before the deletion happened.

## Procedure

First create the Target and the Installation again:

1. Insert the kubeconfig of your target cluster into your [target.yaml](installation/target.yaml). 

2. On the Landscaper resource cluster, create namespace `example` and apply 
   the [target.yaml](installation/target.yaml) and the [installation.yaml](installation/installation.yaml):
   
   ```shell
   kubectl create ns example
   kubectl apply -f <path to target.yaml>
   kubectl apply -f <path to installation.yaml>
   ```

## Delete the Target and the Installation

In the next step, we will delete the target `my-cluster`:

```shell
kubectl delete target -n example my-cluster
```

Now, when we try to delete the Installation, there is no access information of the target cluster available from where the Helm chart should be removed. This will lead to a failing deletion of the Installation:

```shell
kubectl delete installation -n example hello-world
```

It will take some time until the timeout occurs and the Installation fails, i.e. `phase: DeletionFailed` is being reached. You can speed this up by setting the interrupt annotation (as described in more detail in the [previous example](..//timeout-error/readme.md#interrupting-a-deployment)): 

```shell
kubectl annotate installation -n example hello-world landscaper.gardener.cloud/operation=interrupt
```

## Resolve the failed Deletion

### Recreate the Target

For prerequisites, [see](../../README.md#prerequisites-and-basic-definitions).

The usual way to resolve this failed Deletion is to recreate the target and re-trigger the deletion of the installation by setting the `reconcile` annotation:

```shell
kubectl apply -f <path to target.yaml>
kubectl annotate installation -n example hello-world landscaper.gardener.cloud/operation=reconcile
```

Now the installation should be gone and the deployed Helm chart should be uninstalled.

### Force Delete

If for some reason it is not possible to resolve the problems as described above and it is ok that the deployed Helm chart is not uninstalled automatically, you could use the [landscaper-cli](https://github.com/gardener/landscapercli) to remove the installation. This is the preferred solution if the target cluster does not exist anymore. You can achieve this with the following command:

```shell
landscaper-cli installations force-delete -n example hello-world
```

> Note: It is **not recommended** to use this approach if you have a successful Installation and want to remove just the Installation, without
uninstalling the deployed components on the target cluster. In such a situation, the annotation
`landscaper.gardener.cloud/delete-without-unstall: true` should be used.


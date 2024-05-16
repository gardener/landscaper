---
title: Handling Deletion Errors
sidebar_position: 3
---

# Handling a Delete Error

In this example, we will again deploy the Helm chart of the hello-world example and then delete it. During the deletion, an error will be provoked by removing the target before the deletion happened.

For prerequisites, see [here](../../README.md).

## Procedure

First create the Target and the Installation again:

1. In the [settings](commands/settings) file, adjust the variables `RESOURCE_CLUSTER_KUBECONFIG_PATH`
   and `TARGET_CLUSTER_KUBECONFIG_PATH`.

2. On the Landscaper resource cluster, create a namespace `cu-example`.

3. Run script [commands/deploy-k8s-resources.sh](commands/deploy-k8s-resources.sh).
   It creates a Target and an Installation on the resource cluster.


## Delete the Target and the Installation

In the next step, delete the target `my-cluster`:

```shell
kubectl delete target -n cu-example my-cluster
```

Now, when we try to delete the Installation, there is no access information for the target cluster available from where the Helm chart should be removed. This will lead to a failing deletion of the Installation:

```shell
kubectl delete installation -n cu-example hello-world
```

It will take some time until the timeout occurs and the Installation fails, i.e. `phase: DeletionFailed` is being reached. You can speed this up by setting the interrupt annotation (as described in more detail in the [previous example](..//timeout-error/README.md#interrupting-a-deployment)): 

```shell
kubectl annotate installation -n cu-example hello-world landscaper.gardener.cloud/operation=interrupt
```

## Resolve the failed Deletion

### Recreate the Target

For prerequisites, [see](../../README.md#prerequisites-and-basic-definitions).

The usual way to resolve this failed Deletion is to recreate the target. 
Script [commands/deploy-target.sh](commands/deploy-target.sh) does this. 
Afterwards, re-trigger the deletion of the installation by setting the `reconcile` annotation:

```shell
kubectl annotate installation -n cu-example hello-world landscaper.gardener.cloud/operation=reconcile
```

Now the installation should be gone and the deployed Helm chart should be uninstalled.

### Force Delete

If for some reason it is not possible to resolve the problems as described above and it is ok that the deployed Helm chart is not uninstalled automatically, 
you could use the [landscaper-cli](https://github.com/gardener/landscapercli) to remove the installation. 
This is the preferred solution if the target cluster does not exist anymore. You can achieve this with the following command:

```shell
landscaper-cli installations force-delete -n cu-example hello-world
```

> Note: It is **not recommended** to use this approach if you have a successful Installation and want to remove just the Installation, without
uninstalling the deployed components on the target cluster. In such a situation, the annotation
`landscaper.gardener.cloud/delete-without-uninstall: true` should be used.

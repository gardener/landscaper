# Handling a Delete Error

For prerequisites see [here](../../README.md#prerequisites-and-basic-definitions).

In this example, we deploy again the Helm chart of the hello-world example and then delete it again whereby we provoke
an error by removing the target before. 

## Procedure

First create the Target and the Installation again:

1. Insert in file [target.yaml](installation/target.yaml) the kubeconfig of your target cluster.

2. On the Landscaper resource cluster, create namespace `example` and apply 
   the [target.yaml](installation/target.yaml) and the [installation.yaml](installation/installation.yaml):
   
   ```shell
   kubectl create ns example
   kubectl apply -f <path to target.yaml>
   kubectl apply -f <path to installation.yaml>
   ```

## Delete the Target and the Installation

In the next step we delete the target `my-cluster`:

```shell
kubectl delete target -n example my-cluster
```

Now when we delete the Installation, there is no access information of the target cluster available from where
the Helm chart should be removed and therefore the deletion of the Installation fails:

```shell
kubectl delete installation -n example hello-world
```

Now it requires some time until the timeout occurs and the Installation fails, i.e. `phase: DeletionFailed`. You could 
speed this up by setting the interrupt annotation: 

```shell
kubectl annotate installation -n example hello-world landscaper.gardener.cloud/operation=interrupt
```

## Resolve the failed Installation

### Recreate the Target

For prerequisites [see](../../README.md#prerequisites-and-basic-definitions).

The usual way to resolve the failed Installation is to recreate the target and re-trigger the deletion of the
installation by setting the `reconcile` annotation:

```shell
kubectl apply -f <path to target.yaml>
kubectl annotate installation -n example hello-world landscaper.gardener.cloud/operation=reconcile
```

Now the installation is gone and the deployed Helm chart was uninstalled.

### Force Delete

If it is not possible to resolve the problems and it is ok that the deployed Helm chart is not uninstalled,
you could use the [landscaper-cli](https://github.com/gardener/landscapercli) to remove the installation. For example,
this is the preferred solution if the target cluster does not exist anymore. You achieve this with the following 
command:

```shell
landscaper-cli installations force-delete -n example hello-world
```

It is not recommended to use this approach, if you have a successful Installation and want to remove it without
uninstalling the deployed components on the target cluster. In such a situation use the annotation
`landscaper.gardener.cloud/delete-without-unstall: true`.


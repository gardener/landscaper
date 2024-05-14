---
title: Hello World Example
sidebar_position: 2
---

# Hello World Example

In this example, we use the Landscaper to deploy a simple Helm chart.

For prerequisites, see [here](../README.md).

Our [hello-world Helm chart](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/hello-world/chart/hello-world) 
is minimalistic on purpose, in order to concentrate on Landscaper rather than Helm features. Therefore, the chart only 
deploys a ConfigMap. We have uploaded the chart to a [public registry](https://eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:1.0.0) 
from where the Landscaper reads it during the deployment.

## Procedure

First of all, we need to create two custom resources:
- a `Target` custom resource, containing the access information for the target cluster
- and an `Installation` custom resource containing the instructions for deploying the Helm chart.

1. In the [settings](https://github.com/gardener/landscaper/blob/master/docs/guided-tour/hello-world/commands/settings) file, adjust these variables:
    - variable `RESOURCE_CLUSTER_KUBECONFIG_PATH` so that it contains the path to the kubeconfig of your resource cluster,
      i.e. the cluster where the Landscaper resources like Installations, Targets etc. are created,
    - variable `TARGET_CLUSTER_KUBECONFIG_PATH` so that it contains the path to the kubeconfig of your target cluster,
      i.e. the cluster on which the deployment will be done.

   > **Note:**  
   > If your target cluster is a Gardener Shoot cluster, it is **not** possible to use an oidc / gardenlogin kubeconfig in a Target.  
   > If you have only such a kubeconfig, see 
   > ["Constructing a Target Resource"](https://github.com/gardener/landscaper/blob/master/docs/guided-tour//targets/README.md)
   > how to construct a kubeconfig and a Target, that you can use with the Landscaper.

2. On the Landscaper resource cluster, create a namespace `cu-example`.

   Please note: In case you are using a Landscaper instance managed by Landscaper-as-a-Service, you cannot
   create a namespace directly. Instead, you have to follow the procedure described 
   [here](https://github.com/gardener/landscaper-service/blob/main/docs/usage/Namespaceregistration.md).

3. Run script [commands/deploy-k8s-resources.sh](commands/deploy-k8s-resources.sh). 
It templates a [target.yaml.tpl](installation/target.yaml.tpl) and an [installation.yaml.tpl](installation/installation.yaml.tpl)
and applies both on the resource cluster.

### Alternative Way to Create a Target

There is an alternative way to create a Target custom resource. It requires the [Landscaper CLI](https://github.com/gardener/landscapercli):

```shell
landscaper-cli targets create kubernetes-cluster \
  --name my-cluster \
  --namespace cu-example \
  --target-kubeconfig "${TARGET_CLUSTER_KUBECONFIG_PATH}" \
  | kubectl apply -f - --kubeconfig="${RESOURCE_CLUSTER_KUBECONFIG_PATH}"
```

The command uses two kubeconfig paths:
- `RESOURCE_CLUSTER_KUBECONFIG_PATH` must contain the path to the kubeconfig of your Landscaper resource cluster,
  i.e. the cluster where the Target and Installation should be created.
- `TARGET_CLUSTER_KUBECONFIG_PATH` must contain the path to the kubeconfig of your target cluster, i.e. the cluster
  on which you want to deploy something, in our case the hello-world Helm chart.

## Landscaper Processes the Installation

After applying the `Target` and `Installation` resources to the Landscaper resource cluster, the Landscaper starts with 
the installation of the Helm chart. Please note that the Landscaper only starts working on an installation, if the 
annotation `landscaper.gardener.cloud/operation: reconcile` is present. This annotation is automatically removed by the 
Landscaper as soon as it starts with processing the installation.

If you require the Landscaper to process the installation again (in case you did some changes to the `installation` 
resource and thus require a reconciliation), just add the `landscaper.gardener.cloud/operation: reconcile` annotation again.

## Inspect the Result

You can now check the status of the installation:

```shell
kubectl get inst -n cu-example hello-world
```

The most important field in the status section is the `phase`, which should have show the value `Succeeded` as soon as 
the Helm chart has been successfully deployed.

```yaml
status:
  phase: Succeeded
```

If you have already installed the [Landscaper CLI](https://github.com/gardener/landscapercli), 
you can inspect the status of the installation with the following command, executed on the Landscaper resource cluster:

```shell
landscaper-cli inst inspect -n cu-example hello-world
```

Another important entry in status section of an installation is the `observedGeneration`. It describes to which version 
of the installation, defined by its `generation`, the current status refers to. In order to check if the latest version 
of an installation has been processed, you must check
- whether `phase` is equal to `Succeeded` or `Failed` (or `DeleteFailed`, if the deletion of the installation failed) and
-  whether `generation` is equal to `observedGeneration`.

After the successful installation, you should find the ConfigMap, which was deployed as part of the Helm chart, on the 
target cluster:

```shell
kubectl get configmap -n example hello-world
```

## Have a look at the Installation

In this example, we created an `Installation` custom resource, containing the instructions for deploying our example 
Helm chart, and a `Target` custom resource, containing the access information for the target cluster on which the 
Helm chart should be deployed. 

The `Installation` contains two main sections in its `spec`:

```yaml
spec:

  # Set values for the import parameters of the blueprint
  imports:
    targets:
      - name: cluster        # name of an import parameter of the blueprint
        target: my-cluster   # name of the Target custom resource containing the kubeconfig of the target cluster

  blueprint:
    ...
```

The `imports` section contains the reference to the target object and the `blueprint` section the deploy instructions 
(we will cover the topic of `blueprints` in a later example).


## Delete Installation

You can uninstall the hello-world Helm chart by deleting the `Installation` custom resource from the Landscaper resource cluster:

```shell
kubectl delete inst -n cu-example hello-world
```

Note that deleting an `Installation` like this will also delete the deployed Helm chart, which is the expected behaviour. 

You can keep the `Target`, because we will use it again in the next example. But if you want to delete it, run

```shell
kubectl delete target -n cu-example my-cluster
```

## Automatic Reconcile

Above we wrote that Landscaper only starts working on an Installation if it has the annotation
`landscaper.gardener.cloud/operation: reconcile`. 

There is also the possibility to let Landscaper add this annotation automatically such that you get an automatic 
reconciliation of an Installation. For more details see 
[here](../../usage/Installations.md#automatic-reconciliationprocessing-of-installations).

With the annotation `landscaper.gardener.cloud/reconcile-if-changed: "true"`, Installations are automatically processed
only if their `spec` was changed and a new `generation` created. For more details see
[here](../../usage/Installations.md#automatic-reconciliationprocessing-of-installations-if-spec-was-changed).

## References

[Installations](../../usage/Installations.md)

[Reconcile Annotation](../../usage/Annotations.md#reconcile-annotation)

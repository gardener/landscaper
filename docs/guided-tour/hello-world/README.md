# Hello World Example

In this example, we use the Landscaper to deploy a Helm chart.

Our [hello-world Helm chart](chart/hello-world) is minimalistic to concentrate on Landscaper rather than Helm features. 
It deploys just a ConfigMap. We have uploaded the chart to a 
[public registry](https://eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:1.0.0) from where the Landscaper 
reads it during the deployment.

There are three clusters in total in this example:

- the **Landscaper Host Cluster**, on which the Landscaper runs;
- the **target cluster**, on which the Helm chart shall be deployed.
- the **Landscaper Resource Cluster**, on which the custom resources are stored that are watched by the Landscaper. 
  These custom resources define what should be deployed on which target cluster.

It is possible that some or all of these clusters coincide.

## Procedure

In this example we create a Target custom resource, containing the access information for the target cluster and an
Installation custom resource containing the instructions to deploy our example Helm chart. 

1. Insert in file [target.yaml](installation/target.yaml) the kubeconfig of your target cluster.

2. On the Landscaper resource cluster, create namespace `example` and apply 
   the [target.yaml](installation/target.yaml) and the [installation.yaml](installation/installation.yaml):
   
   ```shell
   kubectl create ns example
   kubectl apply -f <path to target.yaml>
   kubectl apply -f <path to installation.yaml>
   ```

Alternative (requires the [Landscaper CLI](https://github.com/gardener/landscapercli)):

1. In the [commands/settings file](./commands/settings), specify 
   - the path to the kubeconfig of your Landscaper resource cluster and
   - the path to the kubeconfig of your target cluster.

2. Run script [commands/apply-target-and-installation.sh](./commands/apply-target-and-installation.sh).

## Landscaper Processes the Installation

After the deployment of the Target and the Installations the Landscaper picks up these resources and start with
the installation of the Helm chart. Note that the Landscaper only starts working on an Installation if it has the 
annotation `landscaper.gardener.cloud/operation: reconcile`. Landscaper removes the annotation when it starts processing
the Installation. Later, if you want to process the Installation again, just add the annotation another time.


## Inspect the Result

You can now check the status of the Installation:

```shell
kubectl get inst -n example hello-world
```

The most important field in the status section is the `phase` which should have finally the value `Succeeded` if the
Helm chart was successfully deployed.

```yaml
status:
  phase: Succeeded
```

If you have already installed the [Landscaper CLI](https://github.com/gardener/landscapercli), 
you can inspect the status of the installation with the following command, executed on the Landscaper resource cluster:

```shell
landscaper-cli inst inspect -n example hello-world
```

On the target cluster, you should find the ConfigMap, that was deployed as part of the Helm chart:

```shell
kubectl get configmap -n example hello-world
```

## Have a look at the Installation

In this example we created an Installation custom resource containing the instructions to deploy our example Helm chart
a Target custom resource, containing the access information for the target cluster on which the Helm chart should be 
deployed. 

The Installation contains two main sections in its `spec`:

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

The `imports` section contains the reference to the target object and the `blueprint` section the deploy instructions.


## Delete Installation

You can uninstall the hello-world Helm chart by deleting the Installation from the Landscaper resource cluster:

```shell
kubectl delete inst -n example hello-world
```


## References

[Installations](../../usage/Installations.md)

[Reconcile Annotation](../../usage/Annotations.md#reconcile-annotation)

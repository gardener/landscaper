# Upgrading the Hello World Example

In this example, we start by deploying the hello-world Helm chart in its original version `1.0.0`. 
Afterwards, we will upgrade the Installation so that it deploys the newer version `1.0.1` of the chart.

You can find the new Helm chart [here](chart/hello-world). It replaces the ConfigMap of the original chart version by
a Secret. We have uploaded the new chart version to the
[public registry](eu.gcr.io/gardener-project/landscaper/examples/charts/hello-world:1.0.1).


## Procedure

First, we deploy the original hello-world scenario:

1. Insert in file [target.yaml](installation/target.yaml) the kubeconfig of your target cluster.

2. On the Landscaper resource cluster, create namespace `example` and apply 
   the [target.yaml](installation/target.yaml) and the original Installation 
   [installation-1.0.0.yaml](installation/installation-1.0.0.yaml):
   
   ```shell
   kubectl create ns example
   kubectl apply -f <path to target.yaml>
   kubectl apply -f <path to installation-1.0.0.yaml>
   ```

3. Wait until the Installation is in phase `Succeeded` and check that the ConfigMap of the original Helm chart
   was created.

4. Upgrade the Installation by applying [installation-1.0.1.yaml](installation/installation-1.0.1.yaml)
   The upgraded Installation references the newer version `1.0.1` of the hello-world Helm chart, which deploys a Secret
   instead of a ConfigMap.

   ```shell
   kubectl apply -f <path to installation-1.0.1.yaml>
   ```

   Note that the upgraded Installation has the annotation `landscaper.gardener.cloud/operation: reconcile`. Without this
   annotation, Landscaper will not start processing the Installation.

5. Wait until the Installation is again in phase `Succeeded`. The ConfigMap that was deployed by the old chart version 
   should no longer exist. Instead, there should now be a Secret deployed by the new chart version:

   ```shell
   kubectl get secret -n example hello-world
   ```

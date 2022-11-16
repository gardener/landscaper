# Handling an Immediate Error

In this example, we deploy again the Helm chart of the hello-world example.
To illustrate the error handling, we introduced an error in the Installation: a `:` is missing in the imports section
of the blueprint in the [Installation](./installation/installation.yaml).


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

## Inspect the Result

This time, the Installation will fail due to the invalid blueprint.

```yaml
status:
  lastError:
    message: 'unable to decode blueprint definition from inline defined blueprint.yaml: line 6: could not find expected '':'''
    ...
  phase: Failed
```

## Deploy the fixed Installation

You can find a fixed version of the Installation in 
[installation/installation-fixed.yaml](./installation/installation-fixed.yaml).

Deploy this version:

```shell
kubectl apply -f <path to installation-fixed.yaml>
```

Note that this fixed version already contains the annotation `landscaper.gardener.cloud/operation: reconcile`, so
that Landscaper will start processing it. After some time, the phase of the Installation should be `Succeeded` and
the ConfigMap deployed by the Helm chart should exist on the target cluster.

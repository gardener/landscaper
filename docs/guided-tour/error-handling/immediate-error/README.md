---
title: Immediate Errors
sidebar_position: 1
---

# Handling an Immediate Error

In this example, we deploy again the Helm chart of the hello-world example.
To illustrate the error handling, we have introduced an error: a `:` is missing in the imports section
of the blueprint in the [Installation](installation/installation.yaml.tpl).

For prerequisites, see [here](../../README.md).

## Procedure

We will again create a Target custom resource, containing the access information for the target cluster, 
and an Installation custom resource, containing the instructions to deploy our example Helm chart. 

1. In the [settings](commands/settings) file, adjust the variables `RESOURCE_CLUSTER_KUBECONFIG_PATH`
   and `TARGET_CLUSTER_KUBECONFIG_PATH`.

2. On the Landscaper resource cluster, create a namespace `cu-example`.

3. Run script [commands/deploy-k8s-resources.sh](commands/deploy-k8s-resources.sh).
   It creates a Target and an Installation on the resource cluster.

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

Here you can find a fixed version of the Installation: [installation/installation-fixed.yaml](installation/installation-fixed.yaml).

Deploy it with the script [commands/deploy-fixed-installation.sh](commands/deploy-fixed-installation.sh).

> Note that this fixed version already contains the annotation `landscaper.gardener.cloud/operation: reconcile`, so that Landscaper will start processing it. This is considered a good practice, as it eliminates the additional step of adding the reconcile annotation afterwards.

After some time, the phase of the Installation should change to `Succeeded`, and the ConfigMap deployed by the Helm chart should exist on the target cluster.

## Cleanup

You can remove the Installation with the
[delete-installation script](commands/delete-installation.sh).
When the Installation is gone, you can delete the Target with the
[delete-other-k8s-resources script](commands/delete-other-k8s-resources.sh).

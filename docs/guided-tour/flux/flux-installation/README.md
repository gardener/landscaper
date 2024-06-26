---
title: Flux Installation
sidebar_position: 1
---

# Flux Installation

This example deploys flux.


## Procedure

The procedure to install the flux controllers is as follows:

1. Adapt the [settings](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/flux/flux-installation/commands/settings) file
   such that the entry `TARGET_CLUSTER_KUBECONFIG_PATH` points to the kubeconfig of the target cluster.

2. On the Landscaper resource cluster, create a namespace `cu-example`.

3. Run the script [deploy-k8s-resources script](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/flux/flux-installation/commands/deploy-k8s-resources.sh).
   It will create  Landscaper custom resources on the resource cluster in namespace `cu-example`, namely a Target, a Context, and an Installation.

Check the status of the Installation:

```shell
❯ landscaper-cli inst inspect -n cu-example

[✅ Succeeded] Installation flux-installation
    └── [✅ Succeeded] Execution flux-installation
        └── [✅ Succeeded] DeployItem flux-installation-item-bw4tv
```

As a result, on the target cluster, the flux controllers should run in namespace `flux-system`:

```shell
❯ k get deploy -n flux-system

NAME                          READY   UP-TO-DATE   AVAILABLE   AGE
helm-controller               1/1     1            1           12m
image-automation-controller   1/1     1            1           12m
image-reflector-controller    1/1     1            1           12m
kustomize-controller          1/1     1            1           12m
notification-controller       1/1     1            1           12m
source-controller             1/1     1            1           12m
```


## Cleanup

You can remove the Installation with the script
[commands/delete-installation.sh](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/flux/flux-installation/commands/delete-installation.sh).

When the Installation is gone, you can delete the Context and Target with the script
[commands/delete-other-k8s-resources.sh](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/flux/flux-installation/commands/delete-other-k8s-resources.sh).

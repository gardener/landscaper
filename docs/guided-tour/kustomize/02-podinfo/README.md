---
title: Flux Installation
sidebar_position: 2
---

# PodInfo

This example deploys [stefanprodan's podinfo application](https://github.com/stefanprodan/podinfo) on the target cluster.

For prerequisites, see [here](../../README.md). In addition, we assume that the flux controllers are running on the target cluster. 
Example [Flux Installation](../flux-installation/README.md) show how to install them with the Landscaper. 

To deploy the podinfo application, we proceed in two steps: 
- We use the manifest deployer of the Landscaper to create two Flux custom resources on the target cluster: 
  a `GitRepository` and a `Kustomization` resource.
- Next, Flux controllers will reconcile these resources and perform the actual deployment of the podinfo application.

The `GitRepository` resource points to the master branch of the git repository https://github.com/stefanprodan/podinfo:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: podinfo
  namespace: flux-system
spec:
  interval: 60m
  ref:
    branch: master
  timeout: 60s
  url: https://github.com/stefanprodan/podinfo
```

Secondly, we create a `Kustomization` custom resource of Flux. It points to the directory `./kustomize` in the above 
Git repository. The deployment is done using kustomize according to the `kustomization.yaml` in this directory.

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  annotations:
    reconcile.fluxcd.io/requestedAt: "2024-07-01T13:41:49.987Z"
  name: podinfo
  namespace: flux-system
spec:
  force: false
  interval: 30m
  path: ./kustomize
  prune: true
  retryInterval: 2m0s
  sourceRef:
    kind: GitRepository
    name: podinfo
  targetNamespace: cu-podinfo
  timeout: 3m0s
  wait: true
```

The creation of the `GitRepository` and `Kustomization` resources starts a process, which reconciles the resources of 
the podinfo application every 30 minutes (specified in field `spec.interval`), and also when the sources are changed. 
As a consequence, the status of the `GitRepository` and `Kustomization` resources might change over time. However, this 
process is controlled by Flux, not by the Landscaper.

The task of the Landscaper in this scenario is to create the `GitRepository` and `Kustomization` resources. 
Once this is done, the Installation will be in a final phase (`Succeeded` or `Failed`). It will remain in this phase
until a new reconcile of the Installation is triggered.
It is not the task of the Landscaper to watch the `GitRepository` and `Kustomization` resources and to react to status 
changes of them. 


> We should not even wait until the Kustomization is ready for the first time. 
> => Remove readiness check in this example.


## Procedure

1. Adapt the [settings](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/flux/podinfo/commands/settings) file
   such that the entry `TARGET_CLUSTER_KUBECONFIG_PATH` points to the kubeconfig of the target cluster.

2. On the Landscaper resource cluster, create a namespace `cu-example`.

3. Run the script [deploy-k8s-resources script](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/flux/podinfo/commands/deploy-k8s-resources.sh).
   It will create  Landscaper custom resources on the resource cluster in namespace `cu-example`, namely a Target, a Context, and an Installation.

Check the status of the Installation:

```shell
❯ landscaper-cli inst inspect -n cu-example flux-podinfo

[✅ Succeeded] Installation flux-podinfo
    └── [✅ Succeeded] Execution flux-podinfo
        └── [✅ Succeeded] DeployItem flux-podinfo-item-nf2z7
```

As a result, on the target cluster, the podinfo application should run in namespace `cu-podinfo`:

```shell
❯ k get deployments -n cu-podinfo

NAME                      READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/podinfo   2/2     2            2           6m52s
```


## Cleanup

You can remove the Installation with the script
[commands/delete-installation.sh](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/flux/podinfo/commands/delete-installation.sh).

When the Installation is gone, you can delete the Context and Target with the script
[commands/delete-other-k8s-resources.sh](https://github.com/gardener/landscaper/tree/master/docs/guided-tour/flux/podinfo/commands/delete-other-k8s-resources.sh).

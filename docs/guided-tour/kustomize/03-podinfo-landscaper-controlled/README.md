---
title: Deploying the PodInfo Application with Landscaper and Flux - Controlled by Landscaper
sidebar_position: 3
---

# Deploying the PodInfo Application with Landscaper and Flux - Controlled by Landscaper


The scenario is essentially the same as described in the [previous example](../02-podinfo/README.md#scenario):
Landscaper deploys Flux custom resources on a first target cluster. Based of them, Flux deploys the 
[PodInfo application][1] on a second target cluster.

In the previous example, Flux was controlling when the PodInfo application was deployed, namely in intervals and upon 
changes in the source repository. Therefore, without interaction of the Landscaper, the status of the Flux resources 
could change over time.

In this example, the Landscaper controls the deployment. This means, a reconciliation of the Installation triggers a
reconciliation of the Flux resources. At other times, the Flux Kustomization is suspended to prevent further reconciles. 


## Prerequisites

For the usual prerequisites, see [here](../../README.md).
In addition, we use a second target cluster (which may coincide with the first). 
The first target cluster is for the Flux resources, and the second for the podinfo application. 

Moreover, we assume that Flux is installed on the first target cluster.
We need the [source controller][4] and the [kustomize controller][5] of Flux.
Example [Install Flux](../01-kustomize-introduction/README.md#install-flux) shows how to install them with the Landscaper.


## Controlling the Deployment

We describe how to modify the previous example to achieve that Landscaper controls the deployment.
When the Installation is reconciled, the following happens:

- Landscaper creates or updates the Flux custom resources and adds an annotation which 
  [triggers a reconciliation by Flux](#triggering-a-flux-reconciliation). 
- The Installation has a custom readiness check, so that it waits until the Kustomization resource is ready.
- After the readiness check, the Flux Kustomization is suspended to prevent further reconciles by Flux.
  The suspension will be revoked when Landscaper triggers the next reconciliation.


### Triggering a Flux Reconciliation

In the deploy items which create the GitRepository and Kustomization resources, we ensure that the resources get the
following ["reconcile.fluxcd.io/requestedAt" annotation](https://fluxcd.io/flux/components/kustomize/kustomizations/#triggering-a-reconcile):

```yaml
{{- $requestedAt := now | date "2006-01-02T15:04:05.999Z" }}
...
annotations:
  reconcile.fluxcd.io/requestedAt: {{ $requestedAt }}
```

Whenever Landscaper creates or updates the Flux resources, it sets a new value for the annotation (the current timestamp) and
thereby triggers a reconcile by Flux.


### Suspending the Kustomization

In the deploy items which create the Kustomization resources, we add sections 
`patchAfterDeployment` and `patchBeforeDelete`:

```yaml
manifests:
  - policy: manage
    patchAfterDeployment:
      spec:
        suspend: true
    patchBeforeDelete:
      spec:
        suspend: false
    manifest:
      apiVersion: kustomize.toolkit.fluxcd.io/v1
      kind: Kustomization
      ...
```

The `patchAfterDeployment` section has an effect after the readiness check and after the collection of
export parameters (if there are any). Then, Landscaper modifies the resource by merging the `patchAfterDeployment` 
section into it.

In our case, it sets the field `spec.suspend` of the Kustomization resource to `true`. In this way, further
deployments by Flux are suspended.

The Kustomization remains suspended until Landscaper reconciles the next time and sets the value of the `suspend` field 
to the value `false` again:

```yaml
manifests:
  - manifest:
      apiVersion: kustomize.toolkit.fluxcd.io/v1
      kind: Kustomization
      spec:
        suspend: false
```

Moreover, before a deletion, the `patchBeforeDelete` section is merged into the resource.
In our case, it sets the field `spec.suspend` of the Kustomization resource to `false`. Consequently, the Kustomization
is no longer suspended and Flux can perform the deletion.

Note that we also have set large values for the field `interval` in the Flux resources.
It ensures that Flux deploys only once when triggered by the Landscaper. However, most of the time, Flux activities are
anyhow suspended as just described.


## Procedure

1. Adapt the [settings](commands/settings) file
   such that
    - entry `TARGET_CLUSTER_KUBECONFIG_PATH` points to the kubeconfig of the first target cluster,
    - entry `TARGET_CLUSTER_KUBECONFIG_PATH_2` points to the kubeconfig of the second target cluster.

2. Create the following namespaces:
    - namespace `cu-example` on the resource cluster,
    - namespace `flux-system` on the first target cluster,
    - namespace `cu-podinfo` on the second target cluster.

3. Run the script [deploy-k8s-resources script](commands/deploy-k8s-resources.sh).
   It will create the Installation on the resource cluster (shown in the first row of the diagram),
   as well as its Target and Context.

   
## Inspect the Result

On the resource cluster you can inspect the Installation:

```shell
❯ landscaper-cli inst inspect -n cu-example  podinfo
[✅ Succeeded] Installation podinfo
    └── [✅ Succeeded] Execution podinfo
        ├── [✅ Succeeded] DeployItem podinfo-item-1-vbll7
        ├── [✅ Succeeded] DeployItem podinfo-item-2-m5g6k
        └── [✅ Succeeded] DeployItem podinfo-item-3-wljnx
```

On the first target cluster:

```shell
❯ kubectl get secrets -n flux-system
NAME       TYPE     DATA   AGE
cluster2   Opaque   1      40m

❯ kubectl get gitrepositories -n flux-system
NAME         URL                                       AGE   READY   STATUS
podinfo      https://github.com/stefanprodan/podinfo   40m   True    stored artifact for revision '6.7.0@sha1:0b1481aa8ed0a6c34af84f779824a74200d5c1d6'

❯ kubectl get kustomizations -n flux-system
NAME                     AGE   READY
podinfo                  40m   True
```

On the second target cluster, the podinfo application should run in namespace `cu-podinfo`:

```shell
❯ kubectl get deployments -n cu-podinfo

NAME                      READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/podinfo   2/2     2            2           40m
```


## Cleanup

You can remove the Installation with the script
[commands/delete-installation.sh](commands/delete-installation.sh).

When the Installation is gone, you can delete the Context and Target with the script
[commands/delete-other-k8s-resources.sh](commands/delete-other-k8s-resources.sh).


<!-- References -->

[1]: https://github.com/stefanprodan/podinfo
[4]: https://fluxcd.io/flux/components/source/
[5]: https://fluxcd.io/flux/components/kustomize/
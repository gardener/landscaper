# Deployer Contract

Deployers need to follow some rules in order to work together with the landscaper. The purpose of this documentation is 
to define the contract between deployers and the landscaper so that developers who want to create their own deployer 
know which requirements they have to fulfill.

**Index**
- [What is a Deployer](#what-is-a-deployer)
- [Structure of a Deploy Item](#structure-of-a-deploy-item)
- [How is a Deployer expected to act?](#how-is-a-deployer-expected-to-act)
- [How is a Deployer installed](#how-is-a-deployer-installed)


## What is a Deployer?

A 'deployer' is basically a kubernetes controller that watches resources of the type `deployitems.landscaper.gardener.cloud`. 
_Deploy Items_ have a unique type (`.spec.type`) that describes the supported deployment or installation method and is 
also the identifier for the corresponding deployer.

For example: Among the basic deployers that come with the landscaper is a `helm` deployer, which reacts on deploy items 
of type `landscaper.gardener.cloud/helm` and is able to deploy, update, and delete helm charts. Another one would be 
the `manifest` deployer, which manages basic kubernetes manifests which are contained in the corresponding deploy items.


## Structure of a Deploy Item

### Spec

A deploy item looks like this:
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployItem
metadata:
  name: manifest-di
  namespace: default
spec:
  type: landscaper.gardener.cloud/kubernetes-manifest
  target:
    name: my-target
  context: "default" # reference to the Context object in the same namespace.
  config:
    apiVersion: manifest.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderConfiguration
    manifests:
    - apiVersion: v1
      kind: Namespace
      metadata:
        name: foo

```
This example shows a manifest deploy item that will create a namespace called `foo` when applied. Check out the 
[example deploy items](../../examples/deploy-items) for more examples of deploy items.

The type of the deploy item is defined in `spec.type`. It determines which deployer is responsible for this deploy item. 
The manifest deployer will only handle deploy items of type `landscaper.gardener.cloud/kubernetes-manifest` and it 
should be the only deployer that watches for this type.

Deployers may reference a target in `spec.target`. Targets usually contain credentials for accessing the environment 
that is targeted by this type of deploy item. The manifest deployer targets kubernetes clusters, so this target will 
contain a kubeconfig pointing to the cluster where the namespace should be created. Not all deploy items target 
kubernetes clusters though, for example the `terraform` deployer can also target IAAS accounts. There might be cases in 
which multiple deployers of the same type exists, e.g. if there is a fenced environment that is not accessible from the 
outside. In this case, the landscaper and a manifest deployer could run outside of it and another manifest deployer 
could run within the fenced environment to deploy manifests there. The target then determines which deployer handles 
which deploy item.

The content of `spec.config` depends on its type. It is only read and evaluated by the corresponding deployer. 
In this example the configuration for a manifest deploy item consists of a list of kubernetes manifests.

### Status

Once handled by its deployer, a status similar to this one will be attached to the deploy item:

```yaml
status:
  jobID: "fgkjjdfd..."
  jobIDFinished: "fgkjjdfd..."
  observedGeneration: 1
  phase: Succeeded
  lastReconcileTime: "2021-04-15T12:10:51Z"
  deployer:
    nname: "my-deployer"
    identity: "some unique identity"
    version: "v0.0.1"
  providerStatus:
    apiVersion: manifest.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderStatus
    managedResources:
    - policy: manage
      resource:
        apiVersion: v1
        kind: Namespace
        name: foo
        namespace: ""
```

If errors occurred while handling the deploy item, the status will contain an error message:

```yaml
status:
  jobID: "fgkjjdfd..."
  jobIDFinished: "fgkjjdfd..."
  observedGeneration: 1
  phase: Failed
  lastError:
    codes:
    - ERR_TIMEOUT
    lastTransitionTime: "2021-03-24T12:36:21Z"
    lastUpdateTime: "2021-03-24T12:36:21Z"
    message: no deployer has reconciled this deployitem within 300 seconds
    operation: WaitingForPickup
    reason: PickupTimeout
```

The most interesting part of the status is the `phase`, the `jobId`, the `jobIdFinished` and `lastReconcileTime`. 
The Landscaper interacts with the deployer using these fields. A deployer is only allowed to process a deploy item if 
`jobId` is not equal to `jobIdFinished`. This way the Landscaper informs the deployer about processing a deploy item. 

If `jobId` is not equal to `jobIdFinished` and `phase` is either empty, or has the value `Succeeded` or `Failed`, 
the deployer is allowed to process the deploy item. Then it must set `phase` on `Processing` or `Deleting` 
depending on if the item should be just reconciled or deleted/uninstalled. This signals the Landscaper that the deployer 
has picked up the deploy item. Furthermore, the deployer must set `lastReconcileTime` on the current time. The Landscaper 
checks a deploy item regularly and if `lastReconcileTime` is too old, it sees that the responsible deployer does not 
finish within the specified timeframe and sets the deploy item on failed (see [timeouts](../usage/DeployItemTimeouts.md)). 

When the reconcile or the deletion of the deploy item succeeded or failed, the deployer is required to set 
`phase` on `Succeeded` or `Failed` and `jobIdFinished` on the value of `jobId`. This must be done in one 
atomic update operation because it is required that if `jobId` and `jobIdFinished` are equal, the `phase` must 
be `Succeeded` or `Failed`. This informs the Landscaper, that the deployer has finished processing the deploy item and 
only then the Landscaper is allowed to trigger a new operation by updating the `jobId` again. 

The deletion of deploy items is triggered by the Landscaper. The deployers are only responsible to uninstall the deployed
artefacts and remove the finalizers of the deploy item if the uninstallation was successful.  

When the deployer starts working on a deploy item, it could use the status field `phase` to report its internal 
processing state, whereby the following values are allowed:

- `Init`: This is more of a transition phase that shows that the deploy item is about to be handled by a deployer.
- `Progressing`: The deploy item is currently being processed by its deployer.
- `InitDelete` Similar to `Init`, but for deletion.
- `Deleting`: Similar to `Processing`, but instead of being applied, the deploy item is being deleted.
- `Succeeded`: The deploy item successfully finished processing. 
- `Failed`: The deployer finished processing the deploy item, but it was not successful. Whenever this state is set, 
  there should be further information on what went wrong in the `status.lastError` field.
- `DeleteFailed`: Similar to `Failed`, but for deletion.

## How is a Deployer expected to act?

Not only a deployer, but also the landscaper interacts with deploy items. To avoid conflicts between deployers and the 
landscaper, the deployer is expected to follow these steps in the given order:

#### 1. Check the Type of the Deploy Item
A deployer's reconcile loop will be triggered for changes to any deploy item, not only the ones that are handled by 
this deployer. The deployer has to make sure that it only handles deploy items of its own type. A deployer must never 
modify a deploy item of another type in any way!

#### 2. Verify Target
As explained above, even if the type is correct, the deployer might still not be responsible for the deploy item, so the 
target has to be checked too, if any.

#### 3. Check for Need for Action
A deployer is only allowed to process a deploy item if `jobId` is not equal to `jobIdFinished`. The detailed protocol
between the landscaper and the deployer was described above.

#### 4. Deployer Logic and Status
Now the deployer should do its magic. First it must set the status field `lastReconcileTime` on the current time thereby
signaling the Landscaper that a deployer has picked up the deploy item.

As long as the deployer is actually doing something - or waiting for something - `phase` must be set on 
`Processing` or `Deleting` and `jobId` remains different from `jobIdFinished`. 

Some deployers need to store information in the deploy item's status during or after processing it.

A deploy item is deleted by Landscaper. The deployer see this at the deletion timestamp. In such a situation, the deployer
should uninstall the artefacts from the target and if this was successfull remove the finalizers from the deploy item.

There is the following important annotation that needs to be handled by the deployer in case of a deletion:
- `landscaper.gardener.cloud/delete-without-uninstall: true`: If this annotation is set at the deploy item and the deploy
  item is deleted by the Landscaper, the deployer should only remove the finalizer from the deploy item
  without uninstalling the deployed artefacts.

#### 5. Final State
If the deployer successfully finished the task described by the deploy item, the deployer is required to set 
`phase` on `Succeeded` and `jobIdFinished` on the value of `jobId`.

If it wasn't successful and has given up trying, `phase` has to be set on `Failed` (or `DeleteFailed`, respectively) and `jobIdFinished` 
on the value of `jobId`.

## How is a Deployer installed

A Deployer is basically a Kubernetes controller that watches DeployItems.
By default, it's up to the administrator to install and update the deployers.

As most deployer have a similar way to be installed, Landscaper offers a convenient way how to install and manage the 
complete lifecycle of a deployer. This LM (Lifecycle Management) also includes the management of different deployers 
across fenced environments.

A deployer has to implement the DLM contract (Deployer Lifecycle Management) to be managed by the landscaper.
For a technical overview about the DLM see [here](./deployer_lifecycle_management.md).

The DLM contract describes that the Installation of a Deployer has to be defined using a `Blueprint` (Component Descriptor + Blueprint).
By default, the agent comes with a helm deployer so that all deployers can be installed using deploy items of type 
`landscaper.gardener.cloud/helm`. If other deployitems are needed to install your deployer, another deployer 
registration should be created for that deployer. With that the deployers will install each other as long there are no 
cyclic dependencies between them.

The DLM offers some environment specific imports for the deployer blueprint:

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

imports:
- name: cluster # target to the host cluster
  type: target
  targetType: landscaper.gardener.cloud/kubernetes-cluster
- name: landscaperCluster # target to the cluster running the landscaper resources.
  type: target
  targetType: landscaper.gardener.cloud/kubernetes-cluster
  required: false
- name: releaseName
  type: data
  schema:
    type: string
- name: releaseNamespace
  type: data
  schema: 
    type: string
- name: identity
  type: data
  schema:
    type: string
- name: targetSelectors # defaulted to the "landscaper.gardener.cloud/environment" annotation
  type: data
  schema:
    type: array
    items:
      type: object
      properties:
        targets:
          type: array
          items:
            type: object
        annotations:
          type: array
          items:
            type: object
        labels:
          type: array
          items:
            type: object
```

Other imports can be freely used and configured using the `InstallationTemplate` in the `DeployerRegistration`.

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployerRegistration
metadata:
  name: my-deployer

spec:
  # describe the deploy items types the deployer iis able to reconcile
  types: ["my-deploy-item-type"]
  installationTemplate:
    componentDescriptor:
      ...
    blueprint:
      ...

    imports:
      data: [ ]
      targets: [ ]
    
    importDataMappings: {}
```


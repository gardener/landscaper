
# Deployer Contract

Deployers need to follow some rules in order to work together with the landscaper. The purpose of this documentation is to define the contract between deployers and the landscaper so that developers who want to create their own deployer know which requirements they have to fulfill.

**Index**
- [What is a Deployer](#what-is-a-deployer)
- [Structure of a Deploy Item](#structure-of-a-deploy-item)
- [How is a Deployer expected to act?](#how-is-a-deployer-expected-to-act)
- [How is a Deployer installed](#)


## What is a Deployer?

A 'deployer' is basically a kubernetes controller that watches resources of the type `deployitems.landscaper.gardener.cloud`. _Deploy Items_ have a unique type (`.spec.type`) that describes the supported deployment or installation method and is also the identifier for the corresponding deployer.
For example: Among the basic deployers that come with the landscaper is a `helm` deployer, which reacts on deploy items of type `landscaper.gardener.cloud/helm` and is able to deploy, update, and delete helm charts. Another one would be the `manifest` deployer, which manages basic kubernetes manifests which are contained in the corresponding deploy items.


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
    namespace: default
  config:
    apiVersion: manifest.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderConfiguration
    manifests:
    - apiVersion: v1
      kind: Namespace
      metadata:
        name: foo

```
This example shows a manifest deploy item that will create a namespace called `foo` when applied. Check out the [example deploy items](../../examples/deploy-items) for more examples of deploy items.

The type of the deploy item is defined in `spec.type`. It determines which deployer is responsible for this deploy item. The manifest deployer will only handle deploy items of type `landscaper.gardener.cloud/kubernetes-manifest` and it should be the only deployer that watches for this type.

Deployers may reference a target in `spec.target`. Targets usually contain credentials for accessing the environment that is targeted by this type of deploy item.
The manifest deployer targets kubernetes clusters, so this target will contain a kubeconfig pointing to the cluster where the namespace should be created. 
Not all deploy items target kubernetes clusters though, for example the `terraform` deployer can also target IAAS accounts.
There might be cases in which multiple deployers of the same type exists, e.g. if there is a fenced environment that is not accessible from the outside. In this case, the landscaper and a manifest deployer could run outside of it and another manifest deployer could run within the fenced environment to deploy manifests there. The target then determines which deployer handles which deploy item.

The content of `spec.config` depends on its type. It is only read and evaluated by the corresponding deployer. 
In this example the configuration for a manifest deploy item consists of a list of kubernetes manifests.

### Status

Once handled by its deployer, a status similar to this one will be attached to the deploy item:
```yaml
status:
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
  observedGeneration: 0
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

The most interesting part of the status is the `phase`. There are five different phases for deploy items:
- `Init`: This is more of a transition phase that shows that the deploy item is about to be handled by a deployer.
- `Progressing`: The deploy item is currently being processed by its deployer.
- `Deleting`: Similar to `Processing`, but instead of being applied, the deploy item is being deleted.
- `Succeeded`: The deploy item successfully finished processing. When this phase is set, landscaper expects all effects of the deploy item to have been applied.
- `Failed`: The deployer finished processing the deploy item, but it was not successful. Whenever this state is set, there should be further information on what went wrong in the `status.lastError` field.

`Succeeded` and `Failed` are treated as 'final states', because the corresponding deployer won't do anything with the deploy item unless being triggered, while `Init`, `Progressing`, and `Deleting` indicate that the deployer is still working on this deploy item.


## How is a Deployer expected to act?

Not only a deployer, but also the landscaper interacts with deploy items. To avoid conflicts between deployers and the landscaper, the deployer is expected to follow these steps in the given order:

#### 1. Check the Type of the Deploy Item
A deployer's reconcile loop will be triggered for changes to any deploy item, not only the ones that are handled by this deployer. The deployer has to make sure that it only handles deploy items of its own type. A deployer must never modify a deploy item of another type in any way!

#### 2. Verify Target
As explained above, even if the type is correct, the deployer might still not be responsible for the deploy item, so the target has to be checked too, if any.

#### 3. Handle Annotations
There are two important annotations that need to be handled by the deployer: 
The operation annotation `landscaper.gardener.cloud/operation` indicates that either a human operator or the landscaper wants a specific operation to be fulfilled on this deploy item. The value of the annotation specifies the expected operation:
- `reconcile`: This deploy item needs to be reconciled. The deployer has to remove this annotation. In addition, it should set the deploy item's phase to `Init` to show the beginning of a new reconciliation and avoid loss of information in case the deployer dies immediately after removing the annotation. In the status, `lastReconcileTime` has to be set to the current timestamp (this value is used to recognize when a deployer is 'stuck' processing a deploy item).
- `abort`: This annotation is usually attached to deploy items in the `Progressing` phase and means that the deployer should stop processing it. The main purpose of this annotation is to give the deployer time to gracefully stop processing the deploy item and clean up any already created resources before setting the phase to `Failed`. What 'aborting gracefully' means is highly specific to the corresponding deployer logic.

> The landscaper will abort deploy items which are stuck in `Progressing` for too long. The timeout can be configured on the deploy item itself via `spec.timeout` and is defaulted to 10 minutes otherwise. The default can be overwritten by setting `deployItemTimeouts.progressingDefault` in the landscaper configuration. Instead of a time, `none` can be used to disable this check.

> There is also a timeout for deploy items which take too long to abort. This is tracked via a timestamp annotation `landscaper.gardener.cloud/abort-time` which is set by the landscaper together with the abort operation annotation. After the specified time, landscaper will set the deploy item to `Failed`. The timeout can be configured via `deployItemTimeouts.abort` in the landscaper configuration (use `none` to disable, as above). It defaults to 5 minutes.

The second important annotation is `landscaper.gardener.cloud/reconcile-time`. The landscaper adds this annotation - with the current time as value - whenever it expands an `execution` into its deploy items. If this annotation is still present after a defined time, this is interpreted as no deployer having picked up this deploy item and the landscaper will set its phase to `Failed`. Deployers are expected to remove this annotation whenever they start reconciling a deploy item they are responsible for.

> The pickup timeout duration defaults to 5 minutes and can be configured by setting `deployItemTimeouts.pickup` in the landscaper configuration. As for the other timeouts, checking for pickup timeouts can also be disabled by setting the aforementioned value to `none`.

#### 4. Handle Generation
Another indicator that something needs to be done is when `status.observedGeneration` differs from `metadata.generation`. The latter one changes every time the `spec` is modified and a difference in both shows that the deployer has not yet reacted on the latest changes to this deploy item. For this logic to work, the deployer has to set `status.observedGeneration` to the deploy item's generation at the beginning of the reconcile loop. Similarly to the reconcile annotation, the deployer should set the phase of the deploy item to `Init` if it updated the observed generation.

> There is an auxiliary method `HandleAnnotationsAndGeneration` that handles steps 3 and 4 [defined here](../../pkg/deployer/lib/utils.go).

#### 5. Check for Need for Action
For most deployers, there probably isn't anything to do now if the deploy item is still in a final state (phase `Succeeded` or `Failed`) - it was finished before and nothing has changed, so the reconcile can be aborted at this point. Please note that this does not apply to all deployers and only works if the phase is actually set to `Init` when a reconcile annotation or an outdated observed generation is found.

#### 6. Deployer Logic and Status
Now the deployer should do its magic. As long as it is actually doing something - or waiting for something - the deploy item's phase should be `Processing` (or `Deleting`, if it is handling the deletion of the deploy item).
Some deployers need to store information in the deploy item's status during or after processing it.

#### 7. Final State
If the deployer successfully finished the task described by the deploy item, it must set the phase to `Succeeded`, if it wasn't successfuly and has given up trying, the phase has to be `Failed` instead.

## How is a Deployer installed

A Deployer is basically a Kubernetes controller that watches DeployItems.
By default, it's up to the administrator to install and update the deployers.

As most deployer have a similar way to be installed, Landscaper offers a convenient way how to install and manage the complete lifecycle of a deployer.
This LM (Lifecycle Management) also includes the management of different deployers across fenced environments.

A deployer has to implement the DLM contract (Deployer Lifecycle Management) to be managed by the landscaper.
For a technical overview about the DLM see [here](./deployer_lifecycle_management.md).

The DLM contract describes that the Installation of a Deployer has to be defined using a `Blueprint` (Component Descriptor + Blueprint).
By default, the agent comes with a helm deployer so that all deployers can be installed using deploy items of type `landscaper.gardener.cloud/helm`.
If other deployitems are needed to install your deployer, another deployer registration should be created for that deployer.
With that the deployers will install each other as long there are no cyclic dependencies between them.

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

Other imports can be freely used and configured using the `InstllationTemplate` in the `DeployerRegistration`.

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


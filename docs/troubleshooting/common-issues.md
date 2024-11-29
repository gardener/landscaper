---
title: Common Issues
sidebar_position: 2
---

# Common Issues

This document lists issues and how to fix them.

- [Unable to resolve component descriptor](#unable-to-resolve-component-descriptor)  
- [Not found (ResolveBlueprint)](#not-found-resolveblueprint)  
- [Unable to template executions](#unable-to-template-executions)  
- [Unable to fetch data object](#unable-to-fetch-data-object)  
- [Target not found](#target-not-found)  
- [Execution is not finished yet](#execution-is-not-finished-yet)
- [Failed Installation without error message](#failed-installation-without-error-message)  
- [Timeout](#timeout)
- [Invalid configuration: no configuration has been provided, try setting KUBERNETES_MASTER environment variable](#invalid-configuration-no-configuration-has-been-provided-try-setting-kubernetes_master-environment-variable)
- [Kubernetes cluster unreachable](#kubernetes-cluster-unreachable)
- [Unable to install/upgrade helm chart release](#unable-to-installupgrade-helm-chart-release)


## Unable to resolve component descriptor

#### Symptom

The status of an Installation shows the following error:

```yaml
status:
  lastError:
    message: 'unable to resolve component descriptor for ref &v1alpha1.ComponentDescriptorReference{RepositoryContext:(*v2.UnstructuredTypedObject)(0xc00086d9e0),
      ComponentName:"github.com/gardener/landscaper-examples/guided-tour/helm-chart",
      Version:"1.0.99"}: component version "github.com/gardener/landscaper-examples/guided-tour/helm-chart:1.0.99"
      not found: oci artifact "europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper-examples/examples/component-descriptors/github.com/gardener/landscaper-examples/guided-tour/helm-chart:1.0.99"
      not found in component-descriptors/github.com/gardener/landscaper-examples/guided-tour/helm-chart'
    operation: InitPrerequisites
    reason: ResolveBlueprint
```

#### Reason

An Installation references a component version in field `spec.componentDescriptor.ref`:

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
spec:
  componentDescriptor:
    ref:
      componentName: github.com/gardener/landscaper-examples/guided-tour/helm-chart
      version: 1.0.99
```

The error occurs if this component version can not be found.
Possible reason for the error are:

- The component version might not exist. You can check the existence of a component version for example with the 
command [ocm get component][1] of the OCM CLI:

  ```shell
  ❯ ocm get component <BASE_URL>//<COMPONENT_NAME>:<COMPONENT_VERSION>
  
  # Example
  ❯ ocm get component europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper-examples/examples//github.com/gardener/landscaper-examples/guided-tour/helm-chart:1.0.0
  
  COMPONENT                                                      VERSION PROVIDER
  github.com/gardener/landscaper-examples/guided-tour/helm-chart 1.0.0   internal
  ```
  Note the double slash separating base url and component name in this command.

- The component name in the Installation might be wrong. Check field `spec.componentDescriptor.ref.componentName`.

- The component version in the Installation might be wrong. Check field `spec.componentDescriptor.ref.version`.

- The base URL of the OCM repository might be wrong. In the Context resource referenced by the Installation, check the
  base URL of the OCM repository.


## Not found (ResolveBlueprint)

#### Symptom

The status of an Installation shows the following error:

```yaml
status:
  lastError:
    message: not found
    operation: InitPrerequisites
    reason: ResolveBlueprint
```

#### Reason

The Installation specifies a blueprint in field `spec.blueprint.ref.resourceName`:

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
spec:
  blueprint:
    ref:
      resourceName: blueprint 
```

The error occurs if the blueprint is not found, i.e. if the OCM component version has no resource with the in field 
`spec.blueprint.ref.resourceName`. 

Check the resource name, and check whether a resource with that name exists. You can check the existence of a resource
with the command [ocm get resources][2] of the OCM CLI:

```shell
❯ ocm get resources <BASE_URL>//<COMPONENT_NAME>:<COMPONENT_VERSION>

# Example
❯ ocm get resources europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper-examples/examples//github.com/gardener/landscaper-examples/guided-tour/helm-chart:1.0.0

NAME              VERSION IDENTITY TYPE                                RELATION
blueprint         1.0.0            landscaper.gardener.cloud/blueprint local
echo-server-chart 1.0.0            helmChart                           external
echo-server-image v0.2.3           ociImage                            external
```


## Unable to template executions

#### Symptom

The status of an Installation shows the following error:

```yaml
status:
  lastError:
    codes:
      - ERR_FOR_INFO_ONLY
    message: "Op: CreateImportsAndSubobjects - Reason: ReconcileExecution - Message:
      Op: RenderDeployItemTemplates - Reason: Template - Message: unable to template
      executions: ......"
    operation: handlePhaseInit
    reason: CreateImportsAndSubobjects
```

#### Reason

During the processing of an Installation, its DeployItems and subinstallations are templated.
The above error occurs if this templating fails. Check the templates of the DeployItems and subinstallations.
The templates are part of the blueprint. Normally, the error message contains a hint what is wrong.


## Unable to fetch data object

#### Symptom

The status of an Installation shows the following error:

```yaml
status:
  lastError:
    message: 'unable to fetch data object my-release (/my-release): dataobjects.landscaper.gardener.cloud
      "my-release" not found'
    operation: init
    reason: ImportsSatisfied
```

#### Reason

The Installation tries to read the value of an import parameter from a DataObject, but the DataObject does not exist.

The DataObjects for import parameters are specified in `spec.imports.data[].dataRef`:

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
spec:
  imports:
    data:
      - name: release        # name of an import parameter of the blueprint
        dataRef: my-release  # name of a DataObject containing the parameter value
```

- Check that field `spec.imports.data[].dataRef` contains the correct name.
- Check that the DataObject exists. It must belong to the same namespace as the Installation.
- Import values can also be read from Secrets and ConfigMaps. Check that the correct key is used: `dataRef`, `secretRef`,
  resp. `configMapRef`.


## Target not found

#### Symptom

The status of an Installation shows the following error:

```yaml
status:
  lastError:
    message: targets.landscaper.gardener.cloud "my-cluster" not found
    operation: init
    reason: ImportsSatisfied
```

#### Reason

The Installation tries to read a target import parameter from a Target, but the Target custom resource does not exist.

The Targets of an Installation are specified in `spec.imports.data[].dataRef`:

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
spec:
  imports:
    targets:
      - name: cluster       # name of an import parameter of the blueprint
        target: my-cluster  # name of a Target
```

- Check that field `spec.imports.targets[].target` contains the correct name.
- Check that the Target exists. It must belong to the same namespace as the Installation.


## Execution is not finished yet

#### Symptom

The status of an Installation shows the following error:

```yaml
status:
  lastError:
    codes:
      - ERR_UNFINISHED
      - ERR_FOR_INFO_ONLY
      - ERR_NO_RETRY
    message: execution cu-example / echo-server is not finished yet
    operation: handlePhaseProgressing
    reason: JobIDFinished
  phase: Progressing
```

#### Reason

In this case the Installation is not in an error state, but waiting for its subobjects (subinstallations, Executions, 
DeployItem) to finish. For example the processing of a DeployItem might take some time.
If the state persists longer than expected, check the status of the subobjects.


## Failed Installation without error message

#### Symptom

An Installation is in phase `Failed`, but its status does not contain a `lastError`.

```yaml
status:
  phase: Failed
```

#### Reason

In this case the error happened in another object, for example a DeployItem of the Installation.
Use the command `landscaper-cli installation inspect ...` to display the tree of all involved objects and to find out
which of them failed.


## Timeout

#### Symptom

The status of a DeployItem contains a last error with message `timeout at: ...`:

```yaml
status:
  lastError:
    codes:
      - ERR_TIMEOUT
    message: 'timeout at: "helm deployer: start reconcile"'
    operation: StandardTimeoutChecker.TimeoutExceeded
    reason: ProgressingTimeout
  lastErrors:
    - ...
    - ...
    - message: 'Op: TemplateChart - Reason: GetTargetClient - Message: invalid configuration:
      no configuration has been provided, try setting KUBERNETES_MASTER environment
      variable'
      operation: Reconcile
      reason: Template
    - codes:
        - ERR_TIMEOUT
      message: 'timeout at: "helm deployer: start reconcile"'
      operation: StandardTimeoutChecker.TimeoutExceeded
      reason: ProgressingTimeout
```

#### Reason

Usually the timeout is not the root cause of the failure. Check the list of `lastErrors` in the status of the
DeployItem, and there the entries before the last one.


## Invalid configuration: no configuration has been provided, try setting KUBERNETES_MASTER environment variable

#### Symptom

The `lastError` or one of the `lastErrors` in the status of a DeployItem contains the following message:

```yaml
status:
  lastError:
    - message: '... - Message: invalid configuration: no configuration has been provided, try setting KUBERNETES_MASTER environment variable'
```

#### Reason

There is a problem with the kubeconfig in the Target of the DeployItem.

- Check whether you can use the kubeconfig to access the cluster without the Landscaper, for example with a simple
`kubectl` command like `kubectl get namespaces`.

- Check the formatting of the kubeconfig in the Target, in particular the indentation of the lines.


## Kubernetes cluster unreachable

#### Symptom

The `lastError` or one of the `lastErrors` in the status of a DeployItem contains the following message:

```yaml
status:
  lastError:
    message: 'Kubernetes cluster unreachable: ...'
```

#### Reason

There is a problem with the kubeconfig in the Target of the DeployItem.

- Check that the cluster exists and is not hibernated.
- Check whether you can use the kubeconfig to access the cluster without the Landscaper, for example with a simple
  `kubectl` command like `kubectl get namespaces`.
- Note that Landscaper can not work with a Gardenlogin/OIDC kubeconfig. 
  [Constructing a Target Resource](../guided-tour/targets/README.md) describes how to get a kubeconfig based on a 
  ServiceAccount, which you can use with the Landscaper.


## Unable to install/upgrade helm chart release

#### Symptom

The `lastError` or one of the `lastErrors` in the status of a DeployItem contains one of the following messages:

```yaml
status:
  lastError:
    message: 'unable to install helm chart release: ...'
    operation: InstallHelmRelease
    reason: Install
```

or

```yaml
status:
  lastError:
    message: 'unable to upgrade helm chart release: ...'
    operation: UpgradeHelmRelease
    reason: Update
```

#### Reason

This error indicates that the `helm install` or `helm upgrade` operation failed, that was executed by the helm deployer. 

#### Example

A `helm install` or `helm upgrade` operation can fail for various reasons. One possibility is that the templating of 
an object in the helm chart with the provided helm values failed. Here is an example how the full error message could
look like in such a case:

```yaml
    message: 'unable to install helm chart release: template: hello-world/templates/configmap.yaml:7:22:
      executing "hello-world/templates/configmap.yaml" at <.Values.testData.field1.field2>:
      can''t evaluate field field1 in type interface {}'
```


<!-- References -->

[1]: https://ocm.software/docs/cli-reference/get/componentversions/
[2]: https://ocm.software/docs/cli-reference/get/resources/

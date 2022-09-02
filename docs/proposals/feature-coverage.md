# Feature Coverage of the Integration Tests


### Imports

#### Where Imported Values Come From

An `Installation` can import values from various objects. An import of type `data` can come from:

- a DataObject
- the complete data map of a `ConfigMap`
- one data item of a `ConfigMap`
- the complete data map of a `Secret`
- one data item of a `Secret`

An import of type `target` can come from:

- a Target that contains a kubeconfig
- a Target that contains a Secret ref (not supported by the container deployer)

An import of type `targetList` can come from

- a list of `Target` objects
- a `TargetListReference` (when a subinstallation gets a `TargetList` from its parent)


#### Import Data Mapping

Import data mapping that transforms imported values
Special case: hard-coded values

Default import data mapping

#### Validating Values against the Import Definition in the Blueprint

Required and optional imports

Type validation

#### Using Imports

Imports can be passed to subinstallations

Imports can be templated into deloy items

Imports can be used in an export data mapping

#### Update Scenarios

Changed import values (data and targets). New values must be passed to subinstallations and deploy items.

Update of imports:
- adding a new import parameter
- removing an import parameter
- importing first from ConfigMap, and then from Secret (perhaps also switch between ConfigMap and DataObject etc.)
- changing the name of a ConfigMap, Secrets, DataObject and TargetListReference



### Exports

Export execution of the blueprint

Type validation

Export data mapping of the installation

An export value can be written to a `DataObject` or `Target`.

#### Update scenarios

Changed export values (data and targets). New values must be passed upwards.

Update of exports:
- adding new export parameter
- removing an export parameter
- changing the name of a `Target` or `DataObject`

### Root Installations

The reconciliation of a root installation does not start without reconcile annotation.

A reconciliation can be triggered by a reconcile annotation.

A reconciliation can be interrupted by an interrupt annotation.

A root installation triggers its successors when it has succeeded.



### Subinstallations

An installation can have subinstallations

A subinstallation can import values from its parent

A subinstallation can import values from a sibling

The processing order of siblings is determined by the exports and imports

Update Scenarios:
- Adding subinstallations
- Removing subinstallations
- Changing a subinstallation (e.g. increasing the blueprint version)



### Deploy Items

Different deployers (helm, manifest, container)

Dependencies between the deploy items

Update Scenarios:
- Adding deploy items
- Removing deploy items
- Changing dependencies
- Changing the deploy item (e.g a helm chart version)



### Component Descriptor and Blueprint Definition

The component descriptor can be given:
- inline 
- or by reference (field `spec.componentDescriptorRef`).  
  In this case, the repository context can be given 
  - in field `spec.componentDescriptorRef.repositoryContext`
  - in field `repositoryContext` of the `Context` referenced in the `Installation`

The blueprint of an installation can be given:
- inline
- or by referencing a resource in the component descriptor

Subinstallations, deploy executions, export executions can be defined in separate files or in the blueprint.yaml

Pulling a blueprint, component descriptor or jsonschema using a registry pull secret
- from field `registryPullSecrets` of the installation spec
- from field `registryPullSecrets` of the `Context` referenced in the `Installation`



### Manifest Deployer

Creation of manifests

Readiness checks

Export of values from resources on the target cluster. The resource on the target cluster can be referenced
in one step (with `fromResource`) or in two steps (with `fromResource` and `fromObjectRef`).

Update scenarios, depending on the `policy` specified for a manifest



### Helm Deployer

Deployment of a helm chart

Readiness checks

Export of values from resources on the target cluster. The resource on the target cluster can be referenced
in one step (with `fromResource`) or in two steps (with `fromResource` and `fromObjectRef`).

Helm deployment configuration (`atomic`, `timeout`)

Real helm deployment vs. applying manifests

Public vs. private helm chart repo (with credentials in a `Context` resource)

Update scenarios



### Container Deployer

Running a program with the container deployer

The program can import and export parameters (using the environment variables `IMPORTS_PATH`, `EXPORTS_PATH`)

The program can distinguish between the operations `RECONCILE` and `DELETE` (environment variables `OPERATION`)

The program can store data as "state" between reconciliations (environment variables `STATE_PATH`)

The program can access component descriptors (environment variables `COMPONENT_DESCRIPTOR_PATH`)

The program can access the content of the blueprint directory (environment variables `CONTENT_PATH`)

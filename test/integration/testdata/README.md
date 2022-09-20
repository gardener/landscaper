# Integration Test Scenarios

## Import Scenarios

The blueprint of component [import-export/v0.1.0](import-export/v0.1.0) uses the mainfest deployer to create a
ConfigMap. Name, namespace, and data of the ConfigMap can be provided by imports of the blueprint.
The blueprint exports the imported name and data slightly modified.
Installations that use this blueprint can be concatenated and are also used in the subinstallation scenarios.

The blueprint of component [import-export/v0.2.0](import-export/v0.2.0) is just a variation of `v0.1.0`, so that
a version update can be tested.

The blueprint of component [import-export/v0.3.0](import-export/v0.3.0) is just a variation of `v0.1.0`.
The imported name and data are optional and have default values. This scenario does not work and is not used in any 
test case (issue: Default Values for Optional Imports of a Blueprint #172).


### Reading Import Values from Different Objects

[**Installation import-export/installation-1**](./import-export/installation-1/installation.yaml) 
reads the values for import parameters from `DataObjects`.

[**Installation import-export/installation-2**](./import-export/installation-2/installation.yaml) 
reads the values for import parameters from `ConfigMaps`.

[**Installation import-export/installation-3**](./import-export/installation-3/installation.yaml) 
reads the values for import parameters from `Secrets`.


### Import Data Mappings

[**Installation import-export/installation-4**](./import-export/installation-4/installation.yaml) 
reads values from `DataObjects` and transforms them in an import data mapping before passing them to a blueprint.
- Import parameter `configmapNameIn` of the blueprint is set to a constant value.
- Import parameter `configmapData1` is computed in a template from values that have been read from `DataObjects`.
- Import parameter `configmapNamespaceIn` of the blueprint is directly taken from a `DataObject` without import data 
  mapping.

### Validation of Imports

[**Installation import-export/installation-5-neg**](./import-export/installation-5/installation-neg.yaml)
does not provide a required import parameter of the blueprint and will therefore fail.
This is fixed in [Installation import-export/installation-5-pos](./import-export/installation-5/installation-pos.yaml).

[**Installation import-export/installation-6-neg**](./import-export/installation-6/installation.yaml)
provides an import value of the wrong type (boolean instead of string) and will therefore fail.
This is fixed by replacing the DataObject.


## Subinstallation Scenarios

The blueprints in this section have subinstallations. All subinstallations use the import-export component, which 
deploys a ConfigMap.

- Blueprint `v0.1.0` has one such subinstallation, and therefore deploys one ConfigMap.
- Blueprint `v0.2.0` has two subinstallations, and therefore deploys two ConfigMaps.
- Blueprint `v0.3.0` has three subinstallations, and therefore deploys three ConfigMaps.
- Blueprint `v0.4.0` has is the same as `v0.3.0`, except that two of the three subinstallations use another version of 
  the import-export component, resulting in different names of the deployed ConfigMaps.

#### Installation 1: Add and Remove Subinstallations

The following root installations use the corresponding blueprints with 1, 2, resp. 3 subinstallations:

- [subinstallations/installation-1/installation-v0.1.0.yaml](./subinstallations/installation-1/installation-v0.1.0.yaml),
- [subinstallations/installation-1/installation-v0.2.0.yaml](./subinstallations/installation-1/installation-v0.2.0.yaml),
- [subinstallations/installation-1/installation-v0.3.0.yaml](./subinstallations/installation-1/installation-v0.3.0.yaml)

Test case [subinstallations/subinstallations.go](../subinstallations/subinstallations.go)
starts with the root installation with two subinstallations.
Then it updates it to the root installation with three subinstallations.
Finally, it updates it again to the root installation with one subinstallation. In this way the addition and removal
of subinstallations is tested.

#### Installation 2: Update Subinstallations

The following root installations use the corresponding blueprints with 3 subinstallations:

- [subinstallations/installation-2/installation-v0.3.0.yaml](./subinstallations/installation-2/installation-v0.3.0.yaml),
- [subinstallations/installation-2/installation-v0.4.0.yaml](./subinstallations/installation-3/installation-v0.4.0.yaml)

Test case [subinstallations/subinstallations.go](../subinstallations/subinstallations.go)
starts with the `v0.3.0` root installation and updates it to the `v0.4.0` root installation.
In this way the update of subinstallations to another blueprint version is tested.

#### Installation 3: Update Data Imports

We use again a root installations with 3 subinstallations:

- [subinstallations/installation-3/installation.yaml](./subinstallations/installation-3/installation.yaml),

Test case [subinstallations/subinstallations.go](../subinstallations/subinstallations.go)
deploys the root installation, then updates the values in the import DataObjects, and triggers another
reconciliation. It is checked that the updated import values are used.

## Dependency Scenarios

The blueprints in this section are used to check the order in which subinstallations are processed.

The blueprint of the base component [dependencies/component-base](dependencies/component-base) simply imports a 
string and exports it unchanged. 
Remark: the blueprint has neither deploy items, not subinstallations (that's possible).

The aggregated components [dependencies/component-aggr-v0.1.0](dependencies/component-aggr-v0.1.0) and
[dependencies/component-aggr-v0.2.0](dependencies/component-aggr-v0.2.0) have subinstallations using the base 
component. Each subinstallation reads the strings exported by its predecessors and uses an import data mapping to 
combine them and to append the own name. In this way the strings that are passed through the subinstallations track 
the processing order. Version `v0.1.0` consists of a chain of four subinstallations. Version `v0.2.0` consists of 
three independent subinstallations, and a fourth one that depends on the first three.

## Target Scenarios

The blueprint of component
[github.com/gardener/landscaper/integration-tests/target-exporter](targets/component-target-exporter)
imports a string and exports a `Target` with the imported string as kubeconfig.

The blueprint of component
[github.com/gardener/landscaper/integration-tests/target-importer-1](targets/component-target-importer-1)
imports a `Target` and a `TargetList` (both optional). It writes their kubeconfigs into one `ConfigMap`.

The blueprint of component
[github.com/gardener/landscaper/integration-tests/target-importer-2](targets/component-target-importer-2)
imports a `TargetList` and generates a `DeployItem` for each `Target` of the list. Each of the DeployItems creates a
`ConfigMap`.

The blueprint of component
[github.com/gardener/landscaper/integration-tests/target-root-1](targets/component-target-root-1)
imports a `Target` and a `TargetList` and passes them to one subinstallation with the target-importer-1 component.  

The blueprint of component
[github.com/gardener/landscaper/integration-tests/target-root-2](targets/component-target-root-2)
imports a `TargetList` and generates for each `Target` of the list a subinstallation with the target-importer-1 
component. This scenario does not work and is not used in any test case 
(issue: Templating a Subinstallation for Every Target of a TargetList #171).

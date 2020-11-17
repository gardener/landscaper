# Glossary

#### _Installation_

  An installation is the parameterized instance of a Blueprint deployed by a user.
  Landscaper acts upon creation or update and creates or updates dependent [Execution](#_execution_) and [Subinstallations](#_subinstallations).

#### _Subinstallations_

  Subinstallations are Installations that are automatically created by the landscaper as part of a running installation (the installation references a [Aggregated Blueprint](#_aggregated-blueprint_)).

  __Background Knowledge__:
  <details>
    Subinstallations define the usage of other blueprints within an [Aggregated Blueprint](#_aggregated-blueprint_).
    Subinstallations can be nested, when deployed, they are managed by their parent (sub)installation.
  </details>

#### Sibling Installations

  Sibling Installations refer to Installations belonging to the same parent.

#### _Execution_

#### _Blueprint_

  Blueprints contain actual intructions and steps on how to install a software component and what is needed to perform these actions.

  __Background Knowledge__:
    <details>
    Blueprints consists of:
      - Configuration Data ([Imports](#_import_))
      - Installation intructions
        - [DeployItems](#_deploy-items_) or
        - [Subinstallations](#_subinstallations_)
      - [Output](#_export_)
    </details>

#### _Aggregated Blueprint_

  Aggregated Blueprints are Blueprints that bundle multiple other blueprints.
  They contain intruction how they these referenced blueprints interact with each other.

  Their pratical use is to install mutliple components that depend on each other.

#### _Import_

  `Import` has 2 ambigious meanings, whether we are talking about Blueprints or Installations.

  ##### Blueprint

  Imports declare what data will be required to process the Blueprint. Part of the declaration is also the format, which can either be of type [Target](#_target_) or any valid jsonschema.

  ##### Installation

  Imports in an Installation assign acutal values and make them accessible for further processing. They satisfy the requirements (imports) defined in the Blueprint.

#### _Export_

  `Export` has 2 ambigious meanings, whether we are talking about Blueprints or Installations.

  ##### Blueprint

  Exports declare the output expected from a processed Blueprint.

  ##### Installation

  Exports of an Installation pick up actual values and make them accessible for a user, parent or sibling Installation.

  __Background Knowledge__:
    <details>
    Parent Installations can use exports of their subinstallations as their own export.
    They cannot be used as inputs for their deploy items.
    </details>


#### _DataObject_

  DataObjects are vehicles to store arbitrary kinds of data. They exist in a [Context](#_context_) and provide data to Imports / receive data from Exports. They can be considerd to be the implementation of the data flow in an installation. 

#### _Target_

  A Target defines the system in which Landscaper will run the installation steps. Target resources contain all relevant data to access this environment including credentials. 

#### _DeployItem_

  A DeployItem is the interface between the Landscaper controller and the [Deployers](#_deployer_). It contains input data and a set of Deployer-specific instructions on how to install a component (e.g. install a helm chart with some custom values). Additionally, it is used to record the status as returned by the Deployer.

#### _Deployer_

  Deployer are highly specialized controllers that act on [DeployItems](#_deployitem_) of a certain type. They execute the installation instructions and aim to maintain the declared desired state.

#### _Context_

  A context defines the scope in which an Installation runs and all of its data lives.
  For every Installation a dedicated context is created and data can only be accessed within the same context.
  Data can be exchanged between contexts via Import and Export declarations.

  Since Installations can be nested, the resulting Contexts are nested as well.

  For more information see [here](./Context.md).

#### _DeployExecutions_

  A list of templates to generate [deploy items](#_deployitem_) as part of a [Blueprint](#_blueprint_). Mainly used to describe the installation instructions and customize it using the data provided as [imports](#_import_).

#### _ExportExecutions_

  A list of templates describing how to generate [exports](#_export_) as part of a [Blueprint](#_blueprint_).
  These templates contain the instructions which data gets written into the exports and how it might be preprocessed.
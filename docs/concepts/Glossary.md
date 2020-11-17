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

#### _Target_

#### _DeployItem_

#### _Context_

  A context defines the scope in which an Installation runs and all of its data lives.
  Every Installation has its own dedicated context and data can only be accessed within the same context.
  Data can be exchanged between contexts via Import and Export declaration.

  <!-- todo rephrase -->
  Consequently, contexts follow the same tree structure as their blueprint/subinstallation.

  For more information see [here](./Context.md).

#### _DeployExecutions_

  A list of templates to generate [deploy items](#_deployitem_) as part of a [Blueprint](#_blueprint_). Mainly used to describe the installation instructions and customize it using the data provided as [imports](#_import_).

#### _ExportExecutions_

  A list of templates describing how to generate [exports](#_export_) as part of a [Blueprint](#_blueprint_).
  These templates contain the instructions which data gets written into the exports and how it might be preprocessed.
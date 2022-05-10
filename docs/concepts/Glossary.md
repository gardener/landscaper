# Glossary

  - [_Aggregated Blueprint_](#aggregated-blueprint)
  - [_Blueprint_](#blueprint)
  - [_Component Descriptor_](#component-descriptor)
  - [_Context_](#context)
  - [_DataObject_](#dataobject)
  - [_Deployer_](#deployer)
  - [_DeployExecution_](#deployexecution)
  - [_DeployItem_](#deployitem)
  - [_Execution_](#execution)
  - [_Export_](#export)
    - [Blueprint Export](#blueprint-export)
    - [Installation Export](#installation-export)
  - [_ExportExecution_](#exportexecution)
  - [_Import_](#import)
    - [Blueprint Import](#blueprint-import)
    - [Installation Import](#installation-import)
  - [_Installation_](#installation)
  - [_Sibling Installations_](#sibling-installations)
  - [_Sub-Installation_](#sub-installation)
  - [_SubinstallationExecution_](#subinstallationexecution)
  - [_Target_](#target)
  - [_Template_](#template)

#### _Aggregated Blueprint_

  Aggregated Blueprints are Blueprints that bundle the execution of multiple
  other blueprints.
  They contain an orchestration pattern for nested [_Installations_](#installation).

  Their practical use is to install multiple components that depend on each other.

[↑](#glossary)

#### _Blueprint_

Blueprints are entities, which contain the instructions and steps about how to deploy a software component in the form of [_DeployItems_](#deployitem). As these instructions are expected to be idempotent, Blueprints can be defined as descriptions of the target state for the software component. These Blueprints define an interface for import data, as required by its DeployItems. It can also define an export interface to expose output data of its DeployItems.

In other words, Blueprints are reusable collections of installation tasks, which could be parameterized through import parameters. Several DeployItems can be defined in a Blueprint at once. Also, the execution order of these DeployItems in a Blueprint can be specified.

Blueprints in a nutshell:
  - Configuration Data ([Imports](#import))
  - Installation instructions
    - [DeployItems](#deployitem) or
    - [Sub-Installations](#sub-installation)
    - [Output](#export)

[↑](#glossary)

#### _Component Descriptor_
  A Component Descriptor contains references and locations to all resources that are used by Landscaper to deploy and install an application.
  Typically, a Component Descriptor is stored in an OCI registry.

  For more details see [here](https://gardener.github.io/component-spec/format.html) and [here](https://gardener.github.io/component-spec/semantics.html).

[↑](#glossary)

#### _Context_

  The Landscaper defines 2 different kind of contexts.

  One context is an actual resource that is referenced by an Installation and defines common configuration.

  The other context is a logical object that defines the scope in which an Installation runs and all of its data lives.
  For every Installation a dedicated logical context is created and data can only be accessed within the same context.
  Data can be exchanged between contexts via Import and Export declarations.

  Since Installations can be nested, the resulting contexts are nested as well.

[↑](#glossary)

#### _DataObject_

  DataObjects are vehicles to store arbitrary kinds of data. They exist in a [Context](#context) and provide data to Imports / receive data from Exports. They can be considered to be the implementation of the data flow in an installation.

[↑](#glossary)

#### _Deployer_

  Deployer are highly specialized controllers that act on [DeployItems](#deployitem) of a certain type. They execute the installation instructions and aim to maintain the declared desired state.

[↑](#glossary)

#### _DeployExecution_

  A _DeployExecution_ is a dedicated instantiation of a [template](#template) to generate [deploy items](#deployitem) as part of a [Blueprint](#blueprint). Mainly used to describe the installation instructions and customize it using the data provided as [imports](#import).
  It is used in list of such execution under the field `deployExecutions` in a blueprint descriptor.

[↑](#glossary)

#### _DeployItem_

The main objective of Landscaper is to set up large cloud environments with complex relationships consisting of many small installation tasks like Helm deployments, network setup etc. These elementary installation tasks are defined by DeployItems. There are different types of DeployItems for example for deploying Helm charts, plain kubernetes manifests or terraform configurations.

A DeployItem can be described as the interface between the Landscaper controller and the [Deployers](#deployer). It contains input data and a set of Deployer-specific instructions on how to install a component (e.g. install a helm chart with some custom values). Additionally, it is used to record the status as returned by the Deployer.

These DeployItems can be configured such that they can be used in different scenarios. Typical examples of configurable parts of a DeployItem are the target cluster or namespace where a helm chart should be deployed to. A DeployItem can also define output parameters for data it creates and which can be consumed by others.

[↑](#glossary)

#### _Execution_
  
  An _Execution_ describes the instantiation of a [template](#template) in a [_Blueprint_](#blueprint).
  There are several purposes for those templates: 
  - [DeployExecutions](#deployexecution) are used to render [_DeployItems](#deployitem)
  - [ExportExecutions](#exportexecution) are used to render exports of a [_Blueprint_](#blueprint).
  - [SubinstallationExecutions](#subinstallationexecution) are used to render nested [_Installations_](#installation)

[↑](#glossary)

#### _Export_

  `Export` has 2 ambiguous meanings, whether we are talking about Blueprints or Installations.

[↑](#glossary)

##### Blueprint Export

  Exports declare the output expected from a processed Blueprint.

[↑](#glossary)

##### Installation Export

  Exports of an Installation pick up actual values and make them accessible for a user, parent or sibling Installation.

  __Background Knowledge__:
    <details>
    Parent Installations can use exports of their [sub-installations](#sub-installation) as their own export.
    They cannot be used as inputs for their deploy items.
    </details>

[↑](#glossary)

#### _ExportExecution_

  An _ExportExecution_ is the instantiation of a [template](#template) to generate [exports](#export) as part of a [Blueprint](#blueprint).
  These templates contain the instructions which data gets written into the exports and how it might be preprocessed.
  They are specified as lists of executions under the field `exportExecutions` in a blueprint descriptor.

[↑](#glossary)

#### _Import_

  `Import` has 2 ambiguous meanings, whether we are talking about Blueprints or Installations.

##### Blueprint Import

  Imports declare what data will be required to process the Blueprint. Part of the declaration is also the format, which can either be of type [Target](#target) or any valid jsonschema.

##### Installation Import

  Imports in an Installation assign actual values and make them accessible for further processing. They satisfy the requirements (imports) defined in the Blueprint.

[↑](#glossary)

#### _Installation_

  An installation is the parameterized instance of a Blueprint deployed by a user.
  Landscaper acts upon creation or update and creates or updates dependent [Execution](#execution) and [Sub-Installations](#sub-installation).

[↑](#glossary)

#### Sibling Installations

  Sibling Installations refer to Installations belonging to the same parent.

[↑](#glossary)

#### _Sub-Installation_

  Sub-installations are Installations that are automatically created by the landscaper as part of a running installation (the installation references a [Aggregated Blueprint](#aggregated-blueprint)).

  __Background Knowledge__:
  <details>
    Sub-installations define the usage of other blueprints within an [Aggregated Blueprint](#_aggregated-blueprint_).
    Sub-installations can be nested, when deployed, they are managed by their parent (sub)installation.
  </details>

[↑](#glossary)

#### _SubinstallationExecution_

A _SubinstallationExecution_ is the instantiation of a [template](#template) to generate [nested installations](#installation) as part of a [Blueprint](#blueprint).
These templates contain installation descriptions and their wiring, that is instantiated in an own [context](#context) whenever a blueprint is instantiated.
It is used in list of such execution under the field `exportExecutions` in a blueprint descriptor.

[↑](#glossary)

#### _Target_

  A Target defines the system in which Landscaper will run the installation steps. Target resources contain all relevant data to access this environment including credentials.

[↑](#glossary)

#### _Template_

  Templates are used to render other elements for various purposes based on a dedicated
  value binding provided by a [_Blueprint_](#blueprint). The instantiation context of a
  template is called [_Execution_](#execution). There are [_DeployExecutions_](#deployexecution),
  [_ExportExecutions](#exportexecution) and [_SubinstallationExecutions_](#subinstallationexecution).
  The _Landscaper_ supports two kinds of template processors to process those templates: Go templates and [Spiff](https://github.com/mandelsoft/spiff) templates.
  
[↑](#glossary)

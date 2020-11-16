# Glossary

#### _Installation_
  An installation is the parameterized instance of a Blueprint deployed by a user.
  Landscaper acts upon creation or update and creates or updates dependent [Execution](#_execution_) and [Subinstallations](#_subinstallations).
  
#### _Subinstallations_
  Subinstallations are installations that are automatically created by the landscaper as part of a running installation (the installation references a [Aggregated Blueprint](#_aggregated-blueprint_)).
  
  __Background Knowledge__:
  <details>
    Subinstallations define the usage of other blueprints within an [Aggregated Blueprint](#_aggregated-blueprint_).    
    Subinstallations can be nested, when deployed, they are managed by their parent (sub)installation.
  </details>
  
#### _Execution_

#### _Blueprint_
    Blueprints contain actual intructions and steps on how to install a software component and what is needed to perform these actions.
    
    __Background Knowledge__:
      <details>
      Blueprints consists of:
        - Configuration Data ([Imports](#_import_))
        - Installation intructions ()
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
      Imports in an Installation reference acutal values and make them accessible for further processing. They satisfy the requirements defined in the Blueprint. 
    
#### _Export_
    `Export` has 2 ambigious meanings, whether we are talking about Blueprints or Installations.
    
    ##### Blueprint
      Exports declare the output expected from a processed Blueprint (i.e. installation). 
        
    ##### Installation
      Exports of an installation pick up actual values and make them accessible for a user or parent installation.

#### _DataObject_
    
#### _Target_

#### _DeployItem_

#### _Context_

#### _DeployExecutions_
    A list of templates to generate [deploy items](#_deployitem_) as part of a [Blueprint](#_blueprint_). Mainly used to describe the installation instructions and customize it using the data provided as [imports](#_import_).

#### _ExportExecutions_
    A list of templates describing how to generate [exports](#_export_) as part of a [Blueprint](#_blueprint_).
    These templates contain the instructions which data gets written into the exports and how it might be preprocessed.  
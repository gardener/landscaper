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
    Blueprints contains actual intructions and steps on how to install a software component.
    
    __Background Knowledge__:
      <details>
        Blueprints consists of
        - Configuration Data ([Imports](#_import_))
        - Installation intructions ()
          - [DeployItems](#_deploy-items_) or
          - [Subinstallations](#_subinstallations)
        - Output (Exports)
      </details>

#### _Aggregated Blueprint_
    Aggregated Blueprints are Blueprints that bundle multiple other blueprints.
    They contain intruction how they these referenced blueprints interact with each other.
    
    Their pratical use is to install mutliple components that depend on each other.

#### _Import_
    `Import` has 2 ambigious meanings, whether we are talking about Blueprints or Installations.
    
    ##### Blueprint
    
    ##### Installation
    
#### _Export_
    `Export` has 2 ambigious meanings, whether we are talking about Blueprints or Installations.
    
    ##### Blueprint
        
    ##### Installation

#### _DataObject_
    
#### _Target_

#### _Deploy Item_

#### _Context_

  
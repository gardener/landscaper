# Landscaper Scheduling


A installation is triggered by setting a reconcile annotation `landscaper.gardener.cloud/operation=reconcile` or its spec has changed.
With that operation set the installation controller starts to validate if the installation is allowed to be executed.

The validation consists of the following requirements:
- Imports are satisfied: 
  - imports are either exported by another component
  - given in its static data or 
  - imported by its parent
- no sibling installation dependency up to the root is running

### Export Generations

The generation (maybe hash of data) of a exported value is stated in the status of an installation `.status.configGeneration`.

the config generation of a Installation is the hash of the spec and the import's state:
```
struct GenerationHash {
    // Kubernetes generation of the respective Installation resource (`.metadata.generation`).
    // Used to detect any changes in the installation's spec.
    Generation int64
    
    // Imports are all states of imports defined in the the Installations DefintionsRef.
    // The array must be ordered by its key.
    Imports []ImportState
}

type ImportState {
    // Key is the import key of the ComponentDefinition 
    Key string
    
    // Generation is the config generation of the installation where the import's coming from.
    // The hash of the static data is used if the import is coming from static data.
    Generation string
}

hash( gob.NewEncoder().Encode(GenerationHash) )
```

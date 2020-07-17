# Landscaper Scheduling


A installation is triggered by setting a reconcile annotation `landscaper.gardener.cloud/operation=reconcile` or its spec has changed.
With that operation set the installation controller starts to validate if the installation is allowed to be executed.

The validation consists of the following requirements:
- Imports are satisfied: imports are either exported by another component, given in its static data or imported by its parent
- no sibling installation dependency up to the root is running

### Export Generations

The generation (maybe hash of data) of a exported value is stated in the status of an installation `.status.configGeneration`.

*Ideas*:
- the generation is the hash of the spec and the import's state

# Reconciliation Flow

This document aims to describe what exactly happens during the reconciliation of a Landscaper resource.

## Installations

### Core Reconciliation Logic

The core reconciliation logic handles some general checks to determine the type of reconciliation which needs to be performed (reconciliation, force-reconciliation, deletion, ...).

1. Check for ignore annotation

    If the installation is in a final phase and has the ignore annotation, the reconciliation is aborted.

2. Add finalizer

    If the installation has neither a deletion timestamp nor a landscaper finalizer, a landscaper finalizer is added and the reconciliation is aborted. 
  Adding the finalizer will cause a new reconciliation.

3. Check for deletion

    If the installation has a deletion timestamp, the `delete` function is called.

4. Check for operation annotation

    If the installation has an operation annotation, it is evaluated now: the reconcile operation annotation will cause a 'standard' reconcile, and the force-reconcile annotation will cause a 'force' reconcile. The abort annotation is meant to abort the installation, but this is currently not implemented.

5. Check for changes in nested installations

    If the installation is in a final phase and has not been modified (`.status.observedGeneration` == `.metadata.generation`), it is checked whether there have been changes in its nested installations. This updates the phase of the installation and, if it is now `Succeeded`, will trigger other installation which depend on this one by adding the reconcile operation annotation to them. The reconciliation is aborted afterwards, but if a phase change happened, this will trigger a new reconciliation.

6. Now a 'standard' reconcile will happen.

### Standard Reconciliation

This is the default reconciliation logic.

1. Compute combined phase

    The phases of all nested installations and executions are aggregated to determine the current state. The combined phase will only be `Succeeded` if all nested phases are `Succeeded`.

2. Evaluate combined phase 

    If the combined phase is empty, there are neither nested installations nor executions and the phase is set to `Succeeded` by default.
    
    If the combined phase is not a final phase, the installation's phase is set to `Progressing` and the reconciliation is aborted. The installation needs to wait for its nested resources to be completed first.

3. Check for abort operation annotation

    If the installation has the abort operation annotation, its phase is set to `Aborted` and the reconciliation is aborted.

4. Check if an update is required

    If neither the installation itself nor any of its imports has been modified, there is no need to trigger the nested installations and executions. This is checked by comparing generation values with stored 'observed' generation values, the latter of which are updated when a new generation is observed. For imports which are exported by another installation, the generation is read from the exporting installation's status. For imports which are not owned by an installation, it is usually a hash over their respective spec, although the implementations slightly differ, depending on the type of import.
 
   If an update is required, the installation's phase is set to `PendingDependencies` and it is checked whether the installation can be updated now.
    
    1. Check if updating is possible

        There are several requirements which need to be fulfilled before the installation can update its nested installations and executions:
          - all installations which is depended on need to be
            - in phase `Succeeded`
            - up-to-date (`.status.observedGeneration` == `.metadata.generation`)
            - not queued for reconciliation by the reconcile or force-reconcile operation annotation
          - all imports need to be satisfied, which means they have to
            - exist
            - be imported by the parent or be exported by a sibling, if they are owned by an installation
        
        If all of these are fulfilled, the nested installations and executions are updated. Otherwise, the installation will be stuck in `PendingDependencies` and log a corresponding error message.

5. Export generation

    The exports are generated.

6. Trigger depending installations

    All sibling installations which depend on this one are triggered by having the reconcile operation annotation added.


### Force Reconciliation

A force reconciliation only happens if the corresponding operation annotation has been added to the installation. Since it is clear in this case that an update is desired, most of the checks of the standard reconciliation are skipped in this flow. Most noticable, it is possible to update an installation despite installations the current one depends on being `Failed` or `Progressing`.

1. Check if imports are satisfied

    As for the standard reconciliation, imports have to
      - exist
      - be imported by the parent or be exported by a sibling, if they are owned by an installation

    If not fulfilled, the installation will be stuck in `PendingDependencies` and log a corresponding error message.

2. Update nested installations and executions

3. Remove operation annotation

4. Set phase

    The phase of the installation is set to `Progressing` and its observed generation is updated to match its generation.


### Deletion

This reconciliation flow is executed if the installation has a deletion timestamp.

1. Check for force-reconcile operation annotation

    If the installation has a force-reconcile operation annotation, most of the checks for the normal deletion flow are skipped:
    
    1. Update nested installations and executions

        This is to make sure that all nested installations and executions are up-to-date. It will fail with an error if updating is currently not possible.

    2. Delete nested installations and executions

        This step deletes all nested resources belonging to the installation. It also propagates the force-reconcile operation annotation to them.

        If all nested resources are gone, the finalizer is removed from the installation.
    
    3. Remove operation annotation

2. Check for sibling imports

    If there is any sibling which imports something that is exported by this installation, the deletion will abort with an error.

3. Compute combined phase

    The phases of all nested installations and executions are aggregated to determine the current state.

4. Shortcut deletion for empty installations

    If the combined phase is empty, this means there are neither nested installations nor executions. In this case, the finalizer is removed from the installation and the deletion is finished.

5. Evaluate combined phase

    If the combined phase is not a final phase, the deletion needs to wait for the nested resources to finish. The phase is set to `Deleting` and the reconciliation is aborted.

6. Check if an update is required

    It is checked whether the installation or one of its imports has been updated since the last reconciliation. This is done in the same way as explained in step 4 of the standard reconciliation flow.

    If an update is required, it is checked whether all imports are satisfied. This works as described in step 1 of the force reconciliation flow. If that is the case, the nested resources are updated, otherwise the reconciliation is aborted.

7. Delete nested installations and executions

    See step 1.2 for a description (except for the propagation of the force-reconcile operation annotation).

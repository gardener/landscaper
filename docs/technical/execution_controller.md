# Execution Controller

An execution contains in its spec the templated deploy items. The task of the execution controller is to create the
corresponding deploy item resources and to update them if necessary, so that the execution spec and the deploy items 
resources remain in sync.  

Moreover, the status of the managed deploy item resources must be collected, and if all are succeded also their exports.


## Definitions

##### Phases

The status of executions and deploy items contains a phase. The possible values are divided into two classes:
- **completed phases**: Succeeded, Failed,
- **not completed phases**: Init, Progressing, Deleting.

##### Generations: status-up-to-date

Resources like executions or deploy items contain a `generation` in their metadata and an `observed generation` in
their status. If generation and observed generation of a resource are equal, we say that its **status is up-to-date**.

If the phase of an execution is succeeded, this might refers to an old generation of the execution spec. To be sure that 
a succeeded phase refers to the current spec generation, one must check that the status is up-to date by comparing the 
generation and observed generation.


##### Generations: spec-up-to-date

The status of an execution contains information about the managed deploy items. Among others, the execution status 
contains for every managed deploy item:

- the execution generation at the time of the last deploy item update,
- the deploy item generation after the last deploy item update.

At a later time, the controller can compare these past generations with the corresponding present generations of 
execution and deploy item. If both generations are unchanged, we say that the deploy item's **spec is up-to-date**.


## Reconcile Algorithm

### Initial Tasks

In the begining of a reconcile run, the execution controller performs the following tasks:

- If the execution has the ignore annotation `landscaper.gardener.cloud/ignore: true` and has a completed phase,
  the reconcile is skipped.

- A finalizer is added to the execution, except if it already has a finalizer, or if it is being deleted, i.e. if it
  has a deletion timestamp.


### Main Operations

During a reconcile run, the execution controller does the following.

If the execution has a deletion timestamp, a [**delete operation**](#delete-operation) is performed.

Otherwise, a [**reconcile operation**](#reconcile-operation) is performed provided that
- the execution status is not up-to-date,
- or its phase is not completed,
- or the reconcile or force-reconcile annotation is set.

Otherwise, i.e. if

- the execution status is up-to-date,
- and the phase is completed,
- and there is no explicit trigger by a reconcile or force reconcile annotation,

then a [**re-check of deploy items**](#re-check-of-deploy-items) is done. This check might lead to a reconcile operation.


### Reconcile Operation

The controller reads all deploy item resources that are managed by the execution.

Deploy item resources which have no counterpart in the execution spec are **orphaned**. They will be cleaned up by the
controller.

We consider now the pairs consisting of

- a deploy item as specified in the execution spec,
- the corresponding deploy item resource, as far as it exists.

We divide these pairs in the following classes:

- First, the class of all pairs whose deploy item resource exists and whose spec is up-to-date. These deploy items need
  not be updated by the execution controller. We devide them in the following sub-classes:

    - (1A) status is up-to-date and phase = Succeeded
    - (1B) status is up-to-date and phase = Failed
    - (1C) status is not up-to-date or phase is not completed (in work)

- Second, the class of all pairs whose deploy item resource does not exist or whose spec is not up-to-date.
  These deploy items need to be updated by the execution controller. We devide them in the following sub-classes:

    - (2A) no pending dependencies
    - (2B) pending dependencies

In case of a force-reconcile, we consider all pairs as belonging to the second class, i.e. we treat them as if the
deploy item would need an update.

Based on this classification, the reconcile logic is as follows:

- If all deploy items are in class (1A) (no update required and succeeded),
  then the controller collects the export data, sets the execution phase = Succeeded, and returns.

- If there exists a deploy item in class (1B) (no update required and failed),
  then the controller sets the execution phase = Failed and returns.

- Otherwise, perform an update for all deploy items in class (2A) (update required and no pending dependencies).
  Update also the [generations in the execution status](#generations-spec-up-to-date). 
  Set the execution phase = Progressing and return.


### Re-Check of Deploy items

The controller performs a re-check of the deploy items, if a normal reconcile seems unnecessary.

The controller checks whether one of the deploy items is missing, or whether its spec was changed by someone else
after the last update by the controller. Such a change can be detected by comparing the present generation of the 
deploy item resource with the past generation of the deploy item that is stored in the execution status.

If such a change was detected, the execution controller start the normal [reconcile operation](#reconcile-operation).

Otherwise, it computes the "combined" phase of all the deploy items.
If this differs from the current execution phase, it updates the execution phase.

Moreover, if the combined phase is Succeeded and differs from the current execution phase, the exports are collected 
and updated.


### Delete Operation

Set the execution phase = Deleting.

Read the deploy items that are managed by the execution. If all of them are gone, the controller removes the finalizer
of the execution. 

Otherwise, the controller deletes all remaining deploy items, except those that already have a deletion timestamp
and those upon which another deploy item depends that still exists. 

The controller checks the phase of the deploy items that already have a deletion timestamp. If one of them has phase 
Failed, it means that the deletion has failed. Then the execution phase is also set to Failed.

At the end of the reconcile run, the deploy items are not necessarily all gone. But a new reconcile run will be 
triggered when one of the deploy items disappears. Then the delete operation will be repeated.

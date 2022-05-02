package execution

import (
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
)

// generations is a helper struct to store generation and observedGeneration values for an execution and one of its deployitems
type generations struct {
	// ExecutionGenerationInExecution is metadata.generation of the execution.
	ExecutionGenerationInExecution int64
	// ExecutionGenerationInDeployItem is the generation which the execution had when it last applied the deployitem. It is stored in the execution status.
	ExecutionGenerationInDeployItem int64
	// is the generation which the deployitem had when the execution last updated it. It is stored in the execution status.
	DeployItemGenerationInExecution int64
	// DeployItemGenerationInDeployItem is metadata.generation of the deployitem.
	DeployItemGenerationInDeployItem int64
}

// IsUpToDate returns whether the deployitem is up-to-date.
// It will return true if both hasExecutionBeenModified() and hasDeployItemBeenModified() return false, and false otherwise.
func (g generations) IsUpToDate() bool {
	return !(g.HasExecutionBeenModified() || g.HasDeployItemBeenModified())
}

// HasExecutionBeenModified returns true if the execution has been modified since the deployitem has last been updated, and false otherwise.
func (g generations) HasExecutionBeenModified() bool {
	return g.ExecutionGenerationInExecution != g.ExecutionGenerationInDeployItem
}

// HasDeployItemBeenModified returns true if the deployitem has been modified since the execution last updated it, and false otherwise.
func (g generations) HasDeployItemBeenModified() bool {
	return g.DeployItemGenerationInDeployItem != g.DeployItemGenerationInExecution
}

// getGenerations returns a generations struct containing the generations and observedGenerations for the execution and the given deployitem
func newGenerations(exec *lsv1alpha1.Execution, itemName string, deployItem *lsv1alpha1.DeployItem) generations {
	var lastSeenGeneration int64
	if ref, ok := lsv1alpha1helper.GetVersionedNamedObjectReference(exec.Status.DeployItemReferences, itemName); ok {
		lastSeenGeneration = ref.Reference.ObservedGeneration
	}
	var lastAppliedGeneration int64
	if expGen, ok := getExecutionGeneration(exec.Status.ExecutionGenerations, itemName); ok {
		lastAppliedGeneration = expGen.ObservedGeneration
	}
	return generations{
		ExecutionGenerationInExecution:   exec.GetGeneration(),
		ExecutionGenerationInDeployItem:  lastAppliedGeneration,
		DeployItemGenerationInExecution:  lastSeenGeneration,
		DeployItemGenerationInDeployItem: deployItem.GetGeneration(),
	}
}

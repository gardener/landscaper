package execution

import (
	"fmt"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lserrors "github.com/gardener/landscaper/apis/errors"
)

// DeployItemClassification divides all the deploy items of an execution into the following classes.
// Every item belongs to exactly one class.
// - running items:   they have the same jobID as the execution, but are unfinished
// - succeeded items: they have the same jobID as the execution, are finished and succeeded
// - failed items:    they have the same jobID as the execution, are finished and not succeeded (=> failed)
// - runnableItems:   they have an old jobID, which can be updated because there are no pending dependencies
// - pending items:   they have an old jobID, which can not be updated because of pending dependencies
type DeployItemClassification struct {
	runningItems   []*executionItem
	succeededItems []*executionItem
	failedItems    []*executionItem
	runnableItems  []*executionItem
	pendingItems   []*executionItem
}

func (c *DeployItemClassification) HasRunningItems() bool {
	return len(c.runningItems) > 0
}

func (c *DeployItemClassification) HasSucceededItems() bool {
	return len(c.succeededItems) > 0
}

func (c *DeployItemClassification) HasFailedItems() bool {
	return len(c.failedItems) > 0
}

func (c *DeployItemClassification) HasRunnableItems() bool {
	return len(c.runnableItems) > 0
}

func (c *DeployItemClassification) HasPendingItems() bool {
	return len(c.pendingItems) > 0
}

func (c *DeployItemClassification) AllSucceeded() bool {
	return !c.HasRunningItems() && !c.HasFailedItems() && !c.HasRunnableItems() && !c.HasPendingItems()
}

func (c *DeployItemClassification) GetRunnableItems() []*executionItem {
	return c.runnableItems
}

func newDeployItemClassification(executionJobID string, items []*executionItem) (*DeployItemClassification, lserrors.LsError) {
	c := &DeployItemClassification{
		runningItems:   []*executionItem{},
		succeededItems: []*executionItem{},
		failedItems:    []*executionItem{},
		runnableItems:  []*executionItem{},
		pendingItems:   []*executionItem{},
	}

	for i := range items {
		item := items[i]

		if item.DeployItem == nil {
			// The items that we are classifying here were all created in the previous phase and should exist.
			// But a user could have deleted items with "kubectl delete" or "landscaper-cli installations force-delete".
			// We treat missing items as failed.
			c.failedItems = append(c.failedItems, item)
		} else if item.DeployItem.Status.GetJobID() == executionJobID {
			if item.DeployItem.Status.GetJobID() != item.DeployItem.Status.JobIDFinished {
				c.runningItems = append(c.runningItems, item)
			} else if item.DeployItem.Status.Phase == lsv1alpha1.DeployItemPhases.Succeeded {
				c.succeededItems = append(c.succeededItems, item)
			} else {
				c.failedItems = append(c.failedItems, item)
			}
		} else {
			runnable, lsErr := isItemRunnable(executionJobID, item, items)
			if lsErr != nil {
				return nil, lsErr
			}

			if runnable {
				c.runnableItems = append(c.runnableItems, item)
			} else {
				c.pendingItems = append(c.pendingItems, item)
			}
		}
	}

	return c, nil
}

func isItemRunnable(executionJobID string, item *executionItem, items []*executionItem) (bool, lserrors.LsError) {
	if len(item.Info.DependsOn) == 0 {
		return true, nil
	}

	for _, dependentItemName := range item.Info.DependsOn {
		dependentItem := getItemByName(dependentItemName, items)
		if dependentItem == nil {
			return false, lserrors.NewError("IsRunnable", "DependentDeployItemNotFound",
				fmt.Sprintf("dependent deployitem %s of deployitem %s not found", dependentItemName, item.Info.Name))
		}

		// check that the dependentItem has finished the current job
		if dependentItem.DeployItem == nil || dependentItem.DeployItem.Status.JobIDFinished != executionJobID {
			return false, nil
		}
	}

	return true, nil
}

func getItemByName(name string, items []*executionItem) *executionItem {
	for _, item := range items {
		if item.Info.Name == name {
			return item
		}
	}
	return nil
}

func newDeployItemClassificationForDelete(executionJobID string, items []*executionItem) (*DeployItemClassification, lserrors.LsError) {
	c := &DeployItemClassification{
		runningItems:   []*executionItem{},
		succeededItems: []*executionItem{},
		failedItems:    []*executionItem{},
		runnableItems:  []*executionItem{},
		pendingItems:   []*executionItem{},
	}

	for i := range items {
		item := items[i]

		if item.DeployItem == nil {
			c.succeededItems = append(c.succeededItems, item)
		} else if item.DeployItem.Status.GetJobID() == executionJobID {
			if item.DeployItem.Status.GetJobID() != item.DeployItem.Status.JobIDFinished {
				c.runningItems = append(c.runningItems, item)
			} else if item.DeployItem.Status.GetJobID() == item.DeployItem.Status.JobIDFinished &&
				!item.DeployItem.Status.Phase.IsFailed() {
				c.runningItems = append(c.runningItems, item)
			} else {
				c.failedItems = append(c.failedItems, item)
			}
		} else {
			if isItemDeletable(item, items) {
				c.runnableItems = append(c.runnableItems, item)
			} else {
				c.pendingItems = append(c.pendingItems, item)
			}
		}
	}

	return c, nil
}

func newDeployItemClassificationForOrphans(executionJobID string, deployitems []lsv1alpha1.DeployItem) (*DeployItemClassification, lserrors.LsError) {
	items := make([]*executionItem, len(deployitems))

	for i := range deployitems {
		items[i] = &executionItem{
			Info:       lsv1alpha1.DeployItemTemplate{},
			DeployItem: &deployitems[i],
		}
	}

	return newDeployItemClassificationForDelete(executionJobID, items)
}

func isItemDeletable(item *executionItem, items []*executionItem) bool {
	// Check whether the item appears in the DependsOn list of a sibling item that is not yet deleted
	for _, siblingItem := range items {
		if siblingItem.DeployItem != nil {
			for _, dependentItemName := range siblingItem.Info.DependsOn {
				if dependentItemName == item.Info.Name {
					return false
				}
			}
		}
	}

	return true
}

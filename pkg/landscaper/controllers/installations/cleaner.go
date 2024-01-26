package installations

import (
	"context"

	"github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/landscaper/controller-utils/pkg/kubernetes"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
)

type DataObjectAndTargetCleaner struct {
	installation *lsv1alpha1.Installation
	client       client.Client
}

func NewDataObjectAndTargetCleaner(installation *lsv1alpha1.Installation, cl client.Client) *DataObjectAndTargetCleaner {
	return &DataObjectAndTargetCleaner{
		installation: installation,
		client:       cl,
	}
}

// CleanupExports deletes all DataObjects and Targets exported by the given Installation.
// These are the DataObjects and Targets that 1. belong to the namespace of the Installation and 2. have a source
// label (data.landscaper.gardener.cloud/source) indicating that they have been exported by the Installation.
func (c *DataObjectAndTargetCleaner) CleanupExports(ctx context.Context) error {
	doList := &lsv1alpha1.DataObjectList{}
	if err := read_write_layer.ListDataObjects(ctx, c.client, doList, read_write_layer.R000060,
		client.InNamespace(c.installation.Namespace),
		client.MatchingLabels{
			lsv1alpha1.DataObjectSourceLabel:     lsv1alpha1helper.DataObjectSourceFromInstallation(c.installation),
			lsv1alpha1.DataObjectSourceTypeLabel: string(lsv1alpha1.ExportDataObjectSourceType),
		}); err != nil {
		return err
	}

	if err := c.deleteDataObjects(ctx, doList.Items, false); err != nil {
		return err
	}

	targetList := &lsv1alpha1.TargetList{}
	if err := read_write_layer.ListTargets(ctx, c.client, targetList, read_write_layer.R000061,
		client.InNamespace(c.installation.Namespace),
		client.MatchingLabels{
			lsv1alpha1.DataObjectSourceLabel:     lsv1alpha1helper.DataObjectSourceFromInstallation(c.installation),
			lsv1alpha1.DataObjectSourceTypeLabel: string(lsv1alpha1.ExportDataObjectSourceType),
		}); err != nil {
		return err
	}

	if err := c.deleteTargets(ctx, targetList.Items, false); err != nil {
		return err
	}

	return nil
}

// CleanupContext deletes all DataObjects and Targets in the context of the given Installation.
func (c *DataObjectAndTargetCleaner) CleanupContext(ctx context.Context) error {
	doList := &lsv1alpha1.DataObjectList{}
	if err := read_write_layer.ListDataObjects(ctx, c.client, doList, read_write_layer.R000062,
		client.InNamespace(c.installation.Namespace),
		client.MatchingLabels{
			lsv1alpha1.DataObjectContextLabel: lsv1alpha1helper.DataObjectSourceFromInstallation(c.installation),
		}); err != nil {
		return err
	}

	targetList := &lsv1alpha1.TargetList{}
	if err := read_write_layer.ListTargets(ctx, c.client, targetList, read_write_layer.R000063,
		client.InNamespace(c.installation.Namespace),
		client.MatchingLabels{
			lsv1alpha1.DataObjectContextLabel: lsv1alpha1helper.DataObjectSourceFromInstallation(c.installation),
		}); err != nil {
		return err
	}

	isNewDeletion := utils.CheckIfNewContextDeletion(doList, targetList)

	if isNewDeletion {
		if err := c.deleteDataObjects(ctx, doList.Items, true); err != nil {
			return err
		}

		if err := c.deleteTargets(ctx, targetList.Items, true); err != nil {
			return err
		}
	}
	return nil
}

func (c *DataObjectAndTargetCleaner) deleteDataObjects(ctx context.Context, dataObjects []lsv1alpha1.DataObject,
	checkJobID bool) error {
	for i := range dataObjects {
		do := &dataObjects[i]
		if !checkJobID || !kubernetes.HasLabelWithValue(&do.ObjectMeta, lsv1alpha1.DataObjectJobIDLabel, c.installation.Status.JobID) {
			if err := read_write_layer.NewWriter(c.client).DeleteDataObject(ctx, read_write_layer.W000014, do); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *DataObjectAndTargetCleaner) deleteTargets(ctx context.Context, targets []lsv1alpha1.Target,
	checkJobID bool) error {
	for i := range targets {
		target := &targets[i]
		if !checkJobID || !kubernetes.HasLabelWithValue(&target.ObjectMeta, lsv1alpha1.DataObjectJobIDLabel, c.installation.Status.JobID) {
			if err := read_write_layer.NewWriter(c.client).DeleteTarget(ctx, read_write_layer.W000011, target); err != nil {
				return err
			}
		}
	}

	return nil
}

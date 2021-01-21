// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// CheckCompletedSiblingDependentsOfParent checks if siblings and siblings of the parent's parents that the parent depends on (imports data) are completed.
func CheckCompletedSiblingDependentsOfParent(ctx context.Context, op *installations.Operation, parent *installations.Installation) (bool, error) {
	if parent == nil {
		return true, nil
	}
	parentsOperation, err := installations.NewInstallationOperationFromOperation(ctx, op, parent)
	if err != nil {
		return false, fmt.Errorf("unable to create parent operation: %w", err)
	}
	siblingsCompleted, err := CheckCompletedSiblingDependents(ctx, parentsOperation, parent)
	if err != nil {
		return false, err
	}
	if !siblingsCompleted {
		return siblingsCompleted, nil
	}

	// check its own parent
	parentsParent, err := installations.GetParent(ctx, op, parent)
	if err != nil {
		return false, errors.Wrap(err, "unable to get parent of parent")
	}

	if parentsParent == nil {
		return true, nil
	}
	return CheckCompletedSiblingDependentsOfParent(ctx, parentsOperation, parentsParent)
}

// CheckCompletedSiblingDependents checks if siblings that the installation depends on (imports data) are completed
func CheckCompletedSiblingDependents(ctx context.Context, op *installations.Operation, inst *installations.Installation) (bool, error) {
	if inst == nil {
		return true, nil
	}
	// todo: add target support
	for _, dataImport := range inst.Info.Spec.Imports.Data {
		sourceRef, err := getImportSource(ctx, op, inst, dataImport)
		if err != nil {
			return false, err
		}
		if sourceRef == nil {
			continue
		}
		// check if the import is imported from myself or the parent
		// and continue if so as we have a different check for the parent
		if lsv1alpha1helper.ReferenceIsObject(*sourceRef, inst.Info) {
			continue
		}

		parent, err := installations.GetParent(ctx, op, inst)
		if err != nil {
			return false, err
		}
		if parent != nil && lsv1alpha1helper.ReferenceIsObject(*sourceRef, parent.Info) {
			continue
		}

		// we expect that the source ref is always a installation
		inst := &lsv1alpha1.Installation{}
		if err := op.Client().Get(ctx, sourceRef.NamespacedName(), inst); err != nil {
			return false, err
		}

		if !lsv1alpha1helper.IsCompletedInstallationPhase(inst.Status.Phase) {
			op.Log().V(3).Info("dependent installation not completed", "inst", sourceRef.NamespacedName().String())
			return false, nil
		}

		intInst, err := installations.CreateInternalInstallation(ctx, op, inst)
		if err != nil {
			return false, err
		}

		isCompleted, err := CheckCompletedSiblingDependents(ctx, op, intInst)
		if err != nil {
			return false, err
		}
		if !isCompleted {
			return false, nil
		}
	}

	return true, nil
}

// getImportSource returns a reference to the owner of a data import.
func getImportSource(ctx context.Context, op *installations.Operation, inst *installations.Installation, dataImport lsv1alpha1.DataImport) (*lsv1alpha1.ObjectReference, error) {
	status, err := inst.ImportStatus().GetData(dataImport.Name)
	if err == nil && status.SourceRef != nil {
		return status.SourceRef, nil
	}

	// we have to get the corresponding installation from the the cluster
	do := &lsv1alpha1.DataObject{}
	doName := lsv1alpha1helper.GenerateDataObjectName(op.Context().Name, dataImport.DataRef)
	if err := op.Client().Get(ctx, kutil.ObjectKey(doName, inst.Info.Namespace), do); err != nil {
		return nil, fmt.Errorf("unable to fetch data object %s (%s): %w", doName, dataImport.DataRef, err)
	}
	owner := kutil.GetOwner(do.ObjectMeta)
	if owner == nil {
		return nil, nil
	}

	// we cannot validate if the source is not an installation
	if owner.Kind != "Installation" {
		return nil, nil
	}
	return &lsv1alpha1.ObjectReference{Name: owner.Name, Namespace: inst.Info.Namespace}, nil
}

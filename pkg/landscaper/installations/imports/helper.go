// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
)

// CheckCompletedSiblingDependentsOfParent checks if siblings and siblings of the parent's parents that the parent depends on (imports data) are completed.
func CheckCompletedSiblingDependentsOfParent(ctx context.Context, kubeClient client.Client, parent *installations.InstallationBase) (bool, error) {
	if parent == nil {
		return true, nil
	}
	parentCtxName := installations.GetInstallationContextName(parent.Info)
	siblingsCompleted, err := CheckCompletedSiblingDependents(ctx, kubeClient, parentCtxName, parent)
	if err != nil {
		return false, err
	}
	if !siblingsCompleted {
		return siblingsCompleted, nil
	}

	// check its own parent
	parentsParentInst, err := installations.GetParent(ctx, kubeClient, parent.Info)
	if err != nil {
		return false, fmt.Errorf("unable to get parent of parent: %w", err)
	}

	if parentsParentInst == nil {
		// if the parents parent is nil means that the parent itself is a root installation.
		return true, nil
	}
	return CheckCompletedSiblingDependentsOfParent(ctx, kubeClient, installations.NewInstallationBase(parentsParentInst))
}

// CheckCompletedSiblingDependents checks if siblings that the installation depends on (imports data) are completed
func CheckCompletedSiblingDependents(ctx context.Context,
	kubeClient client.Client,
	contextName string,
	inst *installations.InstallationBase) (bool, error) {
	if inst == nil {
		return true, nil
	}

	log := logr.FromContextOrDiscard(ctx)

	checkSource := func(sourceRef *lsv1alpha1.ObjectReference) (isCompleted bool, ignore bool, err error) {
		if sourceRef == nil {
			// only possible if the owner is not an installation
			return false, true, nil
		}
		// check if the import is imported from myself or the parent
		// and continue if so as we have a different check for the parent
		if lsv1alpha1helper.ReferenceIsObject(*sourceRef, inst.Info) {
			return false, true, nil
		}

		parent, err := installations.GetParent(ctx, kubeClient, inst.Info)
		if err != nil {
			return false, false, err
		}
		if parent != nil && lsv1alpha1helper.ReferenceIsObject(*sourceRef, parent) {
			return true, false, nil
		}

		// we expect that the source ref is always an installation
		inst := &lsv1alpha1.Installation{}
		if err := kubeClient.Get(ctx, sourceRef.NamespacedName(), inst); err != nil {
			return false, false, err
		}

		if !lsv1alpha1helper.IsCompletedInstallationPhase(inst.Status.Phase) {
			log.V(3).Info("dependent installation not completed", "inst", sourceRef.NamespacedName().String())
			return false, false, nil
		}

		if inst.Generation != inst.Status.ObservedGeneration {
			log.V(3).Info("dependent installation completed but not up-to-date", "inst", sourceRef.NamespacedName().String())
			return false, false, nil
		}

		if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ReconcileOperation) || lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
			log.V(3).Info("dependent installation completed but has (force-)reconcile annotation", "inst", sourceRef.NamespacedName().String())
			return false, false, nil
		}

		intInst := installations.CreateInternalInstallationBase(inst)

		isCompleted, err = CheckCompletedSiblingDependents(ctx, kubeClient, contextName, intInst)
		if err != nil {
			return false, false, err
		}
		return
	}

	for _, dataImport := range inst.Info.Spec.Imports.Data {
		sourceRef, err := getImportSource(ctx, kubeClient, contextName, inst, dataImport)
		if err != nil {
			return false, err
		}
		isCompleted, ignore, err := checkSource(sourceRef)
		if err != nil {
			return false, err
		}
		if ignore {
			continue
		}
		if !isCompleted {
			return false, nil
		}
	}

	for _, targetImport := range inst.Info.Spec.Imports.Targets {
		sourceRefs, err := getTargetSources(ctx, kubeClient, contextName, inst.Info, targetImport)
		if err != nil {
			return false, err
		}
		for _, sourceRef := range sourceRefs {
			if sourceRef == nil {
				// only possible if the owner is not an installation
				continue
			}
			isCompleted, ignore, err := checkSource(sourceRef)
			if err != nil {
				return false, err
			}
			if ignore {
				continue
			}
			if !isCompleted {
				return false, nil
			}
		}
	}

	return true, nil
}

// getImportSource returns a reference to the owner of a data import.
func getImportSource(ctx context.Context,
	kubeClient client.Client,
	contextName string,
	inst *installations.InstallationBase,
	dataImport lsv1alpha1.DataImport) (*lsv1alpha1.ObjectReference, error) {

	// we have to get the corresponding installation from the cluster
	_, owner, err := installations.GetDataImport(ctx, kubeClient, contextName, inst, dataImport)
	if err != nil {
		return nil, err
	}

	// we cannot validate if the source is not an installation
	if owner == nil || owner.Kind != "Installation" {
		return nil, nil
	}
	return &lsv1alpha1.ObjectReference{Name: owner.Name, Namespace: inst.Info.Namespace}, nil
}

// getTargetSources returns a reference to the owner of all target imports.
func getTargetSources(ctx context.Context,
	kubeClient client.Client,
	contextName string,
	inst *lsv1alpha1.Installation,
	targetImport lsv1alpha1.TargetImport) ([]*lsv1alpha1.ObjectReference, error) {

	// we have to get the corresponding installation from the cluster
	targets, _, err := installations.GetTargets(ctx, kubeClient, contextName, inst, targetImport)
	if err != nil {
		return nil, err
	}

	refs := make([]*lsv1alpha1.ObjectReference, 0)
	for _, target := range targets {
		owner := kutil.GetOwner(target.Raw.ObjectMeta)
		if owner == nil || owner.Kind != "Installation" {
			continue
		}
		refs = append(refs, &lsv1alpha1.ObjectReference{Name: owner.Name, Namespace: inst.Namespace})
	}
	return refs, nil
}

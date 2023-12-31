// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

type InstallationTrigger struct {
	client client.Client
	inst   *lsv1alpha1.Installation
}

func NewInstallationTrigger(cl client.Client, inst *lsv1alpha1.Installation) *InstallationTrigger {
	return &InstallationTrigger{
		client: cl,
		inst:   inst,
	}
}

func (t *InstallationTrigger) DetermineDependents(ctx context.Context) ([]lsv1alpha1.DependentToTrigger, error) {
	var dependents []lsv1alpha1.DependentToTrigger

	if t.inst.Spec.Optimization != nil && t.inst.Spec.Optimization.HasNoSiblingExports {
		return dependents, nil
	}

	_, siblings, err := GetParentAndSiblings(ctx, t.client, t.inst)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sibling installations: %w", err)
	}

	for _, sibling := range siblings {
		if !t.inst.IsSuccessor(sibling) {
			continue
		}

		dependents = append(dependents, lsv1alpha1.DependentToTrigger{
			Name: sibling.GetName(),
		})
	}

	return dependents, nil
}

// TriggerDependents triggers the dependent installations, that had been added to the status when the current
// installation finished. Afterwards, the dependents are removed from the status.
func (t *InstallationTrigger) TriggerDependents(ctx context.Context) error {
	if len(t.inst.Status.DependentsToTrigger) == 0 {
		return nil
	}

	for _, dependent := range t.inst.Status.DependentsToTrigger {
		if err := t.triggerDependent(ctx, dependent); err != nil {
			return err
		}
	}

	if err := t.clearDependents(ctx); err != nil {
		return err
	}

	return nil
}

func (t *InstallationTrigger) triggerDependent(ctx context.Context, dependent lsv1alpha1.DependentToTrigger) error {
	dependentInst, err := t.getDependent(ctx, dependent)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get dependent installation: %w", err)
	}

	if IsRootInstallation(dependentInst) {
		metav1.SetMetaDataAnnotation(&dependentInst.ObjectMeta, lsv1alpha1.OperationAnnotation, string(lsv1alpha1.ReconcileOperation))
	}

	lsv1alpha1helper.Touch(&dependentInst.ObjectMeta)

	if err := read_write_layer.NewWriter(t.client).UpdateInstallation(ctx, read_write_layer.W000040, dependentInst); err != nil {
		return fmt.Errorf("failed to update dependent installation: %w", err)
	}

	return nil
}

func (t *InstallationTrigger) getDependent(ctx context.Context, dependent lsv1alpha1.DependentToTrigger) (*lsv1alpha1.Installation, error) {
	dependentInst := &lsv1alpha1.Installation{}
	dependentKey := client.ObjectKey{
		Namespace: t.inst.GetNamespace(),
		Name:      dependent.Name,
	}
	if err := read_write_layer.GetInstallation(ctx, t.client, dependentKey, dependentInst, read_write_layer.R000014); err != nil {
		return nil, err
	}
	return dependentInst, nil
}

func (t *InstallationTrigger) clearDependents(ctx context.Context) error {
	t.inst.Status.DependentsToTrigger = nil
	if err := read_write_layer.NewWriter(t.client).UpdateInstallationStatus(ctx, read_write_layer.W000042, t.inst); err != nil {
		return fmt.Errorf("failed to clear depends: %w", err)
	}

	return nil
}

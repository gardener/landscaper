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

	_, siblings, err := GetParentAndSiblings(ctx, t.client, t.inst)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sibling installations: %w", err)
	}

	for _, sibling := range siblings {
		if !t.importsAnyExport(t.inst, sibling) {
			continue
		}

		dependents = append(dependents, lsv1alpha1.DependentToTrigger{
			Name: sibling.GetName(),
		})
	}

	return dependents, nil
}

func (t *InstallationTrigger) importsAnyExport(exporter, importer *lsv1alpha1.Installation) bool {
	for _, export := range exporter.Spec.Exports.Data {
		for _, def := range importer.Spec.Imports.Data {
			if def.DataRef == export.DataRef {
				return true
			}
		}
	}

	for _, export := range exporter.Spec.Exports.Targets {
		for _, def := range importer.Spec.Imports.Targets {
			if def.Target == export.Target {
				return true
			}
		}
	}

	return false
}

func (t *InstallationTrigger) TriggerDependents(ctx context.Context) error {
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
	if err := read_write_layer.GetInstallation(ctx, t.client, dependentKey, dependentInst); err != nil {
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

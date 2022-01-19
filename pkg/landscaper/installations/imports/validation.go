// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports

import (
	"context"
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
)

// NewValidator creates new import validator.
// It validates if all imports of a component are satisfied given a context.
func NewValidator(ctx context.Context, op *installations.Operation) (*Validator, error) {
	var parent *installations.Installation
	if op.Context().Parent != nil {
		var err error
		parent, err = installations.CreateInternalInstallationWithContext(ctx,
			op.Context().Parent.Info,
			op.Client(),
			op.ComponentsRegistry(),
			op.Overwriter)
		if err != nil {
			return nil, err
		}
	}

	return &Validator{
		Operation: op,
		parent:    parent,
		siblings:  op.Context().Siblings,
	}, nil
}

// NewTestValidator creates a new import validator for tests.
func NewTestValidator(op *installations.Operation, parent *installations.Installation) *Validator {
	return &Validator{
		Operation: op,
		parent:    parent,
		siblings:  op.Context().Siblings,
	}
}

// getParentBase returns the base installation for the parent
func (v *Validator) getParentBase() *installations.InstallationBase {
	if v.parent == nil {
		return nil
	}
	return &v.parent.InstallationBase
}

// OutdatedImports validates whether a imported data object or target is outdated.
func (v *Validator) OutdatedImports(ctx context.Context) (bool, error) {
	const OutdatedImportsReason = "OutdatedImports"
	fldPath := field.NewPath(fmt.Sprintf("(Inst %s)", v.Inst.Info.Name))
	cond := lsv1alpha1helper.GetOrInitCondition(v.Inst.Info.Status.Conditions, lsv1alpha1.ValidateImportsCondition)

	for _, dataImport := range v.Inst.Info.Spec.Imports.Data {
		impPath := fldPath.Child(dataImport.Name)
		outdated, err := v.checkDataImportIsOutdated(ctx, impPath, v.Inst, dataImport)
		if err != nil {
			v.Inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionUnknown,
				OutdatedImportsReason,
				fmt.Sprintf("Check for outdated data imports failed: %s", err.Error())))
			return false, err
		}
		if outdated {
			v.Inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
				OutdatedImportsReason,
				"A least one data import is outdated"))
			return true, nil
		}
	}

	for _, targetImport := range v.Inst.Info.Spec.Imports.Targets {
		impPath := fldPath.Child(targetImport.Name)
		outdated, err := v.checkTargetImportIsOutdated(ctx, impPath, v.Inst, targetImport)
		if err != nil {
			v.Inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionUnknown,
				OutdatedImportsReason,
				fmt.Sprintf("Check for outdated target imports failed: %s", err.Error())))
			return false, err
		}
		if outdated {
			v.Inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
				OutdatedImportsReason,
				"A least one target import is outdated"))
			return true, nil
		}
	}
	v.Inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
		OutdatedImportsReason,
		"All imports are up-to-date"))
	return false, nil
}

// CheckDependentInstallations checks whether all dependencies are succeeded.
// It traverses through all dependent siblings and all dependent siblings of its parents.
func (v *Validator) CheckDependentInstallations(ctx context.Context) (bool, error) {
	const CheckSiblingDependentsOfParentsReason = "CheckSiblingDependentsOfParents"
	const CheckSiblingDependentsReason = "CheckSiblingDependents"
	cond := lsv1alpha1helper.GetOrInitCondition(v.Inst.Info.Status.Conditions, lsv1alpha1.ValidateImportsCondition)

	if v.IsRoot() {
		completed, err := CheckCompletedSiblingDependents(ctx,
			v.Operation.Client(),
			installations.GetInstallationContextName(v.Inst.Info),
			&v.Inst.InstallationBase)
		if err != nil {
			v.Inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				CheckSiblingDependentsReason,
				fmt.Sprintf("Check for progressing dependents failed: %s", err.Error())))
			return false, err
		}

		if !completed {
			v.Inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				CheckSiblingDependentsReason,
				"Waiting until all progressing dependents are finished"))
		}
		return completed, err
	}

	// check if parent has sibling installation dependencies that are not finished yet
	// we only need to check the parent and not the direct siblings because the
	// parent MUST be progressing if any of its children is progressing.
	completed, err := CheckCompletedSiblingDependentsOfParent(ctx, v.Operation.Client(), v.getParentBase())
	if err != nil {
		v.Inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			CheckSiblingDependentsOfParentsReason,
			fmt.Sprintf("Check for progressing dependents of the parent failed: %s", err.Error())))
		return false, err
	}

	if !completed {
		v.Inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			CheckSiblingDependentsOfParentsReason,
			"Waiting until all progressing dependents of the parent are finished"))
	}
	return completed, err
}

// ImportsSatisfied validates if all imports are satisfied with the correct version.
func (v *Validator) ImportsSatisfied(ctx context.Context, inst *installations.Installation) error {
	const ImportsSatisfiedReason = "ImportsSatisfied"
	cond := lsv1alpha1helper.GetOrInitCondition(inst.Info.Status.Conditions, lsv1alpha1.ValidateImportsCondition)
	fldPath := field.NewPath(fmt.Sprintf("(Inst %s)", inst.Info.Name))

	for _, dataImport := range inst.Info.Spec.Imports.Data {
		impPath := fldPath.Child(dataImport.Name)
		err := v.checkDataImportIsSatisfied(ctx, impPath, inst, dataImport)
		if err != nil {
			inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				ImportsSatisfiedReason,
				fmt.Sprintf("Waiting until all data imports are satisfied: %s", err.Error())))
			return err
		}
	}

	for _, targetImport := range inst.Info.Spec.Imports.Targets {
		impPath := fldPath.Child(targetImport.Name)
		err := v.checkTargetImportIsSatisfied(ctx, impPath, inst, targetImport)
		if err != nil {
			inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				ImportsSatisfiedReason,
				fmt.Sprintf("Waiting until all target imports are satisfied: %s", err.Error())))
			return err
		}
	}

	inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
		ImportsSatisfiedReason,
		"All imports are satisfied"))
	return nil
}

func (v *Validator) checkDataImportIsOutdated(ctx context.Context, fldPath *field.Path, inst *installations.Installation, dataImport lsv1alpha1.DataImport) (bool, error) {
	// get deploy item from current context
	do, owner, err := installations.GetDataImport(ctx, v.Client(), v.Context().Name, &inst.InstallationBase, dataImport)
	if err != nil {
		return false, fmt.Errorf("%s: unable to get data object for '%s': %w", fldPath.String(), dataImport.Name, err)
	}
	importStatus, err := inst.ImportStatus().GetData(dataImport.Name)
	if err != nil {
		return true, nil
	}

	// we cannot validate if the source is not an installation
	if owner == nil || owner.Kind != "Installation" {
		if do.Metadata.Hash != importStatus.ConfigGeneration {
			return true, nil
		}

		return false, nil
	}

	ref := lsv1alpha1.ObjectReference{Name: owner.Name, Namespace: inst.Info.Namespace}
	src := &lsv1alpha1.Installation{}
	if err := v.Client().Get(ctx, ref.NamespacedName(), src); err != nil {
		return false, fmt.Errorf("%s: unable to get source installation %s for '%s': %w", fldPath.String(), ref.NamespacedName().String(), dataImport.Name, err)
	}
	if src.Status.ConfigGeneration != importStatus.ConfigGeneration {
		return true, nil
	}
	return false, nil
}

func (v *Validator) checkTargetImportIsOutdated(ctx context.Context, fldPath *field.Path, inst *installations.Installation, targetImport lsv1alpha1.TargetImport) (bool, error) {
	// get deploy item from current context
	targets, _, err := installations.GetTargets(ctx, v.Client(), v.Context().Name, inst.Info, targetImport)
	if err != nil {
		return false, err
	}
	singleTarget := false
	if len(targetImport.Target) != 0 {
		singleTarget = true
	}

	for _, t := range targets {
		o := t.Owner
		configGen := ""

		importStatus, err := inst.ImportStatus().GetTarget(targetImport.Name)
		if err != nil {
			return true, nil
		}

		if singleTarget {
			configGen = importStatus.ConfigGeneration
		} else {
			found := false
			for _, elem := range importStatus.Targets {
				if t.Raw.Name == elem.Target {
					configGen = elem.ConfigGeneration
					found = true
					break
				}
			}
			if !found {
				return false, fmt.Errorf("no config generation for target '%s' found in installation status", t.Raw.Name)
			}
		}

		// we cannot validate if the source is not an installation
		if o == nil || o.Kind != "Installation" {
			if strconv.Itoa(int(t.Raw.Generation)) != configGen {
				return true, nil
			}
			continue
		}

		ref := lsv1alpha1.ObjectReference{Name: o.Name, Namespace: inst.Info.Namespace}
		src := &lsv1alpha1.Installation{}
		if err := v.Client().Get(ctx, ref.NamespacedName(), src); err != nil {
			return false, fmt.Errorf("%s: unable to get source installation %s for '%s': %w", fldPath.String(), ref.NamespacedName().String(), targetImport.Name, err)
		}
		if src.Status.ConfigGeneration != configGen {
			return true, nil
		}
	}

	return false, nil
}

func (v *Validator) checkDataImportIsSatisfied(ctx context.Context, fldPath *field.Path, inst *installations.Installation, dataImport lsv1alpha1.DataImport) error {
	// get deploy item from current context
	_, owner, err := installations.GetDataImport(ctx, v.Client(), v.Context().Name, &inst.InstallationBase, dataImport)
	if err != nil {
		return fmt.Errorf("%s: unable to get data object for '%s': %w", fldPath.String(), dataImport.Name, err)
	}
	if owner == nil {
		return nil
	}

	// we cannot validate if the source is not an installation
	if owner.Kind != "Installation" {
		return nil
	}
	ref := lsv1alpha1.ObjectReference{Name: owner.Name, Namespace: inst.Info.Namespace}

	// do not validate if I'm the owner of the resource
	if lsv1alpha1helper.ReferenceIsObject(ref, v.Inst.Info) {
		return nil
	}

	// check if the data object comes from the parent
	if v.parent != nil && lsv1alpha1helper.ReferenceIsObject(ref, v.parent.Info) {
		return v.checkStateForParentImport(fldPath, dataImport.DataRef)
	}

	// otherwise validate as sibling export
	return v.checkStateForSiblingDataExport(ctx, fldPath, ref, dataImport.DataRef)
}

func (v *Validator) checkTargetImportIsSatisfied(ctx context.Context, fldPath *field.Path, inst *installations.Installation, targetImport lsv1alpha1.TargetImport) error {
	// get deploy item from current context
	targets, targetImportReferences, err := installations.GetTargets(ctx, v.Client(), v.Context().Name, inst.Info, targetImport)
	if err != nil {
		return err
	}

	allErrs := make([]error, 0)
	for i, t := range targets {
		o := t.Owner
		var targetImportReference string
		if len(targetImportReferences) > 1 {
			targetImportReference = targetImportReferences[i]
		} else {
			targetImportReference = targetImportReferences[0]
		}
		// we cannot validate if the source is not an installation
		if o == nil || o.Kind != "Installation" {
			continue
		}
		ref := lsv1alpha1.ObjectReference{Name: o.Name, Namespace: inst.Info.Namespace}

		// check if the installation itself owns the target
		if lsv1alpha1helper.ReferenceIsObject(ref, v.Inst.Info) {
			continue
		}

		// check if the target comes from the parent
		if v.parent != nil && lsv1alpha1helper.ReferenceIsObject(ref, v.parent.Info) {
			err := v.checkStateForParentImport(fldPath, targetImportReference)
			if err != nil {
				allErrs = append(allErrs, err)
			}
			continue
		}

		// otherwise validate as sibling export
		err := v.checkStateForSiblingTargetExport(ctx, fldPath, ref, targetImportReference)
		if err != nil {
			allErrs = append(allErrs, err)
		}
	}
	return errors.NewAggregate(allErrs)
}

func (v *Validator) checkStateForParentImport(fldPath *field.Path, importName string) error {
	// check if the parent also imports my import
	_, err := v.parent.GetImportDefinition(importName)
	if err != nil {
		return installations.NewImportNotFoundErrorf(err, "%s: import in parent not found", fldPath.String())
	}
	// parent has to be progressing
	if !lsv1alpha1helper.IsProgressingInstallationPhase(v.parent.Info.Status.Phase) {
		return installations.NewImportNotSatisfiedErrorf(nil, "%s: Parent has to be progressing to get imports", fldPath.String())
	}
	return nil
}

// The isExportingDataFunc returns true if the passed sibling has a Data/Target export by the given name.
type isExportingDataFunc func(*installations.InstallationBase, string) bool

func (v *Validator) checkStateForSiblingDataExport(ctx context.Context, fldPath *field.Path, siblingRef lsv1alpha1.ObjectReference, importName string) error {
	isExportingFunc := func(sibling *installations.InstallationBase, name string) bool {
		return sibling.IsExportingData(name)
	}
	return v.checkStateForSiblingExport(ctx, fldPath, siblingRef, importName, isExportingFunc)
}

func (v *Validator) checkStateForSiblingTargetExport(ctx context.Context, fldPath *field.Path, siblingRef lsv1alpha1.ObjectReference, importName string) error {
	isExportingFunc := func(sibling *installations.InstallationBase, name string) bool {
		return sibling.IsExportingTarget(name)
	}
	return v.checkStateForSiblingExport(ctx, fldPath, siblingRef, importName, isExportingFunc)
}

func (v *Validator) checkStateForSiblingExport(ctx context.Context, fldPath *field.Path, siblingRef lsv1alpha1.ObjectReference, importName string, isExporting isExportingDataFunc) error {
	sibling := v.getSiblingForObjectReference(siblingRef)
	if sibling == nil {
		return fmt.Errorf("%s: installation %s is not a sibling", fldPath.String(), siblingRef.NamespacedName().String())
	}

	// search in the sibling for the export mapping where importmap.from == exportmap.to
	if !isExporting(sibling, importName) {
		return installations.NewImportNotFoundErrorf(nil, "%s: export in sibling not found", fldPath.String())
	}

	if sibling.Info.Status.Phase != lsv1alpha1.ComponentPhaseSucceeded {
		return installations.NewNotCompletedDependentsErrorf(nil, "%s: Sibling Installation has to be completed to get exports", fldPath.String())
	}

	// we expect that no dependent siblings are running
	isCompleted, err := CheckCompletedSiblingDependents(ctx, v.Operation.Client(), v.Operation.Context().Name, sibling)
	if err != nil {
		return fmt.Errorf("%s: Unable to check if sibling Installation %q dependencies are not completed yet: %w", fldPath.String(), sibling.Info.Name, err)
	}
	if !isCompleted {
		return installations.NewNotCompletedDependentsErrorf(nil, "%s: Sibling Installation %q dependency is not completed yet", fldPath.String(), sibling.Info.Name)
	}

	return nil
}

func (v *Validator) getSiblingForObjectReference(ref lsv1alpha1.ObjectReference) *installations.InstallationBase {
	for _, sibling := range v.siblings {
		if lsv1alpha1helper.ReferenceIsObject(ref, sibling.Info) {
			return sibling
		}
	}
	return nil
}

// IsRoot returns true if the current component is a root component
func (v *Validator) IsRoot() bool {
	return v.parent == nil
}

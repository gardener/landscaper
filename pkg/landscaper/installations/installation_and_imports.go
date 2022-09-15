package installations

import (
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
)

// InstallationBase is the internal representation of an installation without resolved blueprint.
type InstallationAndImports struct {
	imports      map[string]interface{}
	installation *lsv1alpha1.Installation
	// indexes the import state with from/to as key
	importsStatus ImportStatus
}

// NewInstallationAndImports creates a new object containing the installation, the imports and the status of the imports
func NewInstallationAndImports(inst *lsv1alpha1.Installation) *InstallationAndImports {
	internalInst := InstallationAndImports{
		installation: inst,
		importsStatus: ImportStatus{
			Data:   make(map[string]*lsv1alpha1.ImportStatus, len(inst.Status.Imports)),
			Target: make(map[string]*lsv1alpha1.ImportStatus, len(inst.Status.Imports)),
		},
	}

	for _, importStatus := range inst.Status.Imports {
		internalInst.importsStatus.set(importStatus)
	}

	return &internalInst
}

// ImportStatus returns the internal representation of the internal import state
func (i *InstallationAndImports) ImportStatus() *ImportStatus {
	return &i.importsStatus
}

func (i *InstallationAndImports) GetImports() map[string]interface{} {
	return i.imports
}

func (i *InstallationAndImports) SetImports(imports map[string]interface{}) {
	i.imports = imports
}

func (i *InstallationAndImports) GetInstallation() *lsv1alpha1.Installation {
	return i.installation
}

// IsExportingData checks if the current component exports a data object with the given name.
func (i *InstallationAndImports) IsExportingData(name string) bool {
	for _, def := range i.installation.Spec.Exports.Data {
		if def.DataRef == name {
			return true
		}
	}
	return false
}

// IsExportingTarget checks if the current component exports a target with the given name.
func (i *InstallationAndImports) IsExportingTarget(name string) bool {
	for _, def := range i.installation.Spec.Exports.Targets {
		if def.Target == name {
			return true
		}
	}
	return false
}

// IsImportingData checks if the current component imports a data object with the given name.
func (i *InstallationAndImports) IsImportingData(name string) bool {
	for _, def := range i.installation.Spec.Imports.Data {
		if def.DataRef == name {
			return true
		}
	}
	return false
}

// IsImportingTarget checks if the current component imports a target with the given name.
func (i *InstallationAndImports) IsImportingTarget(name string) bool {
	for _, def := range i.installation.Spec.Imports.Targets {
		if def.Target == name {
			return true
		}
	}
	return false
}

// MergeConditions updates or adds the given condition to the installation's condition.
func (i *InstallationAndImports) MergeConditions(conditions ...lsv1alpha1.Condition) {
	i.installation.Status.Conditions = lsv1alpha1helper.MergeConditions(i.installation.Status.Conditions, conditions...)
}

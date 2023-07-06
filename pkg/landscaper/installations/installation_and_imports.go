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

// MergeConditions updates or adds the given condition to the installation's condition.
func (i *InstallationAndImports) MergeConditions(conditions ...lsv1alpha1.Condition) {
	i.installation.Status.Conditions = lsv1alpha1helper.MergeConditions(i.installation.Status.Conditions, conditions...)
}

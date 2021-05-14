// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"fmt"

	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

// InstallationBase is the internal representation of an installation without resolved blueprint.
type InstallationBase struct {
	Info *lsv1alpha1.Installation
	// indexes the import state with from/to as key
	importsStatus ImportStatus
}

// NewInstallationBase creates a new internal representation of an installation without blueprint
func NewInstallationBase(inst *lsv1alpha1.Installation) *InstallationBase {
	internalInst := newInstallationBase(inst)
	return &internalInst
}

// newInstallationBase creates a new internal representation of an installation without blueprint
func newInstallationBase(inst *lsv1alpha1.Installation) InstallationBase {
	internalInst := InstallationBase{
		Info: inst,
		importsStatus: ImportStatus{
			Data:   make(map[string]*lsv1alpha1.ImportStatus, len(inst.Status.Imports)),
			Target: make(map[string]*lsv1alpha1.ImportStatus, len(inst.Status.Imports)),
		},
	}

	for _, importStatus := range inst.Status.Imports {
		internalInst.importsStatus.set(importStatus)
	}

	return internalInst
}

// ImportStatus returns the internal representation of the internal import state
func (i *InstallationBase) ImportStatus() *ImportStatus {
	return &i.importsStatus
}

func (i *InstallationBase) GetInfo() *lsv1alpha1.Installation {
	return i.Info
}

// IsExportingData checks if the current component exports a data object with the given name.
func (i *InstallationBase) IsExportingData(name string) bool {
	for _, def := range i.Info.Spec.Exports.Data {
		if def.DataRef == name {
			return true
		}
	}
	return false
}

// IsExportingTarget checks if the current component exports a target with the given name.
func (i *InstallationBase) IsExportingTarget(name string) bool {
	for _, def := range i.Info.Spec.Exports.Targets {
		if def.Target == name {
			return true
		}
	}
	return false
}

// IsImportingData checks if the current component imports a data object with the given name.
func (i *InstallationBase) IsImportingData(name string) bool {
	for _, def := range i.Info.Spec.Imports.Data {
		if def.DataRef == name {
			return true
		}
	}
	return false
}

// IsImportingTarget checks if the current component imports a target with the given name.
func (i *InstallationBase) IsImportingTarget(name string) bool {
	for _, def := range i.Info.Spec.Imports.Targets {
		if def.Target == name {
			return true
		}
	}
	return false
}

// MergeConditions updates or adds the given condition to the installation's condition.
func (i *InstallationBase) MergeConditions(conditions ...lsv1alpha1.Condition) {
	i.Info.Status.Conditions = lsv1alpha1helper.MergeConditions(i.Info.Status.Conditions, conditions...)
}

// Installation is the internal representation of a installation
type Installation struct {
	InstallationBase
	Blueprint *blueprints.Blueprint
}

// New creates a new internal representation of an installation with blueprint
func New(inst *lsv1alpha1.Installation, blueprint *blueprints.Blueprint) (*Installation, error) {
	internalInst := &Installation{
		InstallationBase: newInstallationBase(inst),
		Blueprint:        blueprint,
	}

	return internalInst, nil
}

// GetImportDefinition return the import for a given key
func (i *Installation) GetImportDefinition(key string) (lsv1alpha1.ImportDefinition, error) {
	for _, def := range i.Blueprint.Info.Imports {
		if def.Name == key {
			return def, nil
		}
	}
	return lsv1alpha1.ImportDefinition{}, fmt.Errorf("import with key %s not found", key)
}

// GetExportDefinition return the export definition for a given key
func (i *Installation) GetExportDefinition(key string) (lsv1alpha1.ExportDefinition, error) {
	for _, def := range i.Blueprint.Info.Exports {
		if def.Name == key {
			return def, nil
		}
	}
	return lsv1alpha1.ExportDefinition{}, fmt.Errorf("export with key %s not found", key)
}

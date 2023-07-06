// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints

import (
	"fmt"
	"os"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/core"
	"github.com/gardener/landscaper/apis/core/validation"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
)

// Blueprint is the internal resolved type of a blueprint.
type Blueprint struct {
	Info *lsv1alpha1.Blueprint
	Fs   vfs.FileSystem
}

// New creates a new internal Blueprint from a blueprint definition and its filesystem content.
func New(blueprint *lsv1alpha1.Blueprint, content vfs.FileSystem) *Blueprint {
	b := &Blueprint{
		Info: blueprint,
		Fs:   content,
	}
	return b
}

// NewFromFs creates a new internal Blueprint from a filesystem.
// The blueprint is automatically read from within the filesystem
func NewFromFs(content vfs.FileSystem) (*Blueprint, error) {
	data, err := vfs.ReadFile(content, lsv1alpha1.BlueprintFileName)
	if err != nil {
		return nil, fmt.Errorf("unable to read blueprint from filesystem: %w", err)
	}
	blueprint := &lsv1alpha1.Blueprint{}
	if _, _, err := api.Decoder.Decode(data, nil, blueprint); err != nil {
		return nil, fmt.Errorf("unable to decode blueprint: %w", err)
	}
	return New(blueprint, content), nil
}

func (b *Blueprint) GetImportByName(name string) *lsv1alpha1.ImportDefinition {
	for _, elem := range b.Info.Imports {
		if elem.Name == name {
			return &elem
		}
	}
	return nil
}

// GetSubinstallations gets the direct subinstallation templates for a blueprint.
func (b *Blueprint) GetSubinstallations() ([]*lsv1alpha1.InstallationTemplate, error) {
	var (
		allErrs   field.ErrorList
		fldPath   = field.NewPath("subinstallations")
		templates = make([]*lsv1alpha1.InstallationTemplate, len(b.Info.Subinstallations))
	)
	for i, subInstTmpl := range b.Info.Subinstallations {
		instPath := fldPath.Index(i)
		if subInstTmpl.InstallationTemplate != nil {
			templates[i] = subInstTmpl.InstallationTemplate
			continue
		}

		if len(subInstTmpl.File) == 0 {
			return nil, fmt.Errorf("neither a inline installation template nor a file is defined in index %d", i)
		}
		data, err := vfs.ReadFile(b.Fs, subInstTmpl.File)
		if err != nil {
			if os.IsNotExist(err) {
				allErrs = append(allErrs, field.NotFound(instPath.Child("file"), subInstTmpl.File))
				continue
			}
			allErrs = append(allErrs, field.InternalError(instPath.Child("file"), err))
			continue
		}

		coreInstTmpl := &core.InstallationTemplate{}
		if _, _, err := api.Decoder.Decode(data, nil, coreInstTmpl); err != nil {
			allErrs = append(allErrs, field.Invalid(
				instPath.Child("file"),
				subInstTmpl.File,
				fmt.Sprintf("unable to decode installation template: %s", err.Error())))
			continue
		}
		if valErrs := validation.ValidateInstallationTemplate(instPath, coreInstTmpl); len(valErrs) != 0 {
			allErrs = append(allErrs, valErrs...)
			continue
		}

		instTmpl := &lsv1alpha1.InstallationTemplate{}
		if err := lsv1alpha1.Convert_core_InstallationTemplate_To_v1alpha1_InstallationTemplate(coreInstTmpl, instTmpl, nil); err != nil {
			allErrs = append(allErrs, field.InternalError(instPath, err))
			continue
		}
		templates[i] = instTmpl
	}
	if len(allErrs) != 0 {
		return nil, allErrs.ToAggregate()
	}

	return templates, nil
}

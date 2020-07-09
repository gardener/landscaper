// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exports

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/dataobject"
	"github.com/gardener/landscaper/pkg/landscaper/datatype"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/landscapeconfig"
)

// Validators is a struct that contains everything to
// validate if all imports of a installation are satisfied.
type Validator struct {
	*installations.Operation

	lsConfig *landscapeconfig.LandscapeConfig
	parent   *installations.Installation
	siblings []*installations.Installation
}

// NewValidator creates a new export validator.
func NewValidator(op *installations.Operation) *Validator {
	return &Validator{
		Operation: op,
	}
}

// Validate validates the exports of a installation and
// checks if the config is of the configured form and type.
func (v *Validator) Validate(ctx context.Context, inst *installations.Installation, values map[string]interface{}) error {
	fldPath := field.NewPath(inst.Info.Name)
	cond := lsv1alpha1helper.GetOrInitCondition(inst.Info.Status.Conditions, lsv1alpha1.ValidateExportCondition)

	do := &dataobject.DataObject{Data: values}

	if err := v.validateExports(ctx, fldPath, inst, do); err != nil {
		cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			"ValidationFailed", "Export validation failed")
		_ = v.UpdateInstallationStatus(ctx, inst.Info, lsv1alpha1.ComponentPhaseFailed, cond)
		return err
	}

	cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
		"Exports validated", "All exported fields are successfully validated")
	return v.UpdateInstallationStatus(ctx, inst.Info, inst.Info.Status.Phase, cond)
}

func (v *Validator) validateExports(ctx context.Context, fldPath *field.Path, inst *installations.Installation, do *dataobject.DataObject) error {
	for i, exportMapping := range inst.Info.Spec.Exports {
		expPath := fldPath.Index(i)

		exportDef, err := inst.GetExportDefinition(exportMapping.From)
		if err != nil {
			return err
		}

		dt, ok := v.GetDataType(exportDef.Type)
		if !ok {
			return fmt.Errorf("%s: cannot find DataType %s", expPath.String(), exportDef.Type)
		}

		var data interface{}
		if err := do.GetData(exportMapping.To, &data); err != nil {
			return errors.Wrapf(err, "%s: unable to get data", expPath.String())
		}

		if err := datatype.Validate(*dt, data); err != nil {
			return errors.Wrapf(err, "%s: unable to validate data against %s", expPath.String(), exportDef.Type)
		}
	}
	return nil
}

func (v *Validator) getExportConfig(ctx context.Context, inst *installations.Installation) (*dataobject.DataObject, error) {
	return nil, nil
}

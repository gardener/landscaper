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

package jsonschema

import (
	"github.com/xeipuuv/gojsonschema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type Validator struct {
	Config *LoaderConfig
}

// ValidateSchema validates a jsonschema schema definition.
func ValidateSchema(schemaBytes []byte) error {
	_, err := gojsonschema.NewSchemaLoader().Compile(gojsonschema.NewBytesLoader(schemaBytes))
	if err != nil {
		return err
	}
	return nil
}

func (v *Validator) ValidateGoStruct(schemaBytes []byte, data interface{}) error {
	return v.validate(schemaBytes, gojsonschema.NewGoLoader(data))
}

func (v *Validator) ValidateBytes(schemaBytes []byte, data []byte) error {
	return v.validate(schemaBytes, gojsonschema.NewBytesLoader(data))
}

func (v *Validator) validate(schemaBytes []byte, documentLoader gojsonschema.JSONLoader) error {
	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)

	// Wrap default loader if config is defined
	if v.Config != nil {
		schemaLoader = NewWrappedLoader(*v.Config, schemaLoader)
	}

	sl := gojsonschema.NewSchemaLoader()
	schema, err := sl.Compile(schemaLoader)
	if err != nil {
		return err
	}

	res, err := schema.Validate(documentLoader)
	if err != nil {
		return err
	}

	if !res.Valid() {
		var allErrs field.ErrorList
		for _, err := range res.Errors() {
			allErrs = append(allErrs, field.Invalid(field.NewPath(err.Field()), err.Value(), err.Description()))
		}
		return allErrs.ToAggregate()
	}
	return nil
}

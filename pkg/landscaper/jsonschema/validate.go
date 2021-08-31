// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

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

// ValidateSchemaWithLoader validates a jsonschema schema definition by using the given loader.
func ValidateSchemaWithLoader(loader gojsonschema.JSONLoader) error {
	_, err := gojsonschema.NewSchemaLoader().Compile(loader)
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

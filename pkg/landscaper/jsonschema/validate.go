// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package jsonschema

import (
	"errors"

	"github.com/xeipuuv/gojsonschema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type Validator struct {
	Context *ReferenceContext
	Schema  *gojsonschema.Schema
}

// NewValidator returns a new Validator with the given reference context.
func NewValidator(context *ReferenceContext) *Validator {
	return &Validator{
		Context: context,
	}
}

// ValidateSchema validates a jsonschema schema definition.
func ValidateSchema(schemaBytes []byte) error {
	_, err := gojsonschema.NewSchemaLoader().Compile(gojsonschema.NewBytesLoader(schemaBytes))
	return err
}

// ValidateSchemaWithLoader validates a jsonschema schema definition by using the given loader.
func ValidateSchemaWithLoader(loader gojsonschema.JSONLoader) error {
	_, err := gojsonschema.NewSchemaLoader().Compile(loader)
	return err
}

func ValidateGoStruct(schemaBytes []byte, data interface{}, context *ReferenceContext) error {
	return validate(schemaBytes, gojsonschema.NewGoLoader(data), context)
}

func ValidateBytes(schemaBytes []byte, data []byte, context *ReferenceContext) error {
	return validate(schemaBytes, gojsonschema.NewBytesLoader(data), context)
}

func validate(schemaBytes []byte, documentLoader gojsonschema.JSONLoader, context *ReferenceContext) error {
	v := NewValidator(context)
	err := v.CompileSchema(schemaBytes)
	if err != nil {
		return err
	}
	return v.validate(documentLoader)
}

// CompileSchema compiles the given schema and sets it as schema for the validator
func (v *Validator) CompileSchema(schemaBytes []byte) error {
	ref := NewReferenceResolver(v.Context)
	resolved, err := ref.Resolve(schemaBytes)
	if err != nil {
		return err
	}
	schema, err := gojsonschema.NewSchemaLoader().Compile(gojsonschema.NewGoLoader(resolved))
	if err != nil {
		return err
	}
	v.Schema = schema
	return nil
}

func (v *Validator) ValidateGoStruct(data interface{}) error {
	return v.validate(gojsonschema.NewGoLoader(data))
}

func (v *Validator) ValidateBytes(data []byte) error {
	return v.validate(gojsonschema.NewBytesLoader(data))
}

func (v *Validator) validate(documentLoader gojsonschema.JSONLoader) error {
	if v.Schema == nil {
		return errors.New("internal error: schema has not been compiled")
	}
	res, err := v.Schema.Validate(documentLoader)
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

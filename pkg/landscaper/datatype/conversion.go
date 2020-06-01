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

package datatype

import (
	"github.com/go-openapi/spec"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
)

func ConvertJSONSchemaProps(in *lsv1alpha1.JSONSchemaProps, out *spec.Schema) error {
	out.ID = in.ID
	out.Schema = spec.SchemaURL(in.Schema)
	out.Description = in.Description
	if in.Type != "" {
		out.Type = spec.StringOrArray([]string{in.Type})
	}
	out.Nullable = in.Nullable
	out.Format = in.Format
	out.Title = in.Title
	out.ExclusiveMaximum = in.ExclusiveMaximum
	out.ExclusiveMinimum = in.ExclusiveMinimum
	out.MaxLength = in.MaxLength
	out.MinLength = in.MinLength
	out.Pattern = in.Pattern
	out.MaxItems = in.MaxItems
	out.MinItems = in.MinItems
	out.UniqueItems = in.UniqueItems
	out.MaxProperties = in.MaxProperties
	out.MinProperties = in.MinProperties
	out.Required = in.Required

	if in.Maximum != nil {
		f, err := lsv1alpha1helper.DecimalToFloat64(*in.Maximum)
		if err != nil {
			return err
		}
		out.Maximum = &f
	}
	if in.Minimum != nil {
		f, err := lsv1alpha1helper.DecimalToFloat64(*in.Minimum)
		if err != nil {
			return err
		}
		out.Minimum = &f
	}
	if in.MultipleOf != nil {
		f, err := lsv1alpha1helper.DecimalToFloat64(*in.MultipleOf)
		if err != nil {
			return err
		}
		out.MultipleOf = &f
	}

	if in.Default != nil {
		out.Default = *(in.Default)
	}
	if in.Example != nil {
		out.Example = *(in.Example)
	}

	if in.Enum != nil {
		out.Enum = make([]interface{}, len(in.Enum))
		for k, v := range in.Enum {
			out.Enum[k] = v
		}
	}

	if err := convertSliceOfJSONSchemaProps(&in.AllOf, &out.AllOf); err != nil {
		return err
	}
	if err := convertSliceOfJSONSchemaProps(&in.OneOf, &out.OneOf); err != nil {
		return err
	}
	if err := convertSliceOfJSONSchemaProps(&in.AnyOf, &out.AnyOf); err != nil {
		return err
	}

	if in.Not != nil {
		in, out := &in.Not, &out.Not
		*out = new(spec.Schema)
		if err := ConvertJSONSchemaProps(*in, *out); err != nil {
			return err
		}
	}

	var err error
	out.Properties, err = convertMapOfJSONSchemaProps(in.Properties)
	if err != nil {
		return err
	}

	out.PatternProperties, err = convertMapOfJSONSchemaProps(in.PatternProperties)
	if err != nil {
		return err
	}

	out.Definitions, err = convertMapOfJSONSchemaProps(in.Definitions)
	if err != nil {
		return err
	}

	if in.Ref != nil {
		out.Ref, err = spec.NewRef(*in.Ref)
		if err != nil {
			return err
		}
	}

	if in.AdditionalProperties != nil {
		in, out := &in.AdditionalProperties, &out.AdditionalProperties
		*out = new(spec.SchemaOrBool)
		if err := convertJSONSchemaPropsOrBool(*in, *out); err != nil {
			return err
		}
	}

	if in.AdditionalItems != nil {
		in, out := &in.AdditionalItems, &out.AdditionalItems
		*out = new(spec.SchemaOrBool)
		if err := convertJSONSchemaPropsOrBool(*in, *out); err != nil {
			return err
		}
	}

	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = new(spec.SchemaOrArray)
		if err := convertJSONSchemaPropsOrArray(*in, *out); err != nil {
			return err
		}
	}

	if in.Dependencies != nil {
		in, out := &in.Dependencies, &out.Dependencies
		*out = make(spec.Dependencies, len(*in))
		for key, val := range *in {
			newVal := new(spec.SchemaOrStringArray)
			if err := convertJSONSchemaPropsOrStringArray(&val, newVal); err != nil {
				return err
			}
			(*out)[key] = *newVal
		}
	}

	if in.ExternalDocs != nil {
		out.ExternalDocs = &spec.ExternalDocumentation{}
		out.ExternalDocs.Description = in.ExternalDocs.Description
		out.ExternalDocs.URL = in.ExternalDocs.URL
	}

	return nil
}

func convertSliceOfJSONSchemaProps(in *[]lsv1alpha1.JSONSchemaProps, out *[]spec.Schema) error {
	if in != nil {
		for _, jsonSchemaProps := range *in {
			schema := spec.Schema{}
			if err := ConvertJSONSchemaProps(&jsonSchemaProps, &schema); err != nil {
				return err
			}
			*out = append(*out, schema)
		}
	}
	return nil
}

func convertMapOfJSONSchemaProps(in map[string]lsv1alpha1.JSONSchemaProps) (map[string]spec.Schema, error) {
	if in == nil {
		return nil, nil
	}

	out := make(map[string]spec.Schema)
	for k, jsonSchemaProps := range in {
		schema := spec.Schema{}
		if err := ConvertJSONSchemaProps(&jsonSchemaProps, &schema); err != nil {
			return nil, err
		}
		out[k] = schema
	}
	return out, nil
}

func convertJSONSchemaPropsOrArray(in *lsv1alpha1.JSONSchemaPropsOrArray, out *spec.SchemaOrArray) error {
	if in.Schema != nil {
		in, out := &in.Schema, &out.Schema
		*out = new(spec.Schema)
		if err := ConvertJSONSchemaProps(*in, *out); err != nil {
			return err
		}
	}
	if in.JSONSchemas != nil {
		in, out := &in.JSONSchemas, &out.Schemas
		*out = make([]spec.Schema, len(*in))
		for i := range *in {
			if err := ConvertJSONSchemaProps(&(*in)[i], &(*out)[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func convertJSONSchemaPropsOrBool(in *lsv1alpha1.JSONSchemaPropsOrBool, out *spec.SchemaOrBool) error {
	out.Allows = in.Allows
	if in.Schema != nil {
		in, out := &in.Schema, &out.Schema
		*out = new(spec.Schema)
		if err := ConvertJSONSchemaProps(*in, *out); err != nil {
			return err
		}
	}
	return nil
}

func convertJSONSchemaPropsOrStringArray(in *lsv1alpha1.JSONSchemaPropsOrStringArray, out *spec.SchemaOrStringArray) error {
	out.Property = in.Property
	if in.Schema != nil {
		in, out := &in.Schema, &out.Schema
		*out = new(spec.Schema)
		if err := ConvertJSONSchemaProps(*in, *out); err != nil {
			return err
		}
	}
	return nil
}

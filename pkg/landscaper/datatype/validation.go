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
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
	"github.com/hashicorp/go-multierror"

	landscaperv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// Validate validates data for a certain datatype
func Validate(scheme landscaperv1alpha1.JSONSchemaProps, data interface{}) error {

	// convert versioned JSONSchemaProps to apiextensions internal version
	//internalscheme := &apiextensions.JSONSchemaProps{}
	//if err := apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(&scheme.JSONSchemaProps, internalscheme, nil); err != nil {
	//	return err
	//}

	openAPITypes := &spec.Schema{}
	if err := ConvertJSONSchemaProps(&scheme, openAPITypes); err != nil {
		return err
	}

	schemeValidator := validate.NewSchemaValidator(openAPITypes, nil, "", strfmt.Default)
	res := schemeValidator.Validate(data)

	if len(res.Errors) == 0 {
		return nil
	}

	var allErrs *multierror.Error
	for _, err := range res.Errors {
		allErrs = multierror.Append(allErrs, err)
	}

	return allErrs
}

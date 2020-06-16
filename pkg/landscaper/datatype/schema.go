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
	"fmt"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
)

type Datatype struct {
	Info       *lsv1alpha1.DataType
	Referenced []*lsv1alpha1.DataType
}

// New creates a new internal datatype
func New(dt *lsv1alpha1.DataType, refs []*lsv1alpha1.DataType) *Datatype {
	return &Datatype{
		Info:       dt,
		Referenced: refs,
	}
}

// CreateDatatypesMap creates a map to of datatype name -> internal datatype
func CreateDatatypesMap(datatypes []lsv1alpha1.DataType) (map[string]*Datatype, error) {
	rawDTMap := make(map[string]*lsv1alpha1.DataType, 0)
	for _, obj := range datatypes {
		dt := obj
		rawDTMap[dt.Name] = &dt
	}

	dtMap := make(map[string]*Datatype, 0)
	for _, obj := range datatypes {
		dt := obj
		usedReferences := lsv1alpha1helper.GetUsedReferencedSchemes(&dt.Schema.OpenAPIV3Schema)

		refs := make([]*lsv1alpha1.DataType, len(usedReferences))
		for i, ref := range usedReferences.List() {
			usedDT, ok := rawDTMap[ref]
			if !ok {
				return nil, fmt.Errorf("datatype %s is used but cannot be found", ref)
			}
			refs[i] = usedDT
		}

		dtMap[dt.Name] = New(&dt, refs)
	}

	return dtMap, nil
}

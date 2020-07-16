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

package dataobject

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/dataobject/jsonpath"
)

type DataObject struct {
	Raw  *corev1.Secret
	Data map[string]interface{}
}

func New(secret *corev1.Secret) (*DataObject, error) {
	data := make(map[string]interface{})

	raw, ok := secret.Data[lsv1alpha1.DataObjectSecretDataKey]
	if !ok {
		return nil, errors.New("secret does not contain any data")
	}

	if err := yaml.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return &DataObject{
		Raw:  secret,
		Data: data,
	}, nil
}

// GetData searches its data for the given Javscript Object Notation path
// and unmarshals it into the given object
func (do *DataObject) GetData(path string, out interface{}) error {
	return jsonpath.GetValue(path, do.Data, out)
}

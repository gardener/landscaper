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

package landscapeconfig

import (
	corev1 "k8s.io/api/core/v1"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/dataobject"
)

// LandscapeConfig is the internal representation of a LandscapeConfiguration
type LandscapeConfig struct {
	Info *lsv1alpha1.LandscapeConfiguration
	Data *dataobject.DataObject
}

// New creates a new internal landscape configuration
func New(lsConfig *lsv1alpha1.LandscapeConfiguration, secret *corev1.Secret) (*LandscapeConfig, error) {
	data, err := dataobject.New(secret)
	if err != nil {
		return nil, err
	}

	return &LandscapeConfig{
		Info: lsConfig,
		Data: data,
	}, nil
}

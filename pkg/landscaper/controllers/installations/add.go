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

package installations

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/registry"
	"github.com/gardener/landscaper/pkg/utils/componentrepository"
)

func AddActuatorToManager(mgr manager.Manager, registry registry.Registry, comp componentrepository.Client) error {
	a, err := NewActuator(registry, comp)
	if err != nil {
		return err
	}

	if _, err := inject.LoggerInto(ctrl.Log.WithName("controllers").WithName("ComponentInstallations"), a); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Installation{}).
		Owns(&v1alpha1.Execution{}).
		Owns(&v1alpha1.Installation{}).
		Complete(a)
}

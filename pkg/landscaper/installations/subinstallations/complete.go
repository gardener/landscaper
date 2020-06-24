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

package subinstallations

import (
	"context"

	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/installations"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// CombinedState returns the combined state of all subinstallations
func (o *Operation) CombinedState(ctx context.Context, inst *installations.Installation) (lsv1alpha1.ComponentInstallationPhase, error) {
	subinsts, err := o.getSubInstallations(ctx, inst.Info)
	if err != nil {
		return lsv1alpha1.ComponentPhaseFailed, err
	}

	phases := make([]lsv1alpha1.ComponentInstallationPhase, len(subinsts))

	for _, v := range subinsts {
		phases = append(phases, v.Status.Phase)
	}

	return helper.CombinedInstallationPhase(phases...), nil
}

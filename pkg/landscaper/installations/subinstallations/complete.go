// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package subinstallations

import (
	"context"

	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/installations"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// CombinedState returns the combined state of all subinstallations
func (o *Operation) CombinedState(ctx context.Context, inst *installations.Installation) (lsv1alpha1.ComponentInstallationPhase, error) {
	subinsts, err := o.GetSubInstallations(ctx, inst.Info)
	if err != nil {
		return "", err
	}

	phases := make([]lsv1alpha1.ComponentInstallationPhase, len(subinsts))

	for _, v := range subinsts {
		if v.Generation != v.Status.ObservedGeneration {
			phases = append(phases, lsv1alpha1.ComponentPhaseProgressing)
		} else {
			phases = append(phases, v.Status.Phase)
		}
	}

	return helper.CombinedInstallationPhase(phases...), nil
}

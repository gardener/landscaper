// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package continuousreconcile

import (
	"strings"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// ContinuousReconcileSpec represents the specification of a continuous reconcile schedule.
type ContinuousReconcileSpec struct {
	// Cron is a standard crontab specification of the reconciliation schedule.
	// Either Cron or Every has to be specified.
	// +optional
	Cron string `json:"cron,omitempty"`

	// Every specifies a delay after which the reconcile should happen.
	// Either Cron or Every has to be specified.
	// +optional
	Every *lsv1alpha1.Duration `json:"every,omitempty"`
}

func (crs ContinuousReconcileSpec) String() string {
	var sb strings.Builder
	sb.WriteString("{")
	if len(crs.Cron) != 0 {
		sb.WriteString("Cron: \"")
		sb.WriteString(crs.Cron)
		sb.WriteString("\"")
		if crs.Every != nil && crs.Every.Duration != 0 {
			sb.WriteString(", ")
		}
	}
	if crs.Every != nil && crs.Every.Duration != 0 {
		sb.WriteString("Every: \"")
		sb.WriteString(crs.Every.String())
		sb.WriteString("\"")
	}
	sb.WriteString("}")
	return sb.String()
}

// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package continuousreconcile

import (
	lscore "github.com/gardener/landscaper/apis/core"
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
	Every *lscore.Duration `json:"every,omitempty"`
}

// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	componentcliMetrics "github.com/gardener/component-cli/ociclient/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

/*
  This package contains all metrics that are exposed by the landscaper.
  It offers a function to register the metrics on a prometheus registry
*/

// RegisterMetrics allows to register all landscaper exposed metrics
func RegisterMetrics(reg prometheus.Registerer) {
	componentcliMetrics.RegisterCacheMetrics(reg)
}

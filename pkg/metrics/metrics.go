// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	componentcliMetrics "github.com/gardener/component-cli/ociclient/metrics"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gardener/landscaper/pkg/components/cache"

	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

/*
  This package contains all metrics that are exposed by the landscaper.
  It offers a function to register the metrics on a prometheus registry
*/

// RegisterMetrics allows to register all landscaper exposed metrics
func RegisterMetrics(reg prometheus.Registerer) {
	cache.RegisterStoreMetrics(reg)
	blueprints.RegisterStoreMetrics(reg)
	componentcliMetrics.RegisterCacheMetrics(reg)
}

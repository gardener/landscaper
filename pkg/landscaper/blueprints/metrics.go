// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints

import (
	"github.com/prometheus/client_golang/prometheus"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

const (
	storeSubsystemName = "blueprintStore"
)

var (
	// DiskUsage discloses disk used by the blueprint store
	DiskUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: lsv1alpha1.LandscaperMetricsNamespaceName,
			Subsystem: storeSubsystemName,
			Name:      "disk_usage_bytes",
			Help:      "Bytes on disk currently used by blueprint store instance.",
		},
	)

	// StoredItems discloses the number of items stored by the blueprint store.
	StoredItems = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: lsv1alpha1.LandscaperMetricsNamespaceName,
			Subsystem: storeSubsystemName,
			Name:      "items_total",
			Help:      "Total number of items currently stored by the blueprint store.",
		},
	)
)

// RegisterStoreMetrics allows to register blueprint store metrics with a given prometheus registerer
func RegisterStoreMetrics(reg prometheus.Registerer) {
	reg.MustRegister(DiskUsage)
	reg.MustRegister(StoredItems)
}

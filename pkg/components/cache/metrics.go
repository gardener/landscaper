// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"github.com/prometheus/client_golang/prometheus"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

const (
	storeSubsystemName = "blueprintCacheStore"

	ociClientNamespaceName = "ociclient"
	cacheSubsystemName     = "blueprintCache"
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

	// CacheMemoryUsage discloses memory used by caches
	CacheMemoryUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: ociClientNamespaceName,
			Subsystem: cacheSubsystemName,
			Name:      "memory_usage_bytes",
			Help:      "Bytes in memory currently used by cache instance.",
		},
		[]string{"id"},
	)

	// CacheDiskUsage discloses disk used by caches
	CacheDiskUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: ociClientNamespaceName,
			Subsystem: cacheSubsystemName,
			Name:      "disk_usage_bytes",
			Help:      "Bytes on disk currently used by cache instance.",
		},
		[]string{"id"},
	)

	// CachedItems discloses the number of items stored by caches
	CachedItems = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: ociClientNamespaceName,
			Subsystem: cacheSubsystemName,
			Name:      "items_total",
			Help:      "Total number of items currently cached by instance.",
		},
		[]string{"id"},
	)

	// CacheHitsDisk discloses the number of hits for items cached on disk
	CacheHitsDisk = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: ociClientNamespaceName,
			Subsystem: cacheSubsystemName,
			Name:      "disk_hits_total",
			Help:      "Total number of hits for items cached on disk by an instance.",
		},
		[]string{"id"},
	)

	// CacheHitsMemory discloses the number of hits for items cached in memory
	CacheHitsMemory = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: ociClientNamespaceName,
			Subsystem: cacheSubsystemName,
			Name:      "memory_hits_total",
			Help:      "Total number of hits for items cached in memory by an instance.",
		},
		[]string{"id"},
	)
)

// RegisterStoreMetrics allows to register blueprint store metrics with a given prometheus registerer
func RegisterStoreMetrics(reg prometheus.Registerer) {
	reg.MustRegister(DiskUsage)
	reg.MustRegister(StoredItems)

	reg.MustRegister(CacheHitsDisk)
	reg.MustRegister(CacheHitsMemory)
	reg.MustRegister(CachedItems)
	reg.MustRegister(CacheDiskUsage)
	reg.MustRegister(CacheMemoryUsage)
}

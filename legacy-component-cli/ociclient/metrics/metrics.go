// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import "github.com/prometheus/client_golang/prometheus"

const (
	ociClientNamespaceName = "ociclient"
	cacheSubsystemName     = "cache"
)

var (
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

// RegisterCacheMetrics allows to register ociclient cache metrics with a given prometheus registerer
func RegisterCacheMetrics(reg prometheus.Registerer) {
	reg.MustRegister(CacheHitsDisk)
	reg.MustRegister(CacheHitsMemory)
	reg.MustRegister(CachedItems)
	reg.MustRegister(CacheDiskUsage)
	reg.MustRegister(CacheMemoryUsage)
}

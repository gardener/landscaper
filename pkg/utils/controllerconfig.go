// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/gardener/landscaper/apis/config"
)

// ConvertCommonControllerConfigToControllerOptions converts the landscaper CommonControllerConfig to controller.Options.
func ConvertCommonControllerConfigToControllerOptions(cfg config.CommonControllerConfig) controller.Options {
	opts := controller.Options{
		MaxConcurrentReconciles: cfg.Workers,
	}
	if cfg.CacheSyncTimeout != nil {
		opts.CacheSyncTimeout = cfg.CacheSyncTimeout.Duration
	}
	return opts
}

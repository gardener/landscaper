// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package continuousreconcile

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/robfig/cron/v3"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	cr "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile"
	"github.com/gardener/landscaper/pkg/deployer/lib/extension"
)

// ContinuousReconcileActiveAnnotation can be used to deactivate continuous reconciliation on a deploy item without changing its spec.
// Setting it to "false" will suppress continuous reconciliation, even if it is configured for the deploy item otherwise.
// Setting it to any other value has no effect.
const ContinuousReconcileActiveAnnotation = "continuousreconcile.extensions.landscaper.gardener.cloud/active"

// ContinuousReconcileExtension returns an extension hook function which can handle continuous reconciliation.
// It is meant to be used as a ShouldReconcile hook and might yield unexpected results if used with another hook handle.
// The function which it takes as an argument is expected to take a time and return the time when the deploy item should be scheduled for the next automatic reconciliation.
//   It should return nil if continuous reconciliation is not configured for the deploy item.
// The returned function will panic if the provided deploy item is nil.
func ContinuousReconcileExtension(nextReconcile func(context.Context, time.Time, *lsv1alpha1.DeployItem) (*time.Time, error)) extension.ReconcileExtensionHook {
	return func(ctx context.Context, log logr.Logger, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target, hype extension.HookType) (*extension.HookResult, error) {
		logger := log.WithName("continuousReconcileExtension")
		logger.V(7).Info("execute")
		if di == nil {
			panic("deploy item must not be nil")
		}

		// check for annotation
		if active, ok := di.Annotations[ContinuousReconcileActiveAnnotation]; ok && active == "false" {
			logger.V(5).Info("continuous reconciliation disabled by annotation", "annotation", ContinuousReconcileActiveAnnotation)
			return nil, nil
		}

		nextRaw, err := nextReconcile(ctx, di.Status.LastReconcileTime.Time, di)
		if err != nil {
			return nil, fmt.Errorf("unable to check whether reconciliation is due: %w", err)
		}
		if nextRaw == nil {
			logger.V(7).Info("no continuous reconcile specified")
			return nil, nil
		}
		next := nextRaw.Truncate(time.Second) // when the next reconcile is scheduled to happen
		now := time.Now().Truncate(time.Second)
		res := &extension.HookResult{}

		if next.After(now) {
			// reconcile is not yet due
			// since this is a ShouldReconcile hook, we can set AbortReconcile to false and the deploy item will still be reconciled if necessary (e.g. due to spec changes)
			res.AbortReconcile = true
		} else {
			// reconcile is (over-)due
			logger.V(5).Info("reconcile deploy item")

			// compute next reconciliation time based on reconciliation which will happen now
			nextRaw, err = nextReconcile(ctx, now, di)
			if err != nil {
				return nil, fmt.Errorf("unable to compute next reconcile time: %w", err)
			}
			if nextRaw == nil {
				logger.V(7).Info("no further reconcile specified")
				// return without setting RequeueAfter
				return res, nil
			}
			next = nextRaw.Truncate(time.Second)
		}

		res.ReconcileResult.RequeueAfter = next.Sub(now)
		logger.V(7).Info("requeue deploy item", "nextReconcileTime", next.Format(time.RFC3339), "nextReconcileAfter", res.ReconcileResult.RequeueAfter.String())

		return res, nil
	}
}

// ContinuousReconcileExtensionSetup is a wrapper for ContinuousReconcileExtension.
// The return value also contains the ShouldReconcile hook type.
func ContinuousReconcileExtensionSetup(nextReconcile func(context.Context, time.Time, *lsv1alpha1.DeployItem) (*time.Time, error)) extension.ReconcileExtensionHookSetup {
	return extension.ReconcileExtensionHookSetup{
		Hook:      ContinuousReconcileExtension(nextReconcile),
		HookTypes: []extension.HookType{extension.ShouldReconcileHook},
	}
}

// Schedule returns a cron schedule based on the specification.
// If both Cron and Every are specified (which should be prevented by validation), Cron takes precedence.
// If neither is specified, an error is returned (this should also be prevented by validation).
func Schedule(crs *cr.ContinuousReconcileSpec) (cron.Schedule, error) {
	if crs == nil {
		return nil, nil
	}
	if len(crs.Cron) != 0 {
		return cron.ParseStandard(crs.Cron)
	}
	if crs.Every != nil {
		return cron.Every(crs.Every.Duration), nil
	}
	return nil, fmt.Errorf("neither Cron nor Every is specified")
}

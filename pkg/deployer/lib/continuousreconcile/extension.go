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
	duv1alpha1 "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/lib/extension"
)

// ContinuousReconcileExtension returns an extension hook function which can handle continuous reconciliation.
// It is meant to be used as a ShouldReconcile hook and might yield unexpected results if used with another hook handle.
// The function which it takes as an argument is expected to take a time and return the time when the deploy item should be scheduled for the next automatic reconciliation.
//   It should return nil if continuous reconciliation is not configured for the deploy item.
func ContinuousReconcileExtension(nextReconcile func(context.Context, time.Time, *lsv1alpha1.DeployItem) (*time.Time, error)) extension.ReconcileExtensionHook {
	return func(ctx context.Context, log logr.Logger, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target, hype extension.HookType) (*extension.HookResult, error) {
		logger := log.WithName("continuousReconcileExtension")
		logger.V(7).Info("execute")
		if di == nil {
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
		next := nextRaw.Round(time.Second)
		now := time.Now().Round(time.Second)
		res := &extension.HookResult{}
		if next.After(now) {
			// reconcile is not yet due
			// since this is a ShouldReconcile hook, we can set AbortReconcile to false and the deploy item will still be reconciled if necessary (e.g. due to spec changes)
			res.AbortReconcile = true
		} else {
			// reconcile is due
			logger.V(5).Info("reconcile deploy item")
		}
		nextReconcileTimeRaw, err := nextReconcile(ctx, now, di)
		if err != nil {
			return nil, fmt.Errorf("unable to compute next reconcile time: %w", err)
		}
		if nextReconcileTimeRaw == nil {
			logger.V(7).Info("no further reconcile specified")
			// return without setting RequeueAfter
			return res, nil
		}
		nextReconcileTime := nextReconcileTimeRaw.Round(time.Second)
		requeueAfter := nextReconcileTime.Sub(now)
		logger.V(7).Info("requeue deploy item", "nextReconcileTime", nextReconcileTime.Format(time.RFC3339), "nextReconcileAfter", requeueAfter.String())
		res.ReconcileResult.RequeueAfter = requeueAfter

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
func Schedule(crs *duv1alpha1.ContinuousReconcileSpec) (cron.Schedule, error) {
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

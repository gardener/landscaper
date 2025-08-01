// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package extension

import (
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// HookResult represents the result of a reconciliation extension hook.
type HookResult struct {
	// ReconcileResult will be returned by the reconcile function if no error occurs.
	ReconcileResult reconcile.Result

	// If set to true, reconciliation will be aborted with returning ReconcileResult after the current execution.
	AbortReconcile bool
}

// AggregateHookResults aggregates multiple hook results into a single one.
// - If all hook results are nil, nil will be returned.
// - If there is exactly one non-nil hook result, a copy of it will be returned.
// - Otherwise the non-nil hook results will be aggregated as follows:
//   - AbortReconcile and ReconcileResult.Requeue are ORed, so if one of them is true for any of the hook results,
//     it will be true in the return value.
//   - ReconcileResult.RequeueAfter is aggregated using a minimum function, so the return value's field will be set
//     to the smallest value greater than zero that was set among the given hook results.
//   - If ReconcileResult.Requeue is true, RequeueAfter will be set to zero to ensure an immediate reconcile.
func AggregateHookResults(hrs ...*HookResult) *HookResult {
	return aggregateHookResults(func(a, b bool) bool { return a || b }, hrs...)
}

// AggregateHookResultsWithInvertedAbortPriority works exactly like AggregateHookResults with the exception that
// the AbortReconcile values are ANDed instead of ORed.
// This is required for a few special cases where setting the value to 'false' is meant to enforce continuation.
func AggregateHookResultsWithInvertedAbortPriority(hrs ...*HookResult) *HookResult {
	return aggregateHookResults(func(a, b bool) bool { return a && b }, hrs...)
}

func aggregateHookResults(aggregate func(bool, bool) bool, hrs ...*HookResult) *HookResult {
	var res *HookResult
	for _, hr := range hrs {
		if hr == nil {
			continue
		}
		if res == nil {
			res = hr.DeepCopy()
			continue
		}
		res.AbortReconcile = aggregate(res.AbortReconcile, hr.AbortReconcile)
		if hr.ReconcileResult.RequeueAfter > 0 && (res.ReconcileResult.RequeueAfter == 0 || hr.ReconcileResult.RequeueAfter < res.ReconcileResult.RequeueAfter) {
			res.ReconcileResult.RequeueAfter = hr.ReconcileResult.RequeueAfter
		}
	}
	return res
}

func (hr *HookResult) DeepCopy() *HookResult {
	return &HookResult{
		ReconcileResult: reconcile.Result{
			RequeueAfter: hr.ReconcileResult.RequeueAfter,
		},
		AbortReconcile: hr.AbortReconcile,
	}
}

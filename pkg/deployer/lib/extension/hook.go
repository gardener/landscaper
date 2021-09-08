// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package extension

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

type HookType string

const (
	// Called at the beginning of the reconciliation.
	// Because this hook is called before the deploy item is fetched from the cluster, deploy item and target will always be nil in the hook function call.
	StartHook HookType = "Start"

	// Called after the responsibility has been checked, but before it is evaluated.
	// Beware: a non-nil hook result will override the result of the responsibility check - if something is returned, only HookResult.AbortReconcile decides whether to continue or abor the reconciliation.
	DuringResponsibilityCheckHook HookType = "DuringResponsibilityCheck" // called after the responsibility has been checked but before the reconcile is aborted, in case the deployer is not responsible

	// Called after it has been determined that the deployer is responsible for the deploy item.
	AfterResponsibilityCheckHook HookType = "AfterResponsibilityCheck"

	// Called while it is evaluated whether a reconcile is necessary.
	// Beware: a non-nil hook result with HookResult.AbortReconcile set to 'false' will enforce a full reconcile, even if it is not required by the default logic.
	//   However, a result with AbortReconcile set to 'true' will not abort the reconcile, if it is required by the default logic.
	//   The reason for this is that by the time the hook is called, the deploy item has already been altered (phase, lastReconcileTime, ...) and aborting the reconciliation would lead to an inconsistent state.
	ShouldReconcileHook HookType = "ShouldReconcile"

	// Called if the deploy item is to be aborted due to the abort operation annotation.
	// Beware: Aborting the reconciliation at this point might leave the deploy item in an inconsistent state and is discouraged.
	BeforeAbortHook HookType = "BeforeAbort"

	// Called if the deploy item is to be force-reconciled due to the force reconcile operation annotation.
	// Beware: Aborting the reconciliation at this point might leave the deploy item in an inconsistent state and is discouraged.
	BeforeForceReconcileHook HookType = "BeforeForceReconcile"

	// Called if the deploy item is to be deleted.
	// Beware: Aborting the reconciliation at this point might leave the deploy item in an inconsistent state and is discouraged.
	BeforeDeleteHook HookType = "BeforeDelete"

	// Called if a "normal" reconcile of the deploy item is about to happen.
	// This is the case if a reconcile is required and the deploy item is not aborted, force-reconciled, or deleted.
	// Beware: Aborting the reconciliation at this point might leave the deploy item in an inconsistent state and is discouraged.
	BeforeReconcileHook HookType = "BeforeReconcile"

	// Called before any of abortion, deletion, and (force-)reconciliation.
	// This will always be called if the reconciliation has not been aborted before or during the ShouldReconcile check.
	// Beware: Aborting the reconciliation at this point might leave the deploy item in an inconsistent state and is discouraged.
	BeforeAnyReconcileHook HookType = "BeforeAnyReconcile"

	// Called after a successful reconciliation.
	// This will always be called at the end of reconciliation, unless an error has occurred or the reconciliation has been aborted
	//   during the responsibility or ShouldReconcile check or by one of the earlier hooks.
	EndHook HookType = "End"
)

// ReconcileExtensionHook represents a function which will be called when the hook is executed.
type ReconcileExtensionHook func(context.Context, logr.Logger, *lsv1alpha1.DeployItem, *lsv1alpha1.Target, HookType) (*HookResult, error)

// ReconcileExtensionHooks maps hook types to a list of hook functions.
type ReconcileExtensionHooks map[HookType][]ReconcileExtensionHook

// ReconcileExtensionHookSetup can be used to couple a hook function with the hooks it is meant for.
type ReconcileExtensionHookSetup struct {
	Hook      ReconcileExtensionHook
	HookTypes []HookType
}

// ExecuteHooks executes all hooks of a given type in the order they are specified in.
func (hooks ReconcileExtensionHooks) ExecuteHooks(ctx context.Context, log logr.Logger, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target, hype HookType) (*HookResult, error) {
	logger := log.WithName(string(hype))
	logger.V(7).Info("calling extension hooks")
	typedHooks, ok := hooks[hype]
	if !ok {
		switch hype {
		case StartHook, DuringResponsibilityCheckHook, AfterResponsibilityCheckHook,
			ShouldReconcileHook, BeforeAbortHook, BeforeForceReconcileHook,
			BeforeDeleteHook, BeforeReconcileHook, BeforeAnyReconcileHook, EndHook:
			return nil, nil
		default:
			return nil, fmt.Errorf("unknown hook type %q", string(hype))
		}
	}
	hookRes := make([]*HookResult, len(typedHooks))
	for i, hook := range typedHooks {
		logger.WithValues("index", i).V(5).Info("calling extension hook")
		var err error
		hookRes[i], err = hook(ctx, logger, di, target, hype)
		if err != nil {
			return nil, fmt.Errorf("error executing reconciliation extension hook %d of type %q: %w", i, string(hype), err)
		}
	}
	return AggregateHookResults(hookRes...), nil
}

// RegisterHook appends the given hook function to the list of hook functions for all given hook types.
// It returns the ReconcileExtensionHooks object it is called on for chaining.
func (hooks ReconcileExtensionHooks) RegisterHook(hook ReconcileExtensionHook, hypes ...HookType) ReconcileExtensionHooks {
	for _, hype := range hypes {
		hookList, ok := hooks[hype]
		if !ok {
			hookList = []ReconcileExtensionHook{}
		}
		hooks[hype] = append(hookList, hook)
	}
	return hooks
}

// RegisterHookSetup is a wrapper for RegisterHook which uses a ReconcileExtensionHookSetup object instead of a hook function and types.
// It returns the ReconcileExtensionHooks object it is called on for chaining.
func (hooks ReconcileExtensionHooks) RegisterHookSetup(hookSetup ReconcileExtensionHookSetup) ReconcileExtensionHooks {
	return hooks.RegisterHook(hookSetup.Hook, hookSetup.HookTypes...)
}

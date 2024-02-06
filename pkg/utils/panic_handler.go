// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

func HandlePanics(ctx context.Context, result *reconcile.Result) {
	logger, _ := logging.FromContextOrNew(ctx, nil)

	if r := recover(); r != nil {
		result.Requeue = true
		result.RequeueAfter = time.Minute * 5

		debug.PrintStack()

		if err, ok := r.(runtime.Error); ok {
			logger.Error(err, "observed a panic", "recoverResult", r,
				constants.KeyString, string(debug.Stack()))

			if err.Error() == "runtime error: invalid memory address or nil pointer dereference" {
				logger.Error(err, "Recovered from a nil pointer dereference or invalid memory address error")
				return
			}

			if strings.HasPrefix(err.Error(), "runtime error: index out of range") {
				logger.Error(err, "Recovered from an index out of range error")
				return
			}

			if strings.HasPrefix(err.Error(), "runtime error: integer divide by zero") {
				logger.Error(err, "Recovered from a integer divide by zero error")
				return
			}

			if strings.HasPrefix(err.Error(), "interface conversion:") {
				logger.Error(err, "Recovered from a type assertion error")
				return
			}

		} else if err2, ok := r.(error); ok {
			logger.Error(err2, "observed a panic", "recoverResult", r, constants.KeyString, string(debug.Stack()))
		} else {
			logger.Error(nil, "observed a panic", "recoverResult", r, constants.KeyString, string(debug.Stack()))
		}

		panic(r)
	}
}

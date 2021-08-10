// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"context"

	"github.com/gardener/landscaper/test/utils/envtest"
)

type CleanupFunc func(ctx context.Context) error

// State wraps the envtest.State with framework related functionality.
type State struct {
	*envtest.State
	dumper  *Dumper
	cleanup CleanupFunc
}

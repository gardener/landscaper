// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"context"

	"github.com/gardener/component-spec/bindings-go/ctf"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

type RegistryAccess interface {
	GetComponentVersion(ctx context.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (ComponentVersion, error)

	// temporary
	GetComponentResolver() ctf.ComponentResolver
}

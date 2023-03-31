// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"context"
)

type ComponentVersion interface {
	GetName() string
	GetVersion() string
	GetRepositoryContext() []byte
	GetDescriptor(ctx context.Context) ([]byte, error)
	GetDependency(ctx context.Context, name string) (ComponentVersion, error)
	GetResource(name string, identity map[string]string) (Resource, error)
}

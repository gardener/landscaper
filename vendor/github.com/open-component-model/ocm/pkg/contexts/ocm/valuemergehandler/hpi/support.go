// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package hpi

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/valuemergehandler/internal"
)

type EmptyConfig struct{}

var _ Config = (*EmptyConfig)(nil)

func (c *EmptyConfig) Complete(ctx Context) error {
	return nil
}

type Merger[C, T any] func(ctx Context, cfg C, local T, target *T) (bool, error)

func New[C any, L any, P internal.ConfigPointer[C]](algo string, desc string, merger internal.Merger[P, L]) Handler {
	return internal.New[C, L, P](algo, desc, merger)
}

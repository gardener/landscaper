// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// resolve package cycle among default merge handler and
// labelmergehandler by separating commonly used types
// into this package

// same problem for the embedding into the OCM environment
// required for the ocm.Context access.
// Because of this cycle, the registry implementation and the
// required types have to be placed into the internal package of
// ocm and forwarded to the cpi packages. From there it can be consumed
// here to break the dependency cycle.

type (
	Context       = cpi.Context
	Handler       = cpi.ValueMergeHandler
	Config        = cpi.ValueMergeHandlerConfig
	Registry      = cpi.ValueMergeHandlerRegistry
	Specification = metav1.MergeAlgorithmSpecification
	Value         = runtime.RawValue
)

func Register(h Handler) {
	cpi.RegisterValueMergeHandler(h)
}

func Assign(hint string, spec *Specification) {
	cpi.AssignValueMergeHandler(hint, spec)
}

func NewRegistry(base ...Registry) Registry {
	return cpi.NewValueMergeHandlerRegistry(base...)
}

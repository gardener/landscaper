// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"github.com/open-component-model/ocm/pkg/contexts/oci"
)

//
// resolve cyclic package dependency between genericocireg and core

type OCISpecFunction func(ctx oci.Context) (RepositoryType, error)

var ociimpl OCISpecFunction

func RegisterOCIImplementation(impl OCISpecFunction) {
	if ociimpl != nil {
		panic("oci implementation already registered")
	}
	ociimpl = impl
}

// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package oci_repository_prepare

import (
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/action/handlers"
	"github.com/open-component-model/ocm/pkg/generics"
)

func Execute(hdlrs handlers.Registry, host, repo string, creds common.Properties) (*ActionResult, error) {
	return generics.AsE[*ActionResult](hdlrs.Execute(Spec(host, repo), creds))
}

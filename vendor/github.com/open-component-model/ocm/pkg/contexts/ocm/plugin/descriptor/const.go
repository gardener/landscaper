// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package descriptor

import (
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/action"
	"github.com/open-component-model/ocm/pkg/errors"
	ocmlog "github.com/open-component-model/ocm/pkg/logging"
)

const (
	KIND_PLUGIN       = "plugin"
	KIND_DOWNLOADER   = "downloader"
	KIND_UPLOADER     = "uploader"
	KIND_ACCESSMETHOD = errors.KIND_ACCESSMETHOD
	KIND_ACTION       = action.KIND_ACTION
	KIND_VALUESET     = "value set"
	KIND_PURPOSE      = "purposet"
)

var REALM = ocmlog.DefineSubRealm("OCM plugin handling", "plugins")

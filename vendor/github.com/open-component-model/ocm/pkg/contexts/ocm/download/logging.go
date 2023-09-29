// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package download

import (
	"github.com/mandelsoft/logging"

	ocmlog "github.com/open-component-model/ocm/pkg/logging"
)

var REALM = ocmlog.DefineSubRealm("Downloaders", "downloader")

func Logger(ctx logging.ContextProvider, messageContext ...logging.MessageContext) logging.Logger {
	return ctx.LoggingContext().Logger(append([]logging.MessageContext{REALM}, messageContext...))
}

// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package datacontext

import (
	ocmlog "github.com/open-component-model/ocm/pkg/logging"
)

var Realm = ocmlog.DefineSubRealm("context lifecycle", "context")

var Logger = ocmlog.DynamicLogger(Realm)

func Debug(c Context, msg string, keypairs ...interface{}) {
	c.LoggingContext().Logger(Realm).Debug(msg, append(keypairs, "id", c.GetId())...)
}

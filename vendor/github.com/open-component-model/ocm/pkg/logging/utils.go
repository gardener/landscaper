// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"path"

	"github.com/mandelsoft/logging"
)

func ErrorMessage(err error) *string {
	if err == nil {
		return nil
	}
	m := err.Error()
	return &m
}

func SubRealm(names ...string) logging.Realm {
	return logging.NewRealm(path.Join(REALM.Name(), path.Join(names...)))
}

func DefineSubRealm(desc string, names ...string) logging.Realm {
	return logging.DefineRealm(path.Join(REALM.Name(), path.Join(names...)), desc)
}

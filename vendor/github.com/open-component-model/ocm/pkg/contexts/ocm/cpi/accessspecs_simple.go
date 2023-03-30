// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"strings"

	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/internal"
	"github.com/open-component-model/ocm/pkg/runtime"
)

type accessTypeTarget struct {
	*accessType
}

func (t accessTypeTarget) SetDescription(value string) {
	t.description = value
}

func (t accessTypeTarget) SetFormat(value string) {
	t.format = value
}

func (t accessTypeTarget) SetConfigHandler(value flagsets.ConfigOptionTypeSetHandler) {
	t.handler = value
}

////////////////////////////////////////////////////////////////////////////////

type accessType struct {
	runtime.ObjectVersionedType
	runtime.TypedObjectDecoder
	description string
	format      string
	handler     flagsets.ConfigOptionTypeSetHandler
}

func NewAccessSpecType(name string, proto internal.AccessSpec, opts ...AccessSpecTypeOption) AccessType {
	t := accessTypeTarget{&accessType{
		ObjectVersionedType: runtime.NewVersionedObjectType(name),
		TypedObjectDecoder:  runtime.MustNewDirectDecoder(proto),
	}}
	for _, o := range opts {
		o.ApplyToAccessSpecOptionTarget(t)
	}
	return t.accessType
}

func (t *accessType) ConfigOptionTypeSetHandler() flagsets.ConfigOptionTypeSetHandler {
	return t.handler
}

func (t *accessType) Description() string {
	return t.description
}

func (t *accessType) Format(cli bool) string {
	group := ""
	if t.handler != nil && cli {
		opts := t.handler.OptionTypeNames()
		var names []string
		if len(opts) > 0 {
			for _, o := range opts {
				names = append(names, "<code>--"+o+"</code>")
			}
			group = "\nOptions used to configure fields: " + strings.Join(names, ", ")
		}
	}
	return t.format + group
}

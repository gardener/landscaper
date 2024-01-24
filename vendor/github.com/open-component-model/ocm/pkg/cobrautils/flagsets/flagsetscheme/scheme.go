// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package flagsetscheme

import (
	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets"
	"github.com/open-component-model/ocm/pkg/logging"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/runtime/descriptivetype"
	"github.com/open-component-model/ocm/pkg/utils"
)

// VersionTypedObjectType is the appropriately extended type interface
// based on runtime.VersionTypedObjectType.
type VersionTypedObjectType[T runtime.VersionedTypedObject] interface {
	descriptivetype.TypedObjectType[T]

	ConfigOptionTypeSetHandler() flagsets.ConfigOptionTypeSetHandler
}

////////////////////////////////////////////////////////////////////////////////

// TypeScheme is the appropriately extended scheme interface based on
// runtime.TypeScheme.
type TypeScheme[T runtime.VersionedTypedObject, R VersionTypedObjectType[T]] interface {
	descriptivetype.TypeScheme[T, R]

	CreateConfigTypeSetConfigProvider() flagsets.ConfigTypeOptionSetConfigProvider
}

type _typeScheme[T runtime.VersionedTypedObject, R VersionTypedObjectType[T]] interface {
	descriptivetype.TypeScheme[T, R]
}

type typeScheme[T runtime.VersionedTypedObject, R VersionTypedObjectType[T], S TypeScheme[T, R]] struct {
	cfgname     string
	description string
	group       string
	typeOption  string
	_typeScheme[T, R]
}

func flagExtender[T runtime.VersionedTypedObject, R VersionTypedObjectType[T]](ty R) string {
	if h := ty.ConfigOptionTypeSetHandler(); h != nil {
		return utils.IndentLines(flagsets.FormatConfigOptions(h), "  ")
	}
	return ""
}

// NewTypeScheme provides an TypeScheme implementation based on the interfaces
// and the default runtime.TypeScheme implementation.
func NewTypeScheme[T runtime.VersionedTypedObject, R VersionTypedObjectType[T], S TypeScheme[T, R]](kindname string, cfgname, typeOption, desc, group string, unknown runtime.Unstructured, acceptUnknown bool, base ...S) TypeScheme[T, R] {
	scheme := descriptivetype.NewTypeScheme[T, R](kindname, flagExtender[T, R], unknown, acceptUnknown, utils.Optional(base...))
	return &typeScheme[T, R, S]{
		cfgname:     cfgname,
		description: desc,
		group:       group,
		typeOption:  typeOption,
		_typeScheme: scheme,
	}
}

func (t *typeScheme[T, R, S]) CreateConfigTypeSetConfigProvider() flagsets.ConfigTypeOptionSetConfigProvider {
	var prov flagsets.ConfigTypeOptionSetConfigProvider
	if t.typeOption == "" {
		prov = flagsets.NewExplicitlyTypedConfigProvider(t.cfgname, t.description, true)
	} else {
		prov = flagsets.NewTypedConfigProvider(t.cfgname, t.description, t.typeOption, true)
	}
	if t.group != "" {
		prov.AddGroups(t.group)
	}
	for _, p := range t.KnownTypes() {
		err := prov.AddTypeSet(p.ConfigOptionTypeSetHandler())
		if err != nil {
			logging.Logger().LogError(err, "cannot compose type CLI options", "type", t.cfgname)
		}
	}
	if t.BaseScheme() != nil {
		base := t.BaseScheme()
		for _, s := range base.(S).CreateConfigTypeSetConfigProvider().OptionTypeSets() {
			if prov.GetTypeSet(s.GetName()) == nil {
				err := prov.AddTypeSet(s)
				if err != nil {
					logging.Logger().LogError(err, "cannot compose type CLI options", "type", t.cfgname)
				}
			}
		}
	}

	return prov
}

func (t *typeScheme[T, R, S]) KnownTypes() runtime.KnownTypes[T, R] {
	return t._typeScheme.KnownTypes() // Goland
}

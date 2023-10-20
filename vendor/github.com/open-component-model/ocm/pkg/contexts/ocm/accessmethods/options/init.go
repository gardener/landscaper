// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package options

const (
	TYPE_STRING        = "string"
	TYPE_STRINGARRAY   = "[]string"
	TYPE_STRING2STRING = "string=string"
	TYPE_INT           = "int"
	TYPE_BOOL          = "bool"
	TYPE_YAML          = "YAML"
	TYPE_STRINGMAPYAML = "map[string]YAML"
	TYPE_STRING2YAML   = "string=YAML"
)

func init() {
	DefaultRegistry.RegisterValueType(TYPE_STRING, NewStringOptionType, "string value")
	DefaultRegistry.RegisterValueType(TYPE_STRINGARRAY, NewStringArrayOptionType, "list of string values")
	DefaultRegistry.RegisterValueType(TYPE_STRING2STRING, NewStringMapOptionType, "string map defined by dedicated assignments")
	DefaultRegistry.RegisterValueType(TYPE_INT, NewIntOptionType, "integer value")
	DefaultRegistry.RegisterValueType(TYPE_BOOL, NewBoolOptionType, "boolean flag")
	DefaultRegistry.RegisterValueType(TYPE_YAML, NewYAMLOptionType, "JSON or YAML document string")
	DefaultRegistry.RegisterValueType(TYPE_STRINGMAPYAML, NewValueMapYAMLOptionType, "JSON or YAML map")
	DefaultRegistry.RegisterValueType(TYPE_STRING2YAML, NewValueMapOptionType, "string map with arbitrary values defined by dedicated assignments")
}

func RegisterOption(o OptionType) OptionType {
	DefaultRegistry.RegisterOptionType(o)
	return o
}

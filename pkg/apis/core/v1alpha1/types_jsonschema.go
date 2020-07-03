// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// most of this JSONSchemaProps definitions are base on https://github.com/kubernetes/apiextensions-apiserver/blob/88daf26ec3b8305c61d0a05cce6e1de27aa6e716/pkg/apis/apiextensions/v1/marshal.go
// but adjusted to our needs

package v1alpha1

import (
	"encoding/json"
	"errors"
)

// JSONSchemaProps is a JSON-Schema following Specification Draft 4 (http://json-schema.org/).
// Inspired by https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/apis/apiextensions/v1/types_jsonschema.go
type JSONSchemaProps struct {
	ID          string        `json:"id,omitempty" protobuf:"bytes,1,opt,name=id"`
	Schema      JSONSchemaURL `json:"$schema,omitempty" protobuf:"bytes,2,opt,name=schema"`
	Ref         *string       `json:"$ref,omitempty" protobuf:"bytes,3,opt,name=ref"`
	Description string        `json:"description,omitempty" protobuf:"bytes,4,opt,name=description"`
	Type        string        `json:"type,omitempty" protobuf:"bytes,5,opt,name=type"`

	// format is an OpenAPI v3 format string. Unknown formats are ignored. The following formats are validated:
	//
	// - bsonobjectid: a bson object ID, i.e. a 24 characters hex string
	// - uri: an URI as parsed by Golang net/url.ParseRequestURI
	// - email: an email address as parsed by Golang net/mail.ParseAddress
	// - hostname: a valid representation for an Internet host name, as defined by RFC 1034, section 3.1 [RFC1034].
	// - ipv4: an IPv4 IP as parsed by Golang net.ParseIP
	// - ipv6: an IPv6 IP as parsed by Golang net.ParseIP
	// - cidr: a CIDR as parsed by Golang net.ParseCIDR
	// - mac: a MAC address as parsed by Golang net.ParseMAC
	// - uuid: an UUID that allows uppercase defined by the regex (?i)^[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}$
	// - uuid3: an UUID3 that allows uppercase defined by the regex (?i)^[0-9a-f]{8}-?[0-9a-f]{4}-?3[0-9a-f]{3}-?[0-9a-f]{4}-?[0-9a-f]{12}$
	// - uuid4: an UUID4 that allows uppercase defined by the regex (?i)^[0-9a-f]{8}-?[0-9a-f]{4}-?4[0-9a-f]{3}-?[89ab][0-9a-f]{3}-?[0-9a-f]{12}$
	// - uuid5: an UUID5 that allows uppercase defined by the regex (?i)^[0-9a-f]{8}-?[0-9a-f]{4}-?5[0-9a-f]{3}-?[89ab][0-9a-f]{3}-?[0-9a-f]{12}$
	// - isbn: an ISBN10 or ISBN13 number string like "0321751043" or "978-0321751041"
	// - isbn10: an ISBN10 number string like "0321751043"
	// - isbn13: an ISBN13 number string like "978-0321751041"
	// - creditcard: a credit card number defined by the regex ^(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|6(?:011|5[0-9][0-9])[0-9]{12}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|(?:2131|1800|35\\d{3})\\d{11})$ with any non digit characters mixed in
	// - ssn: a U.S. social security number following the regex ^\\d{3}[- ]?\\d{2}[- ]?\\d{4}$
	// - hexcolor: an hexadecimal color code like "#FFFFFF: following the regex ^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$
	// - rgbcolor: an RGB color code like rgb like "rgb(255,255,2559"
	// - byte: base64 encoded binary data
	// - password: any kind of string
	// - date: a date string like "2006-01-02" as defined by full-date in RFC3339
	// - duration: a duration string like "22 ns" as parsed by Golang time.ParseDuration or compatible with Scala duration format
	// - datetime: a date time string like "2014-12-15T19:30:20.000Z" as defined by date-time in RFC3339.
	Format string `json:"format,omitempty" protobuf:"bytes,6,opt,name=format"`

	Title string `json:"title,omitempty" protobuf:"bytes,7,opt,name=title"`
	// default is a default value for undefined object fields.
	// Defaulting is a beta feature under the CustomResourceDefaulting feature gate.
	// Defaulting requires spec.preserveUnknownFields to be false.
	Default          *JSON                   `json:"default,omitempty" protobuf:"bytes,8,opt,name=default"`
	Maximum          *Decimal                `json:"maximum,omitempty" protobuf:"bytes,9,opt,name=maximum"`
	ExclusiveMaximum bool                    `json:"exclusiveMaximum,omitempty" protobuf:"bytes,10,opt,name=exclusiveMaximum"`
	Minimum          *Decimal                `json:"minimum,omitempty" protobuf:"bytes,11,opt,name=minimum"`
	ExclusiveMinimum bool                    `json:"exclusiveMinimum,omitempty" protobuf:"bytes,12,opt,name=exclusiveMinimum"`
	MaxLength        *int64                  `json:"maxLength,omitempty" protobuf:"bytes,13,opt,name=maxLength"`
	MinLength        *int64                  `json:"minLength,omitempty" protobuf:"bytes,14,opt,name=minLength"`
	Pattern          string                  `json:"pattern,omitempty" protobuf:"bytes,15,opt,name=pattern"`
	MaxItems         *int64                  `json:"maxItems,omitempty" protobuf:"bytes,16,opt,name=maxItems"`
	MinItems         *int64                  `json:"minItems,omitempty" protobuf:"bytes,17,opt,name=minItems"`
	UniqueItems      bool                    `json:"uniqueItems,omitempty" protobuf:"bytes,18,opt,name=uniqueItems"`
	MultipleOf       *Decimal                `json:"multipleOf,omitempty" protobuf:"bytes,19,opt,name=multipleOf"`
	Enum             []JSON                  `json:"enum,omitempty" protobuf:"bytes,20,rep,name=enum"`
	MaxProperties    *int64                  `json:"maxProperties,omitempty" protobuf:"bytes,21,opt,name=maxProperties"`
	MinProperties    *int64                  `json:"minProperties,omitempty" protobuf:"bytes,22,opt,name=minProperties"`
	Required         []string                `json:"required,omitempty" protobuf:"bytes,23,rep,name=required"`
	Items            *JSONSchemaPropsOrArray `json:"items,omitempty" protobuf:"bytes,24,opt,name=items"`
	AllOf            []JSONSchemaProps       `json:"allOf,omitempty" protobuf:"bytes,25,rep,name=allOf"`
	OneOf            []JSONSchemaProps       `json:"oneOf,omitempty" protobuf:"bytes,26,rep,name=oneOf"`
	AnyOf            []JSONSchemaProps       `json:"anyOf,omitempty" protobuf:"bytes,27,rep,name=anyOf"`

	// +kubebuilder:validation:XPreserveUnknownFields
	Not                  *JSONSchemaProps           `json:"not,omitempty" protobuf:"bytes,28,opt,name=not"`
	Properties           map[string]JSONSchemaProps `json:"properties,omitempty" protobuf:"bytes,29,rep,name=properties"`
	AdditionalProperties *JSONSchemaPropsOrBool     `json:"additionalProperties,omitempty" protobuf:"bytes,30,opt,name=additionalProperties"`
	PatternProperties    map[string]JSONSchemaProps `json:"patternProperties,omitempty" protobuf:"bytes,31,rep,name=patternProperties"`
	Dependencies         JSONSchemaDependencies     `json:"dependencies,omitempty" protobuf:"bytes,32,opt,name=dependencies"`
	AdditionalItems      *JSONSchemaPropsOrBool     `json:"additionalItems,omitempty" protobuf:"bytes,33,opt,name=additionalItems"`
	Definitions          JSONSchemaDefinitions      `json:"definitions,omitempty" protobuf:"bytes,34,opt,name=definitions"`
	ExternalDocs         *ExternalDocumentation     `json:"externalDocs,omitempty" protobuf:"bytes,35,opt,name=externalDocs"`
	Example              *JSON                      `json:"example,omitempty" protobuf:"bytes,36,opt,name=example"`
	Nullable             bool                       `json:"nullable,omitempty" protobuf:"bytes,37,opt,name=nullable"`
}

// Decimal is a floating point number represented as string.
// e.g. 1.2
type Decimal string

// JSONSchemaURL represents a scheme url
type JSONSchemaURL string

// JSON represents any valid JSON value.
// These types are supported: bool, int64, float64, string, []interface{}, map[string]interface{} and nil.
type JSON struct {
	Raw []byte `protobuf:"bytes,1,opt,name=raw"`
}

// OpenAPISchemaType is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
//
// See: https://github.com/kubernetes/kube-openapi/tree/master/pkg/generators
func (_ JSON) OpenAPISchemaType() []string {
	// TODO: return actual types when anyOf is supported
	return nil
}

// OpenAPISchemaFormat is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
func (_ JSON) OpenAPISchemaFormat() string { return "" }

// MarshalJSON implements the json marshaling for a JSON
func (s JSON) MarshalJSON() ([]byte, error) {
	if len(s.Raw) > 0 {
		return s.Raw, nil
	}
	return []byte("null"), nil
}

// UnmarshalJSON implements json unmarshaling for a JSON
func (s *JSON) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && string(data) != "null" {
		s.Raw = data
	}
	return nil
}

// JSONSchemaPropsOrArray represents a value that can either be a JSONSchemaProps
// or an array of JSONSchemaProps. Mainly here for serialization purposes.
type JSONSchemaPropsOrArray struct {
	Schema      *JSONSchemaProps  `protobuf:"bytes,1,opt,name=schema"`
	JSONSchemas []JSONSchemaProps `protobuf:"bytes,2,rep,name=jSONSchemas"`
}

// OpenAPISchemaType is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
//
// See: https://github.com/kubernetes/kube-openapi/tree/master/pkg/generators
func (_ JSONSchemaPropsOrArray) OpenAPISchemaType() []string {
	// TODO: return actual types when anyOf is supported
	return nil
}

// OpenAPISchemaFormat is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
func (_ JSONSchemaPropsOrArray) OpenAPISchemaFormat() string { return "" }

// MarshalJSON implements the json marshaling for a JSONSchemaPropsOrArray
func (s JSONSchemaPropsOrArray) MarshalJSON() ([]byte, error) {
	if len(s.JSONSchemas) > 0 {
		return json.Marshal(s.JSONSchemas)
	}
	return json.Marshal(s.Schema)
}

// UnmarshalJSON implements json unmarshaling for a JSONSchemaPropsOrArray
func (s *JSONSchemaPropsOrArray) UnmarshalJSON(data []byte) error {
	var nw JSONSchemaPropsOrArray
	var first byte
	if len(data) > 1 {
		first = data[0]
	}
	if first == '{' {
		var sch JSONSchemaProps
		if err := json.Unmarshal(data, &sch); err != nil {
			return err
		}
		nw.Schema = &sch
	}
	if first == '[' {
		if err := json.Unmarshal(data, &nw.JSONSchemas); err != nil {
			return err
		}
	}
	*s = nw
	return nil
}

// JSONSchemaPropsOrBool represents JSONSchemaProps or a boolean value.
// Defaults to true for the boolean property.
type JSONSchemaPropsOrBool struct {
	Allows bool             `protobuf:"varint,1,opt,name=allows"`
	Schema *JSONSchemaProps `protobuf:"bytes,2,opt,name=schema"`
}

// OpenAPISchemaType is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
//
// See: https://github.com/kubernetes/kube-openapi/tree/master/pkg/generators
func (_ JSONSchemaPropsOrBool) OpenAPISchemaType() []string {
	// TODO: return actual types when anyOf is supported
	return nil
}

// OpenAPISchemaFormat is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
func (_ JSONSchemaPropsOrBool) OpenAPISchemaFormat() string { return "" }

// MarshalJSON implements the json marshaling for a JSONSchemaPropsOrBool
func (s JSONSchemaPropsOrBool) MarshalJSON() ([]byte, error) {
	if s.Schema != nil {
		return json.Marshal(s.Schema)
	}

	if s.Schema == nil && !s.Allows {
		return []byte("false"), nil
	}
	return []byte("true"), nil
}

// UnmarshalJSON implements json unmarshaling for a JSONSchemaPropsOrBool
func (s *JSONSchemaPropsOrBool) UnmarshalJSON(data []byte) error {
	var nw JSONSchemaPropsOrBool
	switch {
	case len(data) == 0:
	case data[0] == '{':
		var sch JSONSchemaProps
		if err := json.Unmarshal(data, &sch); err != nil {
			return err
		}
		nw.Allows = true
		nw.Schema = &sch
	case len(data) == 4 && string(data) == "true":
		nw.Allows = true
	case len(data) == 5 && string(data) == "false":
		nw.Allows = false
	default:
		return errors.New("boolean or JSON schema expected")
	}
	*s = nw
	return nil
}

// JSONSchemaDependencies represent a dependencies property.
type JSONSchemaDependencies map[string]JSONSchemaPropsOrStringArray

// JSONSchemaPropsOrStringArray represents a JSONSchemaProps or a string array.
type JSONSchemaPropsOrStringArray struct {
	Schema   *JSONSchemaProps `protobuf:"bytes,1,opt,name=schema"`
	Property []string         `protobuf:"bytes,2,rep,name=property"`
}

// OpenAPISchemaType is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
//
// See: https://github.com/kubernetes/kube-openapi/tree/master/pkg/generators
func (_ JSONSchemaPropsOrStringArray) OpenAPISchemaType() []string {
	// TODO: return actual types when anyOf is supported
	return nil
}

// OpenAPISchemaFormat is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
func (_ JSONSchemaPropsOrStringArray) OpenAPISchemaFormat() string { return "" }

// MarshalJSON implements the json marshaling for a JSONSchemaPropsOrStringArray
func (s JSONSchemaPropsOrStringArray) MarshalJSON() ([]byte, error) {
	if len(s.Property) > 0 {
		return json.Marshal(s.Property)
	}
	if s.Schema != nil {
		return json.Marshal(s.Schema)
	}
	return []byte("null"), nil
}

// UnmarshalJSON implements json unmarshaling for a JSONSchemaPropsOrStringArray
func (s *JSONSchemaPropsOrStringArray) UnmarshalJSON(data []byte) error {
	var first byte
	if len(data) > 1 {
		first = data[0]
	}
	var nw JSONSchemaPropsOrStringArray
	if first == '{' {
		var sch JSONSchemaProps
		if err := json.Unmarshal(data, &sch); err != nil {
			return err
		}
		nw.Schema = &sch
	}
	if first == '[' {
		if err := json.Unmarshal(data, &nw.Property); err != nil {
			return err
		}
	}
	*s = nw
	return nil
}

// JSONSchemaDefinitions contains the models explicitly defined in this spec.
type JSONSchemaDefinitions map[string]JSONSchemaProps

// ExternalDocumentation allows referencing an external resource for extended documentation.
type ExternalDocumentation struct {
	Description string `json:"description,omitempty" protobuf:"bytes,1,opt,name=description"`
	URL         string `json:"url,omitempty" protobuf:"bytes,2,opt,name=url"`
}

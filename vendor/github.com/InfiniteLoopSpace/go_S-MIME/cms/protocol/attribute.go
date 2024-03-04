package protocol

import (
	"encoding/asn1"
)

// Attribute ::= SEQUENCE {
//   attrType OBJECT IDENTIFIER,
//   attrValues SET OF AttributeValue }
//
// AttributeValue ::= ANY
type Attribute struct {
	Type asn1.ObjectIdentifier

	// This should be a SET OF ANY, but Go's asn1 parser can't handle slices of
	// RawValues. Use value() to get an AnySet of the value.
	RawValue []asn1.RawValue `asn1:"set"`
}

// NewAttribute creates a single-value Attribute.
func NewAttribute(attrType asn1.ObjectIdentifier, val interface{}) (attr Attribute, err error) {
	var rv asn1.RawValue
	if rv, err = RawValue(val); err != nil {
		return
	}

	attr = Attribute{attrType, []asn1.RawValue{rv}}

	return
}

// Attributes is a common Go type for SignedAttributes and UnsignedAttributes.
//
// SignedAttributes ::= SET SIZE (1..MAX) OF Attribute
//
// UnsignedAttributes ::= SET SIZE (1..MAX) OF Attribute
type Attributes []Attribute

// GetOnlyAttributeValueBytes gets an attribute value, returning an error if the
// attribute occurs multiple times or has multiple values.
func (attrs Attributes) GetOnlyAttributeValueBytes(oid asn1.ObjectIdentifier) (rv asn1.RawValue, err error) {
	var vals [][]asn1.RawValue
	if vals, err = attrs.GetValues(oid); err != nil {
		return
	}
	if len(vals) != 1 {
		err = ASN1Error{"bad attribute count"}
		return
	}
	if len(vals[0]) != 1 {
		err = ASN1Error{"bad attribute element count"}
		return
	}

	return vals[0][0], nil
}

// GetValues retreives the attributes with the given OID. A nil value is
// returned if the OPTIONAL SET of Attributes is missing from the SignerInfo. An
// empty slice is returned if the specified attribute isn't in the set.
func (attrs Attributes) GetValues(oid asn1.ObjectIdentifier) ([][]asn1.RawValue, error) {
	if attrs == nil {
		return nil, nil
	}

	vals := [][]asn1.RawValue{}
	for _, attr := range attrs {
		if attr.Type.Equal(oid) {
			vals = append(vals, attr.RawValue)
		}
	}

	return vals, nil
}

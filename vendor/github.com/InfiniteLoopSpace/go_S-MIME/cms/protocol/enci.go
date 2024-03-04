package protocol

import (
	"encoding/asn1"

	oid "github.com/InfiniteLoopSpace/go_S-MIME/oid"
)

// EncapsulatedContentInfo ::= SEQUENCE {
//   eContentType ContentType,
//   eContent [0] EXPLICIT OCTET STRING OPTIONAL }
type EncapsulatedContentInfo struct {
	EContentType asn1.ObjectIdentifier ``                               // ContentType ::= OBJECT IDENTIFIER
	EContent     []byte                `asn1:"optional,explicit,tag:0"` //
}

// NewDataEncapsulatedContentInfo creates a new EncapsulatedContentInfo of type
// id-data.
func NewDataEncapsulatedContentInfo(data []byte) (EncapsulatedContentInfo, error) {
	return NewEncapsulatedContentInfo(oid.Data, data)
}

// NewEncapsulatedContentInfo creates a new EncapsulatedContentInfo.
func NewEncapsulatedContentInfo(contentType asn1.ObjectIdentifier, content []byte) (EncapsulatedContentInfo, error) {
	return EncapsulatedContentInfo{
		EContentType: contentType,
		EContent:     content,
	}, nil
}

// IsTypeData checks if the EContentType is id-data.
func (eci EncapsulatedContentInfo) IsTypeData() bool {
	return eci.EContentType.Equal(oid.Data)
}

package protocol

import (
	"errors"
	"fmt"
)

// ASN1Error is an error from parsing ASN.1 structures.
type ASN1Error struct {
	Message string
}

// Error implements the error interface.
func (err ASN1Error) Error() string {
	return fmt.Sprintf("cms/protocol: ASN.1 Error — %s", err.Message)
}

var (
	// ErrWrongType is returned by methods that make assumptions about types.
	// Helper methods are defined for accessing CHOICE and  ANY feilds. These
	// helper methods get the value of the field, assuming it is of a given type.
	// This error is returned if that assumption is wrong and the field has a
	// different type.
	ErrWrongType = errors.New("cms/protocol: wrong choice or any type")

	// ErrNoCertificate is returned when a requested certificate cannot be found.
	ErrNoCertificate = errors.New("no certificate found")

	// ErrNoKeyFound is returned when a requested certificate cannot be found.
	ErrNoKeyFound = errors.New("no key for decryption found")

	// ErrUnsupported is returned when an unsupported type or version
	// is encountered.
	ErrUnsupported = ASN1Error{"unsupported type or version"}

	// ErrTrailingData is returned when extra data is found after parsing an ASN.1
	// structure.
	ErrTrailingData = ASN1Error{"unexpected trailing data"}
)

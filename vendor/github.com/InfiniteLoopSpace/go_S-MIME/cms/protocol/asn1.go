package protocol

import (
	"encoding/asn1"

	asn "github.com/InfiniteLoopSpace/go_S-MIME/asn1"
)

// RawValue marshals val and returns the asn1.RawValue
func RawValue(val interface{}, params ...string) (rv asn1.RawValue, err error) {
	param := ""
	if len(params) > 0 {
		param = params[0]
	}

	var der []byte
	if der, err = asn.MarshalWithParams(val, param); err != nil {
		return
	}

	if _, err = asn.Unmarshal(der, &rv); err != nil {
		return
	}
	return
}

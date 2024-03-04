package timestamp

import (
	"encoding/asn1"
	"fmt"
	"strings"

	cms "github.com/InfiniteLoopSpace/go_S-MIME/cms/protocol"
)

// PKIStatusInfo ::= SEQUENCE {
//    status        PKIStatus,
//    statusString  PKIFreeText     OPTIONAL,
//    failInfo      PKIFailureInfo  OPTIONAL  }
type PKIStatusInfo struct {
	Status       int
	StatusString PKIFreeText    `asn1:"optional"`
	FailInfo     asn1.BitString `asn1:"optional"`
}

// PKIFreeText ::= SEQUENCE SIZE (1..MAX) OF UTF8String
type PKIFreeText []asn1.RawValue

// GetError represents an unsuccessful PKIStatusInfo as an error.
func (si PKIStatusInfo) GetError() error {
	if si.Status == 0 {
		return nil
	}
	return si
}

// Error implements the error interface.
func (si PKIStatusInfo) Error() string {
	fiStr := ""
	if si.FailInfo.BitLength > 0 {
		fibin := make([]byte, si.FailInfo.BitLength)
		for i := range fibin {
			if si.FailInfo.At(i) == 1 {
				fibin[i] = byte('1')
			} else {
				fibin[i] = byte('0')
			}
		}
		fiStr = fmt.Sprintf(" FailInfo(0b%s)", string(fibin))
	}

	statusStr := ""
	if len(si.StatusString) > 0 {
		if strs, err := si.StatusString.Strings(); err == nil {
			statusStr = fmt.Sprintf(" StatusString(%s)", strings.Join(strs, ","))
		}
	}

	return fmt.Sprintf("Bad TimeStampResp: Status(%d)%s%s", si.Status, statusStr, fiStr)
}

// Append returns a new copy of the PKIFreeText with the provided string
// appended.
func (ft PKIFreeText) Append(t string) PKIFreeText {
	return append(ft, asn1.RawValue{
		Class: asn1.ClassUniversal,
		Tag:   asn1.TagUTF8String,
		Bytes: []byte(t),
	})
}

// Strings decodes the PKIFreeText into a []string.
func (ft PKIFreeText) Strings() ([]string, error) {
	strs := make([]string, len(ft))

	for i := range ft {
		if rest, err := asn1.Unmarshal(ft[i].FullBytes, &strs[i]); err != nil {
			return nil, err
		} else if len(rest) != 0 {
			return nil, cms.ErrTrailingData
		}
	}

	return strs, nil
}

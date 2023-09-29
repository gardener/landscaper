// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/open-component-model/ocm/pkg/errors"
)

// Binary holds binary data which will be marshaled
// a base64 encoded string.
// If the string starts with a '!', the data is used as string
// byte sequence.
type Binary []byte

// MarshalJSON returns m as the JSON encoding of m.
func (m Binary) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return json.Marshal(base64.StdEncoding.EncodeToString(m))
}

// UnmarshalJSON sets *m to a copy of data.
func (m *Binary) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.Newf("runtime.Binary: UnmarshalJSON on nil pointer")
	}
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(s, "!") {
		*m, err = base64.StdEncoding.DecodeString(s)
		return err
	}
	*m = []byte(s[1:])
	return nil
}

var _ json.Marshaler = (*Binary)(nil) //nolint: gofumpt // yes
var _ json.Unmarshaler = (*Binary)(nil)

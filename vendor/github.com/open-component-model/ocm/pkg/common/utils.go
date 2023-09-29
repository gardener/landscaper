// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"reflect"
	"strings"

	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/modern-go/reflect2"
	"github.com/opencontainers/go-digest"
)

// DigestToFileName returns the file name for a digest.
func DigestToFileName(digest digest.Digest) string {
	return strings.Replace(digest.String(), ":", ".", 1)
}

// PathToDigest retuurns the digest encoded into a file name.
func PathToDigest(path string) digest.Digest {
	n := filepath.Base(path)
	idx := strings.LastIndex(n, ".")
	if idx < 0 {
		return ""
	}
	return digest.Digest(n[:idx] + ":" + n[idx+1:])
}

////////////////////////////////////////////////////////////////////////////////

func IterfaceSlice(slice interface{}) []interface{} {
	if reflect2.IsNil(slice) {
		return nil
	}
	v := reflect.ValueOf(slice)
	r := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		r[i] = v.Index(i).Interface()
	}
	return r
}

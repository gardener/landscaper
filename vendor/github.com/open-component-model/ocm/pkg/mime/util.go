// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package mime

import (
	"strings"
)

func IsJSON(mime string) bool {
	if mime == MIME_JSON || mime == MIME_JSON_ALT {
		return true
	}
	if strings.HasSuffix(mime, "+json") {
		return true
	}
	return false
}

func IsYAML(mime string) bool {
	if mime == MIME_YAML || mime == MIME_YAML_ALT {
		return true
	}
	if strings.HasSuffix(mime, "+yaml") {
		return true
	}
	return false
}

func BaseType(mime string) string {
	i := strings.Index(mime, "+")
	if i > 0 {
		return mime[:i]
	}
	return mime
}

func IsGZip(mime string) bool {
	return strings.HasSuffix(mime, "/gzip") || strings.HasSuffix(mime, "+gzip")
}

func IsMoreGeneral(m string, specific string) bool {
	if m == "" {
		return true
	}
	for {
		if m == specific {
			return true
		}
		i := strings.LastIndex(specific, "+")
		if i < 0 {
			return false
		}
		specific = specific[:i]
	}
}

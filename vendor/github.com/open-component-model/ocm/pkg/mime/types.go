// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package mime

const (
	MIME_TEXT  = "text/plain"
	MIME_OCTET = "application/octet-stream"

	MIME_JSON     = "application/x-json"
	MIME_JSON_ALT = "text/json" // no utf8
	MIME_YAML     = "application/x-yaml"
	MIME_YAML_ALT = "text/yaml" // no utf8

	MIME_GZIP    = "application/gzip"
	MIME_TAR     = "application/x-tar"
	MIME_TGZ     = "application/x-tgz"
	MIME_TGZ_ALT = MIME_TAR + "+gzip"
)

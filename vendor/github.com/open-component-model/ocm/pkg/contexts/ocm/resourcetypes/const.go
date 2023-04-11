// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resourcetypes

const (
	// OCI_ARTIFACT describes a generic OCI artifact following the
	//   [open containers image specification](https://github.com/opencontainers/image-spec/blob/main/spec.md).
	OCI_ARTIFACT = "ociArtifact"
	// OCI_IMAGE describes an OCIArtifact containing an image.
	OCI_IMAGE = "ociImage"
	// HELM_CHART describes a helm chart, either stored as OCI artifact or as tar
	// blob (tar media type).
	HELM_CHART = "helmChart"
	// BLOB describes any anonymous untyped blob data.
	BLOB = "blob"
	// FILESYSTEM describes a directory structure stored as archive (tar, tgz).
	FILESYSTEM        = "directoryTree"
	FILESYSTEM_LEGACY = "filesystem"
	// EXECUTABLE describes an OS executable.
	EXECUTABLE = "executable"
	// PLAIN_TEXT describes plain text.
	PLAIN_TEXT = "plainText"
	// OCM_PLUGIN describes an OS executable OCM plugin.
	OCM_PLUGIN = "ocmPlugin"

	// OCM_FILE describes a generic file or unspecified byte stream.
	OCM_FILE = "file"
	// OCM_YAML describes a YAML file.
	OCM_YAML = "yaml"
	// OCM_JSON describes a JSON file.
	OCM_JSON = "json"
	// OCM_XML describes a XML file.
	OCM_XML = "xml"
)

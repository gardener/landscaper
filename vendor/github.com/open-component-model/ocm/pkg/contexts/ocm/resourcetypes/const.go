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
	FILESYSTEM = "filesystem"
	// EXECUTABLE describes an OS executable.
	EXECUTABLE = "executable"
	// PLAIN_TEXT describes plain text.
	PLAIN_TEXT = "plainText"
	// OCM_PLUGIN describes an OS executable OCM plugin.
	OCM_PLUGIN = "ocmPlugin"
)

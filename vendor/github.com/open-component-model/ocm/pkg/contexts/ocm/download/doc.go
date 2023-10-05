// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

// Package download provides an API for resource download handlers.
// A download handler is used for downloading resoures. By default the native
// blob as provided by the access method is the resukt of a download.
// A download handler can influence the outbound blob format according
// to the concrete type of the resource.
// For example, a helm download for a helm artifact stored as oci artifact
// will not provide the oci format chosen for representing the artifact
// in OCI but a regular helm archive according to its specification.
// The sub package handlers provides dedicated packages for standard handlers.
//
// A downloader registry is stores as attribute ATTR_DOWNLOADER_HANDLERS
// for the OCM context, it is not a static part of the OCM context.
// The downloaders are basically used by clients requiring access
// to the effective resource content.
package download

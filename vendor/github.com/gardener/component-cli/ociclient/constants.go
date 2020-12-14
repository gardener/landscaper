// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociclient

import "k8s.io/apimachinery/pkg/util/sets"

// MediaTypeTarGzip is the media type for a gzipped tar
const MediaTypeTarGzip = "application/tar+gzip"

// MediaTypeTar is the media type for a tar
const MediaTypeTar = "application/tar"

// DefaultKnownMediaTypes contain also known media types of the oci client
var DefaultKnownMediaTypes = sets.NewString(
	MediaTypeTarGzip,
	MediaTypeTar,
)

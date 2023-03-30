// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package artdesc

import (
	"strings"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
)

func DefaultBlobDescriptor(blob accessio.BlobAccess) *Descriptor {
	return &Descriptor{
		MediaType:   blob.MimeType(),
		Digest:      blob.Digest(),
		Size:        blob.Size(),
		URLs:        nil,
		Annotations: nil,
		Platform:    nil,
	}
}

func IsDigest(version string) (bool, digest.Digest) {
	if strings.HasPrefix(version, "@") {
		return true, digest.Digest(version[1:])
	}
	if strings.Contains(version, ":") {
		return true, digest.Digest(version)
	}
	return false, ""
}

func ToContentMediaType(media string) string {
loop:
	for {
		last := strings.LastIndex(media, "+")
		if last < 0 {
			break
		}
		switch media[last+1:] {
		case "tar":
			fallthrough
		case "gzip":
			fallthrough
		case "yaml":
			fallthrough
		case "json":
			media = media[:last]
		default:
			break loop
		}
	}
	return media
}

func ToDescriptorMediaType(media string) string {
	return ToContentMediaType(media) + "+json"
}

func IsOCIMediaType(media string) bool {
	c := ToContentMediaType(media)
	for _, t := range ContentTypes() {
		if t == c {
			return true
		}
	}
	return false
}

func ContentTypes() []string {
	r := []string{}
	for _, t := range DescriptorTypes() {
		r = append(r, ToContentMediaType(t))
	}
	return r
}

func DescriptorTypes() []string {
	return []string{
		MediaTypeImageManifest,
		MediaTypeImageIndex,
		MediaTypeDockerSchema2Manifest,
		MediaTypeDockerSchema2ManifestList,
	}
}

func ArchiveBlobTypes() []string {
	r := []string{}
	for _, t := range ContentTypes() {
		t = ToContentMediaType(t)
		r = append(r, t+"+tar", t+"+tar+gzip")
	}
	return r
}

func ArtifactMimeType(cur, def string, legacy bool) string {
	if cur != "" {
		return cur
	}
	return MapArtifactMimeType(def, legacy)
}

func MapArtifactMimeType(mime string, legacy bool) string {
	if legacy {
		switch mime {
		case MediaTypeImageManifest:
			return MediaTypeDockerSchema2Manifest
		case MediaTypeImageIndex:
			return MediaTypeDockerSchema2ManifestList
		}
	} else {
		switch mime {
		case MediaTypeDockerSchema2Manifest:
			// return MediaTypeImageManifest
		case MediaTypeDockerSchema2ManifestList:
			// return MediaTypeImageIndex
		}
	}
	return mime
}

func MapArtifactBlobMimeType(blob accessio.BlobAccess, legacy bool) accessio.BlobAccess {
	mime := blob.MimeType()
	mapped := MapArtifactMimeType(mime, legacy)
	if mapped != mime {
		return accessio.BlobWithMimeType(mapped, blob)
	}
	return blob
}

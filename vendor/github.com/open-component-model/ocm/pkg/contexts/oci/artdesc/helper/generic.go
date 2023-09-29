// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helper

import (
	"fmt"

	"github.com/containerd/containerd/images"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/open-component-model/ocm/pkg/errors"
)

const SchemeVersion = 2

type GenericDescriptor struct {
	ociv1.Manifest
	// Manifests references platform specific manifests.
	Manifests []ociv1.Descriptor `json:"manifests"`
}

func (g *GenericDescriptor) Validate() error {
	if g.SchemaVersion != SchemeVersion {
		return errors.ErrUnknown("schema version", fmt.Sprintf("%d", g.SchemaVersion))
	}
	switch g.MediaType {
	case ociv1.MediaTypeImageIndex:
	case ociv1.MediaTypeImageManifest:
	case images.MediaTypeDockerSchema2Manifest:
	case images.MediaTypeDockerSchema2ManifestList:
	case "":
	default:
		return errors.ErrUnknown("media type", g.MediaType)
	}
	if len(g.Layers) > 0 && len(g.Manifests) > 0 && g.MediaType == "" {
		return errors.Newf("invalid manifest")
	}
	if g.IsManifest() && (g.Config.MediaType == "" || g.Config.Digest == "") {
		return errors.Newf("config media type and digest must be set for oci manifest")
	}
	return nil
}

func (g *GenericDescriptor) IsManifest() bool {
	return g.MediaType == ociv1.MediaTypeImageManifest || len(g.Layers) > 0
}

func (g *GenericDescriptor) AsManifest() *ociv1.Manifest {
	return &ociv1.Manifest{
		Versioned:   g.Versioned,
		MediaType:   g.MediaType,
		Config:      g.Config,
		Layers:      g.Layers,
		Annotations: g.Annotations,
	}
}

func (g *GenericDescriptor) AsIndex() *ociv1.Index {
	return &ociv1.Index{
		Versioned:   g.Versioned,
		MediaType:   g.MediaType,
		Manifests:   g.Manifests,
		Annotations: g.Annotations,
	}
}

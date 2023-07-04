// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprint

import (
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	registry "github.com/open-component-model/ocm/pkg/contexts/ocm/download"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/resourcetypes"
	"github.com/open-component-model/ocm/pkg/finalizer"
	"github.com/open-component-model/ocm/pkg/generics"
	"github.com/open-component-model/ocm/pkg/mime"
	"github.com/open-component-model/ocm/pkg/utils"
)

const (
	TYPE             = resourcetypes.BLUEPRINT
	LEGACY_TYPE      = resourcetypes.BLUEPRINT_LEGACY
	CONFIG_MIME_TYPE = "application/vnd.gardener.landscaper.blueprint.config.v1"
)

type Extractor func(pr common.Printer, handler *Handler, access accessio.DataAccess, path string, fs vfs.FileSystem) (bool, error)

var (
	supportedArtifactTypes    []string
	mimeTypeExtractorRegistry map[string]Extractor
)

type Handler struct {
	ociConfigMimeTypes generics.Set[string]
}

func init() {
	supportedArtifactTypes = []string{TYPE, LEGACY_TYPE}
	mimeTypeExtractorRegistry = map[string]Extractor{
		mime.MIME_TAR:                        ExtractArchive,
		mime.MIME_TGZ:                        ExtractArchive,
		mime.MIME_TGZ_ALT:                    ExtractArchive,
		BLUEPRINT_MIMETYPE:                   ExtractArchive,
		BLUEPRINT_MIMETYPE_LEGACY:            ExtractArchive,
		BLUEPRINT_MIMETYPE_LEGACY_COMPRESSED: ExtractArchive,
	}
	for _, t := range append(artdesc.ToArchiveMediaTypes(artdesc.MediaTypeImageManifest), artdesc.ToArchiveMediaTypes(artdesc.MediaTypeDockerSchema2Manifest)...) {
		mimeTypeExtractorRegistry[t] = ExtractArtifact
	}

	h := New()

	registry.Register(h, registry.ForArtifactType(TYPE))
	registry.Register(h, registry.ForArtifactType(LEGACY_TYPE))
}

func New(configmimetypes ...string) *Handler {
	if len(configmimetypes) == 0 || utils.Optional(configmimetypes...) == "" {
		configmimetypes = []string{CONFIG_MIME_TYPE}
	}
	return &Handler{
		ociConfigMimeTypes: generics.NewSet[string](configmimetypes...),
	}
}

func (h *Handler) Download(pr common.Printer, racc cpi.ResourceAccess, path string, fs vfs.FileSystem) (_ bool, _ string, err error) {
	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagationf(&err, "downloading blueprint")

	meth, err := racc.AccessMethod()
	if err != nil {
		return false, "", err
	}
	finalize.Close(meth)

	ex := mimeTypeExtractorRegistry[meth.MimeType()]
	if ex == nil {
		return false, "", nil
	}

	ok, err := ex(pr, h, meth, path, fs)
	if err != nil || !ok {
		return ok, "", err
	}
	return true, path, nil
}

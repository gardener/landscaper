// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociutils

import (
	"archive/tar"
	"errors"
	"io"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/compression"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
)

func PrintArtifact(pr common.Printer, art cpi.ArtifactAccess, listFiles bool) {
	if art.IsManifest() {
		pr.Printf("type: %s\n", artdesc.MediaTypeImageManifest)
		PrintManifest(pr, art.ManifestAccess(), listFiles)
		return
	}
	if art.IsIndex() {
		pr.Printf("type: %s\n", artdesc.MediaTypeImageIndex)
		PrintIndex(pr, art.IndexAccess(), listFiles)
		return
	}
	pr.Printf("unspecific\n")
}

func PrintManifest(pr common.Printer, m cpi.ManifestAccess, listFiles bool) {
	data, err := accessio.BlobData(m.Blob())
	if err != nil {
		pr.Printf("descriptor: invalid: %s\n", err)
	} else {
		pr.Printf("descriptor: %s\n", string(data))
	}
	man := m.GetDescriptor()
	pr.Printf("config:\n")
	pr.Printf("  type:        %s\n", man.Config.MediaType)
	pr.Printf("  digest:      %s\n", man.Config.Digest)
	pr.Printf("  size:        %d\n", man.Config.Size)

	config, err := accessio.BlobData(m.GetBlob(man.Config.Digest))
	if err != nil {
		pr.Printf("  error getting config blob: %s\n", err.Error())
	} else {
		pr.Printf("  config json: %s\n", string(config))
	}
	h := getHandler(man.Config.MediaType)

	if h != nil {
		h.Description(pr.AddGap("  "), m, config)
	}
	pr.Printf("layers:\n")
	for _, l := range man.Layers {
		pr.Printf("- type:   %s\n", l.MediaType)
		pr.Printf("  digest: %s\n", l.Digest)
		pr.Printf("  size:   %d\n", l.Size)
		blob, err := m.GetBlob(l.Digest)
		if err != nil {
			pr.Printf("  error getting blob: %s\n", err.Error())
		}
		PrintLayer(pr.AddGap("  "), blob, listFiles)
	}
}

func PrintLayer(pr common.Printer, blob accessio.BlobAccess, listFiles bool) {
	reader, err := blob.Reader()
	if err != nil {
		pr.Printf("cannot read blob: %s\n", err.Error())
		return
	}
	defer reader.Close()
	reader, _, err = compression.AutoDecompress(reader)
	if err != nil {
		pr.Printf("cannot decompress blob: %s\n", err.Error())
		return
	}
	tr := tar.NewReader(reader)
	first := true
	for {
		header, err := tr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			if first {
				pr.Printf("no tar\n")
				return
			}
			pr.Printf("tar error: %s\n", err)
			return
		}
		if !listFiles {
			return
		}
		if first {
			pr.Printf("tar filesystem:\n")
		}
		first = false

		switch header.Typeflag {
		case tar.TypeDir:
			pr.Printf("  dir:  %s\n", header.Name)
		case tar.TypeReg:
			pr.Printf("  file: %s\n", header.Name)
		}
	}
}

func PrintIndex(pr common.Printer, i cpi.IndexAccess, listFiles bool) {
	pr.Printf("manifests:\n")
	for _, l := range i.GetDescriptor().Manifests {
		pr.Printf("- type:   %s\n", l.MediaType)
		pr.Printf("  digest: %s\n", l.Digest)
		a, err := i.GetArtifact(l.Digest)
		if err != nil {
			pr.Printf("  error: %s\n", err)
		} else {
			pr.Printf("  resolved artifact:\n")
			PrintArtifact(pr.AddGap("    "), a, listFiles)
		}
	}
}

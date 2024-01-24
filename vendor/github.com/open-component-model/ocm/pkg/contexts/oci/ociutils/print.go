// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociutils

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/compression"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/utils"
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
	data, err := blobaccess.BlobData(m.Blob())
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

	printAnnotations(pr.AddGap("  "), man.Annotations)
	config, err := blobaccess.BlobData(m.GetBlob(man.Config.Digest))
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

func PrintLayer(pr common.Printer, blob blobaccess.BlobAccess, listFiles bool) {
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
	printAnnotations(pr, i.GetDescriptor().Annotations)
	pr.Printf("manifests:\n")
	for _, l := range i.GetDescriptor().Manifests {
		pr.Printf("- type:   %s\n", l.MediaType)
		pr.Printf("  digest: %s\n", l.Digest)
		if l.Platform != nil {
			pr.Printf("  platform:\n")
			pr := pr.AddGap("    ") //nolint: govet // yes
			optS(pr, "OS           ", l.Platform.OS)
			optS(pr, "Architecture ", l.Platform.Architecture)
			optS(pr, "OSCersion    ", l.Platform.OSVersion)
			optS(pr, "Variant      ", l.Platform.Variant)
			if len(l.Platform.OSFeatures) > 0 {
				pr.Printf("OSFeatures:  %s\n", strings.Join(l.Platform.OSFeatures, ", "))
			}
		}
		a, err := i.GetArtifact(l.Digest)
		if err != nil {
			pr.Printf("  error: %s\n", err)
		} else {
			pr.Printf("  resolved artifact:\n")
			PrintArtifact(pr.AddGap("    "), a, listFiles)
		}
	}
}

func optS(pr common.Printer, key string, value string) {
	if value != "" {
		desc := strings.Replace(key, " ", ":", 1)
		pr.Printf("%s %s\n", desc, value)
	}
}

func printAnnotations(pr common.Printer, annos map[string]string) {
	if len(annos) > 0 {
		pr.Printf("annotations:\n")
		keys := utils.StringMapKeys(annos)
		l := 0
		for _, k := range keys {
			if len(k) > l {
				l = len(k)
			}
		}
		for _, k := range keys {
			pr.Printf(fmt.Sprintf("  %%s:%%%ds %%s\n", l-len(k)), k, "", annos[k])
		}
	}
}

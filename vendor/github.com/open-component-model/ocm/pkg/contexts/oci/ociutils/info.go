// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociutils

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/opencontainers/go-digest"
	"sigs.k8s.io/yaml"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/compression"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/mime"
)

type BlobInfo struct {
	Error    string          `json:"error,omitempty"`
	Unparsed string          `json:"unparsed,omitempty"`
	Content  json.RawMessage `json:"content,omitempty"`
	Type     string          `json:"type,omitempty"`
	Digest   digest.Digest   `json:"digest,omitempty"`
	Size     int64           `json:"size,omitempty"`
	Info     interface{}     `json:"info,omitempty"`
}
type ArtifactInfo struct {
	Digest     digest.Digest `json:"digest"`
	Type       string        `json:"type"`
	Descriptor interface{}   `json:"descriptor"`
	Config     *BlobInfo     `json:"config,omitempty"`
	Layers     []*BlobInfo   `json:"layers,omitempty"`
	Manifests  []*BlobInfo   `json:"manifests,omitempty"`
}

func GetArtifactInfo(art cpi.ArtifactAccess, layerFiles bool) *ArtifactInfo {
	if art.IsManifest() {
		return GetManifestInfo(art.ManifestAccess(), layerFiles)
	}
	if art.IsIndex() {
		return GetIndexInfo(art.IndexAccess(), layerFiles)
	}
	return &ArtifactInfo{Type: "unspecific"}
}

func GetManifestInfo(m cpi.ManifestAccess, layerFiles bool) *ArtifactInfo {
	info := &ArtifactInfo{
		Type:       artdesc.MediaTypeImageManifest,
		Descriptor: m.GetDescriptor(),
	}
	b, err := m.Blob()
	if err == nil {
		info.Digest = b.Digest()
	}
	man := m.GetDescriptor()
	cfg := &BlobInfo{
		Content: nil,
		Type:    man.Config.MediaType,
		Digest:  man.Config.Digest,
		Size:    man.Config.Size,
	}
	info.Config = cfg

	config, err := accessio.BlobData(m.GetBlob(man.Config.Digest))
	if err != nil {
		cfg.Error = "error getting config blob: " + err.Error()
	} else {
		cfg.Content = json.RawMessage(config)
	}
	h := getHandler(man.Config.MediaType)

	if h != nil {
		pr, buf := common.NewBufferedPrinter()
		h.Description(pr, m, config)
		cfg.Info = buf.String()
	}
	for _, l := range man.Layers {
		blobinfo := &BlobInfo{
			Type:   l.MediaType,
			Digest: l.Digest,
			Size:   l.Size,
		}
		blob, err := m.GetBlob(l.Digest)
		if err != nil {
			blobinfo.Error = "error getting blob: " + err.Error()
		} else {
			blobinfo.Info = GetLayerInfo(blob, layerFiles)
		}
		info.Layers = append(info.Layers, blobinfo)
	}
	return info
}

type LayerInfo struct {
	Description string      `json:"description,omitempty"`
	Error       string      `json:"error,omitempty"`
	Unparsed    string      `json:"unparsed,omitempty"`
	Content     interface{} `json:"content,omitempty"`
}

func GetLayerInfo(blob accessio.BlobAccess, layerFiles bool) *LayerInfo {
	info := &LayerInfo{}

	if mime.IsJSON(blob.MimeType()) {
		info.Description = "json document"
		data, err := blob.Get()
		if err != nil {
			info.Error = "cannot read blob: " + err.Error()
			return info
		}
		var j interface{}
		err = json.Unmarshal(data, &j)
		if err != nil {
			if len(data) < 10000 {
				info.Unparsed = string(data)
			}
			info.Error = "invalid json: " + err.Error()
			return info
		}
		info.Content = j
		return info
	}
	if mime.IsYAML(blob.MimeType()) {
		info.Description = "yaml document"
		data, err := blob.Get()
		if err != nil {
			info.Error = "cannot read blob: " + err.Error()
			return info
		}
		var j interface{}
		err = yaml.Unmarshal(data, &j)
		if err != nil {
			if len(data) < 10000 {
				info.Unparsed = string(data)
			}
			info.Error = "invalid yaml: " + err.Error()
			return info
		}
		info.Content = j
		return info
	}
	if !layerFiles {
		return nil
	}
	reader, err := blob.Reader()
	if err != nil {
		info.Error = "cannot read blob: " + err.Error()
		return info
	}
	defer reader.Close()
	reader, _, err = compression.AutoDecompress(reader)
	if err != nil {
		info.Error = "cannot decompress blob: " + err.Error()
		return info
	}
	var files []string
	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				info.Content = files
				return info
			}
			if len(files) == 0 {
				info.Description = "no tar"
				return info
			}
			info.Error = fmt.Sprintf("tar error: %s", err)
			return info
		}
		if len(files) == 0 {
			info.Description = "tar file"
		}

		switch header.Typeflag {
		case tar.TypeDir:
			files = append(files, fmt.Sprintf("dir:  %s\n", header.Name))
		case tar.TypeReg:
			files = append(files, fmt.Sprintf("file: %s\n", header.Name))
		}
	}
}

func GetIndexInfo(i cpi.IndexAccess, layerFiles bool) *ArtifactInfo {
	info := &ArtifactInfo{
		Type:       artdesc.MediaTypeImageIndex,
		Descriptor: i.GetDescriptor(),
	}
	b, err := i.Blob()
	if err == nil {
		info.Digest = b.Digest()
	}
	for _, l := range i.GetDescriptor().Manifests {
		blobinfo := &BlobInfo{
			Type:   l.MediaType,
			Digest: l.Digest,
			Size:   l.Size,
		}
		a, err := i.GetArtifact(l.Digest)
		if err != nil {
			blobinfo.Error = fmt.Sprintf("cannot get artifact: %s\n", err)
		} else {
			blobinfo.Info = GetArtifactInfo(a, layerFiles)
		}
		info.Layers = append(info.Layers, blobinfo)
	}
	return info
}

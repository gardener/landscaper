// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package genericocireg

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/opencontainers/go-digest"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ctf/format"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/genericocireg/componentmapping"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

func NewState(mode accessobj.AccessMode, name, version string, access oci.ManifestAccess, compat ...bool) (accessobj.State, error) {
	return accessobj.NewState(mode, NewStateAccess(access, compat...), NewStateHandler(name, version))
}

// StateAccess handles the component descriptor persistence in an OCI Manifest.
type StateAccess struct {
	access     oci.ManifestAccess
	layerMedia string
	compat     bool
}

var _ accessobj.StateAccess = (*StateAccess)(nil)

func NewStateAccess(access oci.ManifestAccess, compat ...bool) accessobj.StateAccess {
	return &StateAccess{
		compat: utils.Optional(compat...),
		access: access,
	}
}

func (s *StateAccess) Get() (accessio.BlobAccess, error) {
	mediaType := s.access.GetDescriptor().Config.MediaType
	switch mediaType {
	case componentmapping.ComponentDescriptorConfigMimeType,
		componentmapping.LegacyComponentDescriptorConfigMimeType,
		componentmapping.Legacy2ComponentDescriptorConfigMimeType:
		return s.get()
	case "":
		return nil, errors.ErrNotFound(cpi.KIND_COMPONENTVERSION)
	default:
		return nil, errors.Newf("artifact is no component: %s", mediaType)
	}
}

func (s *StateAccess) get() (accessio.BlobAccess, error) {
	var config ComponentDescriptorConfig

	data, err := accessio.BlobData(s.access.GetConfigBlob())
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	if config.ComponentDescriptorLayer == nil || config.ComponentDescriptorLayer.Digest == "" {
		return nil, errors.ErrInvalid("component descriptor config")
	}
	switch config.ComponentDescriptorLayer.MediaType {
	case componentmapping.ComponentDescriptorJSONMimeType,
		componentmapping.LegacyComponentDescriptorJSONMimeType,
		componentmapping.ComponentDescriptorYAMLMimeType,
		componentmapping.LegacyComponentDescriptorYAMLMimeType:
		s.layerMedia = ""
		return s.access.GetBlob(config.ComponentDescriptorLayer.Digest)
	case componentmapping.ComponentDescriptorTarMimeType,
		componentmapping.LegacyComponentDescriptorTarMimeType,
		componentmapping.Legacy2ComponentDescriptorTarMimeType:
		d, err := s.access.GetBlob(config.ComponentDescriptorLayer.Digest)
		if err != nil {
			return nil, err
		}
		r, err := d.Reader()
		if err != nil {
			return nil, err
		}
		defer r.Close()
		data, err := s.readComponentDescriptorFromTar(r)
		if err != nil {
			return nil, err
		}
		s.layerMedia = config.ComponentDescriptorLayer.MediaType
		return accessio.BlobAccessForData(componentmapping.ComponentDescriptorYAMLMimeType, data), nil
	default:
		return nil, errors.ErrInvalid("config mediatype", config.ComponentDescriptorLayer.MediaType)
	}
}

// readComponentDescriptorFromTar reads the component descriptor from a tar.
// The component is expected to be inside the tar at "/component-descriptor.yaml".
func (s *StateAccess) readComponentDescriptorFromTar(r io.Reader) ([]byte, error) {
	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, errors.New("no component descriptor found in tar")
			}
			return nil, fmt.Errorf("unable to read tar: %w", err)
		}

		if strings.TrimLeft(header.Name, "/") != compdesc.ComponentDescriptorFileName {
			continue
		}

		var data bytes.Buffer
		//nolint:gosec // We don't know what size limit we could set, the tar
		// archive can be an image layer and that can even reach the gigabyte range.
		// For now, we acknowledge the risk.
		//
		// We checked other softwares and tried to figure out how they manage this,
		// but it's handled the same way.
		if _, err := io.Copy(&data, tr); err != nil {
			return nil, fmt.Errorf("erro while reading component descriptor file from tar: %w", err)
		}
		return data.Bytes(), err
	}
}

func (s StateAccess) Digest() digest.Digest {
	blob, err := s.access.GetConfigBlob()
	if err != nil {
		return ""
	}
	return blob.Digest()
}

func (s *StateAccess) Put(data []byte) error {
	desc := s.access.GetDescriptor()
	mediaType := desc.Config.MediaType
	if mediaType == "" {
		if s.compat {
			mediaType = componentmapping.LegacyComponentDescriptorConfigMimeType
		} else {
			mediaType = componentmapping.ComponentDescriptorConfigMimeType
		}
		desc.Config.MediaType = mediaType
	}

	arch, err := s.writeComponentDescriptorTar(data)
	if err != nil {
		return err
	}
	config := ComponentDescriptorConfig{
		ComponentDescriptorLayer: artdesc.DefaultBlobDescriptor(arch),
	}

	configdata, err := json.Marshal(&config)
	if err != nil {
		return err
	}

	err = s.access.AddBlob(arch)
	if err != nil {
		return err
	}
	s.layerMedia = arch.MimeType()
	configblob := accessio.BlobAccessForData(mediaType, configdata)
	err = s.access.AddBlob(configblob)
	if err != nil {
		return err
	}
	desc.Config = *artdesc.DefaultBlobDescriptor(configblob)
	if len(desc.Layers) < 2 {
		desc.Layers = []ociv1.Descriptor{*artdesc.DefaultBlobDescriptor(arch)}
	} else {
		desc.Layers[0] = *artdesc.DefaultBlobDescriptor(arch)
	}
	return nil
}

// writeComponentDescriptorTar writes the component descriptor into a tar.
// The component is expected to be inside the tar at "/component-descriptor.yaml".
func (s *StateAccess) writeComponentDescriptorTar(data []byte) (cpi.BlobAccess, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	err := tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     componentmapping.ComponentDescriptorFileName,
		Size:     int64(len(data)),
		ModTime:  format.ModTime,
	})

	media := s.layerMedia
	if media == "" {
		if s.compat {
			media = componentmapping.LegacyComponentDescriptorTarMimeType
		} else {
			media = componentmapping.ComponentDescriptorTarMimeType
		}
	}
	if err != nil {
		return nil, errors.Newf("unable to add component descriptor header: %s", err)
	}
	if _, err := io.Copy(tw, bytes.NewBuffer(data)); err != nil {
		return nil, errors.Newf("unable to write component-descriptor to tar: %s", err)
	}
	if err := tw.Close(); err != nil {
		return nil, errors.Newf("unable to close tar writer: %s", err)
	}
	return accessio.BlobAccessForData(media, buf.Bytes()), nil
}

// ComponentDescriptorConfig is a Component-Descriptor OCI configuration that is used to store the reference to the
// (pseudo-)layer used to store the Component-Descriptor in.
type ComponentDescriptorConfig struct {
	ComponentDescriptorLayer *ociv1.Descriptor `json:"componentDescriptorLayer,omitempty"`
}

////////////////////////////////////////////////////////////////////////////////

// StateHandler handles the encoding of a component descriptor.
type StateHandler struct {
	name    string
	version string
}

var _ accessobj.StateHandler = (*StateHandler)(nil)

func NewStateHandler(name, version string) accessobj.StateHandler {
	return &StateHandler{
		name:    name,
		version: version,
	}
}

func (i StateHandler) Initial() interface{} {
	return compdesc.New(i.name, i.version)
}

// Encode always provides a yaml representation.
func (i StateHandler) Encode(d interface{}) ([]byte, error) {
	desc, ok := d.(*compdesc.ComponentDescriptor)
	if !ok {
		return nil, fmt.Errorf("failed to assert type %t to *compdesc.ComponentDescriptor", d)
	}
	desc.Name = i.name
	desc.Version = i.version
	return compdesc.Encode(desc)
}

// Decode always accepts a yaml representation, and therefore json, also.
func (i StateHandler) Decode(data []byte) (interface{}, error) {
	return compdesc.Decode(data)
}

func (i StateHandler) Equivalent(a, b interface{}) bool {
	ea, err := i.Encode(a)
	if err == nil {
		eb, err := i.Encode(b)
		if err == nil {
			return bytes.Equal(ea, eb)
		}
	}
	return reflect.DeepEqual(a, b)
}

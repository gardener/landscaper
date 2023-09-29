// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"reflect"

	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
)

type ManifestStateHandler struct{}

var _ accessobj.StateHandler = &ManifestStateHandler{}

func NewManifestStateHandler() accessobj.StateHandler {
	return &ManifestStateHandler{}
}

func (i ManifestStateHandler) Initial() interface{} {
	return artdesc.NewManifest()
}

func (i ManifestStateHandler) Encode(d interface{}) ([]byte, error) {
	return artdesc.EncodeManifest(d.(*artdesc.Manifest))
}

func (i ManifestStateHandler) Decode(data []byte) (interface{}, error) {
	return artdesc.DecodeManifest(data)
}

func (i ManifestStateHandler) Equivalent(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}

////////////////////////////////////////////////////////////////////////////////

type IndexStateHandler struct{}

var _ accessobj.StateHandler = &IndexStateHandler{}

func NewIndexStateHandler() accessobj.StateHandler {
	return &IndexStateHandler{}
}

func (i IndexStateHandler) Initial() interface{} {
	return artdesc.NewIndex()
}

func (i IndexStateHandler) Encode(d interface{}) ([]byte, error) {
	return artdesc.EncodeIndex(d.(*artdesc.Index))
}

func (i IndexStateHandler) Decode(data []byte) (interface{}, error) {
	return artdesc.DecodeIndex(data)
}

func (i IndexStateHandler) Equivalent(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}

////////////////////////////////////////////////////////////////////////////////

type ArtifactStateHandler struct{}

var _ accessobj.StateHandler = &ArtifactStateHandler{}

func NewArtifactStateHandler() accessobj.StateHandler {
	return &ArtifactStateHandler{}
}

func (i ArtifactStateHandler) Initial() interface{} {
	return artdesc.New()
}

func (i ArtifactStateHandler) Encode(d interface{}) ([]byte, error) {
	return artdesc.Encode(d.(*artdesc.Artifact))
}

func (i ArtifactStateHandler) Decode(data []byte) (interface{}, error) {
	return artdesc.Decode(data)
}

func (i ArtifactStateHandler) Equivalent(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}

// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v2

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ghodss/yaml"
)

// OCIRegistryType is the access type of a oci registry.
const OCIRegistryType = "ociRegistry"

// OCIRegistryAccess describes the access for a oci registry.
type OCIRegistryAccess struct {
	ObjectType `json:",inline"`

	// ImageReference is the actual reference to the oci image repository and tag.
	// The format is expected to be "repository:tag".
	ImageReference string `json:"imageReference"`
}

var _ AccessAccessor = &OCIRegistryAccess{}

func (O OCIRegistryAccess) GetData() ([]byte, error) {
	return json.Marshal(O)
}

func (O *OCIRegistryAccess) SetData(bytes []byte) error {
	var newOCIImage OCIRegistryAccess
	if err := json.Unmarshal(bytes, &newOCIImage); err != nil {
		return err
	}

	O.ImageReference = newOCIImage.ImageReference
	return nil
}

var ociCodec = &AccessCodecWrapper{
	AccessDecoder: AccessDecoderFunc(func(data []byte) (AccessAccessor, error) {
		var ociImage OCIRegistryAccess
		if err := json.Unmarshal(data, &ociImage); err != nil {
			return nil, err
		}
		return &ociImage, nil
	}),
	AccessEncoder: AccessEncoderFunc(func(accessor AccessAccessor) ([]byte, error) {
		ociImage, ok := accessor.(*OCIRegistryAccess)
		if !ok {
			return nil, fmt.Errorf("accessor is not of type %s", OCIImageType)
		}
		return json.Marshal(ociImage)
	}),
}

// WebType is the type of a web component
const WebType = "web"

// Web describes a web resource access that can be fetched via http GET request.
type Web struct {
	ObjectType `json:",inline"`

	// URL is the http get accessible url resource.
	URL string `json:"url"`
}

var _ AccessAccessor = &Web{}

func (w Web) GetData() ([]byte, error) {
	return yaml.Marshal(w)
}

func (w *Web) SetData(bytes []byte) error {
	var newWeb Web
	if err := json.Unmarshal(bytes, &newWeb); err != nil {
		return err
	}

	w.URL = newWeb.URL
	return nil
}

var webCodec = &AccessCodecWrapper{
	AccessDecoder: AccessDecoderFunc(func(data []byte) (AccessAccessor, error) {
		var web Web
		if err := json.Unmarshal(data, &web); err != nil {
			return nil, err
		}
		return &web, nil
	}),
	AccessEncoder: AccessEncoderFunc(func(accessor AccessAccessor) ([]byte, error) {
		web, ok := accessor.(*Web)
		if !ok {
			return nil, fmt.Errorf("accessor is not of type %s", OCIImageType)
		}
		return json.Marshal(web)
	}),
}

// CustomAccess describes a generic dependency of a resolvable component.
type CustomAccess struct {
	ObjectType `json:",inline"`
	Data       map[string]interface{} `json:"-"`
}

var _ AccessAccessor = &CustomAccess{}

func (c CustomAccess) GetData() ([]byte, error) {
	return json.Marshal(c.Data)
}

func (c *CustomAccess) SetData(data []byte) error {
	var values map[string]interface{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return err
	}
	c.Data = values
	return nil
}

var customCodec = &AccessCodecWrapper{
	AccessDecoder: AccessDecoderFunc(func(data []byte) (AccessAccessor, error) {
		var acc CustomAccess
		if err := yaml.Unmarshal(data, &acc); err != nil {
			return nil, err
		}

		var values map[string]interface{}
		if err := json.Unmarshal(data, &values); err != nil {
			return nil, err
		}

		acc.Data = values
		return &acc, nil
	}),
	AccessEncoder: AccessEncoderFunc(func(accessor AccessAccessor) ([]byte, error) {
		custom, ok := accessor.(*CustomAccess)
		if !ok {
			return nil, errors.New("accessor is not a custom type %s")
		}
		return json.Marshal(custom.Data)
	}),
}

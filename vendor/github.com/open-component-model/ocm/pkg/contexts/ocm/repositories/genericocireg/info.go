// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package genericocireg

import (
	"encoding/json"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/oci/ociutils"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/genericocireg/componentmapping"
	"github.com/open-component-model/ocm/pkg/runtime"
)

func init() {
	ociutils.RegisterInfoHandler(componentmapping.ComponentDescriptorConfigMimeType, &handler{})
}

type handler struct{}

type ComponentVersionInfo struct {
	Error       string          `json:"error,omitempty"`
	Description string          `json:"description"`
	Unparsed    string          `json:"unparsed,omitempty"`
	Descriptor  json.RawMessage `json:"descriptor,omitempty"`
}

func (h handler) Info(m cpi.ManifestAccess, config []byte) interface{} {
	info := &ComponentVersionInfo{
		Description: "component version",
	}
	acc := NewStateAccess(m)
	data, err := blobaccess.BlobData(acc.Get())
	if err != nil {
		info.Error = "cannot read component descriptor: " + err.Error()
		return info
	}
	var raw interface{}
	err = json.Unmarshal(data, &raw)
	if err != nil {
		info.Unparsed = string(data)
		return info
	}
	info.Descriptor = data
	return info
}

func (h handler) Description(pr common.Printer, m cpi.ManifestAccess, config []byte) {
	pr.Printf("component version:\n")
	acc := NewStateAccess(m)
	data, err := blobaccess.BlobData(acc.Get())
	if err != nil {
		pr.Printf("  cannot read component descriptor: %s\n", err.Error())
		return
	}
	pr.Printf("  descriptor:\n")
	var raw interface{}
	err = runtime.DefaultYAMLEncoding.Unmarshal(data, &raw)
	if err != nil {
		pr.Printf("    data: %s\n", string(data))
		pr.Printf("  cannot get unmarshal component descriptor: %s\n", err.Error())
		return
	}

	form, err := json.MarshalIndent(raw, "  ", "    ")
	if err != nil {
		pr.Printf("    data: %s\n", string(data))
		pr.Printf("  cannot get marshal component descriptor: %s\n", err.Error())
		return
	}
	pr.Printf("%s\n", string(form))
}

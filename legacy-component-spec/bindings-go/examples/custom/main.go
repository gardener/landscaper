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

package main

import (
	"fmt"
	"os"

	v2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/codec"
)

const NPMType = "npm"

// NPMAccess defines a custom access type that specifies node npm module.
//
// By all access types are decoded with the JSONDecoder so the json annotation have to be defined on the type.
type NPMAccess struct {
	v2.ObjectType
	NodeModule string `json:"nodeModule"`
	Version    string `json:"version"`
}

var _ v2.TypedObjectAccessor = &NPMAccess{}

func main() {
	data := []byte(`
meta:
  schemaVersion: 'v2'

component:
  name: 'github.com/gardener/gardener'
  version: 'v1.7.2'
  provider: internal
  repositoryContexts: []
  sources: []
  componentReferences: []

  resources:
  - name: 'ftp-res'
    version: 'v1.7.2'
    type: 'custom1'
    relation: local
    access:
      type: 'x-ftp'
      url: ftp://example.com/my-resource

  - name: 'node-mod'
    version: '0.0.1'
    type: 'nodeModule'
    relation: external
    access:
      type: 'npm'
      nodeModule: my-module
      version: 0.0.1
`)

	component := &v2.ComponentDescriptor{}
	err := codec.Decode(data, component)
	check(err)

	res, err := component.GetLocalResource("custom1", "ftp-res", "v1.7.2")
	check(err)
	// by default all types are serialized as unstructured type
	ftpAccess := res.Access
	fmt.Println(ftpAccess.Object["url"]) // prints: ftp://example.com/my-resource

	// By all types are decoded with the JSONDecoder.
	res, err = component.GetExternalResource("nodeModule", "node-mod", "0.0.1")
	check(err)
	npmAccess := &NPMAccess{}
	check(res.Access.DecodeInto(npmAccess))
	fmt.Println(npmAccess.NodeModule) // prints: my-module
}

func check(err error) {
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

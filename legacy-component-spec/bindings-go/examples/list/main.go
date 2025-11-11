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

func main() {
	data := []byte(`
meta:
  schemaVersion: 'v2'

components:
- component:
    name: 'github.com/gardener/gardener'
    version: 'v1.7.2'
    provider: internal

    repositoryContexts: []
    sources: []
    componentReferences: []

    resources:
    - name: 'apiserver'
      version: 'v1.7.2'
      type: 'ociImage'
      relation: local
      access:
        type: 'ociRegistry'
        imageReference: 'eu.gcr.io/gardener-project/gardener/apiserver:v1.7.2'

    - name: 'hyperkube'
      version: 'v1.16.4'
      type: 'ociImage'
      relation: external
      access:
        type: 'ociRegistry'
        imageReference: 'k8s.gcr.io/hyperkube:v1.16.4'
- component:
    name: 'github.com/gardener/etcd'
    version: 'v1.3.0'

    provider: internal

    repositoryContexts: []
    sources: []
    componentReferences: []

    resources:
    - name: 'etcd'
      version: 'v3.5.4'
      type: 'ociImage'
      relation: external
      access:
        type: 'ociRegistry'
        imageReference: 'quay.io/coreos/etcd:v3.5.4'
`)

	list := &v2.ComponentDescriptorList{}
	err := codec.Decode(data, list)
	check(err)

	// get component by name and version
	comp, err := list.GetComponent("github.com/gardener/etcd", "v1.3.0")
	check(err)

	fmt.Println(comp.Resources[0].Name) // prints: etcd

	// get a component by its name
	// The method returns a list as there could be multiple components with the same name but different version
	comps := list.GetComponentByName("github.com/gardener/gardener")
	fmt.Println(len(comps)) // prints: 1
}

func check(err error) {
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

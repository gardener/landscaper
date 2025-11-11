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
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/utils/selector"
)

func main() {
	data := []byte(`
meta:
  schemaVersion: 'v2'

component:
  name: 'github.com/gardener/gardener'
  version: 'v1.7.2'

  provider: internal

  repositoryContexts:
  - type: ociRegistry
    baseUrl: example.com
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
    extraIdentity:
      myid: '1'
    relation: external
    access:
      type: 'ociRegistry'
      imageReference: 'k8s.gcr.io/hyperkube:v1.16.4'
  - name: 'hyperkube'
    version: 'v1.17.4'
    type: 'ociImage'
    extraIdentity:
      myid: '2'
    relation: external
    access:
      type: 'ociRegistry'
      imageReference: 'k8s.gcr.io/hyperkube:v1.16.4'
`)

	component := &v2.ComponentDescriptor{}
	err := codec.Decode(data, component)
	check(err)

	/////////////////////////////////
	//  Repository Context
	////////////////////////////////

	// get the latest repository context.
	// the context is returned as unstructured object (similar to the access types) as differnt repository types
	// with different attributes are possible.
	unstructuredRepoCtx := component.GetEffectiveRepositoryContext()
	// decode the unstructured type into a specific type
	ociRepo := &v2.OCIRegistryRepository{}
	check(unstructuredRepoCtx.DecodeInto(ociRepo))
	fmt.Printf("%s\n", ociRepo.BaseURL) // prints "example.com"

	/////////////////////////////////
	//  Resourcess
	////////////////////////////////

	// get a specific local resource
	res, err := component.GetLocalResource(v2.OCIImageType, "apiserver", "v1.7.2")
	check(err)
	fmt.Printf("%#v\n", res)

	// get a specific external resource
	res, err = component.GetExternalResource(v2.OCIImageType, "hyperkube", "v1.16.4")
	check(err)
	fmt.Printf("%#v\n", res)

	// get the access for a resource
	// specific access type can be decoded using the access type codec.
	ociAccess := &v2.OCIRegistryAccess{}
	check(res.Access.DecodeInto(ociAccess))
	fmt.Println(ociAccess.ImageReference) // prints: k8s.gcr.io/hyperkube:v1.16.4

	/////////////////////////////////
	//  Identity
	////////////////////////////////

	// get a component by its identity via selectors
	idSelector := selector.DefaultSelector{
		"name": "hyperkube",
	}
	resources, err := component.GetResourcesBySelector(idSelector)
	check(err)
	fmt.Printf("%d\n", len(resources)) // prints "2" as both hyperkube images match the identity

	// get a component by additional identity information
	idSelector = selector.DefaultSelector{
		"name": "hyperkube",
		"myid": "1",
	}
	resources, err = component.GetResourcesBySelector(idSelector)
	check(err)
	fmt.Printf("%d\n", len(resources))                               // prints "1" as only one hyperkube image matches the name and myid attribute.
	fmt.Printf("%s - %s\n", resources[0].Name, resources[0].Version) // prints "hyperkube - v1.16.4"

	// select a resource by jsonschema
	schemaSelector, err := selector.NewJSONSchemaSelectorFromString(`
type: object
properties:
  name:
    type: string
    enum: ["hyperkube"]
  myid:
    type: string
    enum: ["1"]
`)
	check(err)

	resources, err = component.GetResourcesBySelector(schemaSelector)
	check(err)
	fmt.Printf("%d\n", len(resources))                               // prints "1" as only one hyperkube image matches the name and myid attribute.
	fmt.Printf("%s - %s\n", resources[0].Name, resources[0].Version) // prints "hyperkube - v1.16.4"
}

func check(err error) {
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

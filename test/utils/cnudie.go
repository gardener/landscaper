// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

// MakeRepositoryContext creates a new oci registry repository context.
func MakeRepositoryContext(rc cdv2.TypedObjectAccessor) *cdv2.UnstructuredTypedObject {
	rctx, _ := cdv2.NewUnstructured(rc)
	return &rctx
}

// DefaultRepositoryContext creates a new oci registry repository context.
func DefaultRepositoryContext(baseUrl string) *cdv2.UnstructuredTypedObject {
	return MakeRepositoryContext(cdv2.NewOCIRegistryRepository(baseUrl, ""))
}

// ExampleRepositoryContext creates a new example repository context.
func ExampleRepositoryContext() *cdv2.UnstructuredTypedObject {
	return DefaultRepositoryContext("example.com")
}

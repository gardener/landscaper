// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

// DefaultRepositoryContext creates a new oci registry repository context.
func DefaultRepositoryContext(baseUrl string) *cdv2.UnstructuredTypedObject {
	rctx, _ := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository(baseUrl, ""))
	return &rctx
}

// ExampleRepositoryContext creates a new example repository context.
func ExampleRepositoryContext() *cdv2.UnstructuredTypedObject {
	return DefaultRepositoryContext("example.com")
}

// BuildLocalFilesystemResource creates a local filesystem blob resource from a path.
func BuildLocalFilesystemResource(name, ttype, mediaType, path string) cdv2.Resource {
	res := cdv2.Resource{}
	res.Name = name
	res.Type = ttype
	res.Relation = cdv2.LocalRelation

	localFsAccess := cdv2.NewLocalFilesystemBlobAccess(path, mediaType)
	uAcc, err := cdv2.NewUnstructured(localFsAccess)
	ExpectNoError(err)
	res.Access = &uAcc
	return res
}

// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

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

// CreateExampleDefaultContext creates default context with the example repository.
func CreateExampleDefaultContext(ctx context.Context, kubeClient client.Client, namespaces ...string) error {
	return CreateDefaultContext(ctx, kubeClient, cdv2.NewOCIRegistryRepository("example.com", ""), namespaces...)
}

// CreateDefaultContext creates default context with a given repository context.
func CreateDefaultContext(ctx context.Context, kubeClient client.Client, repoCtx cdv2.TypedObjectAccessor, namespaces ...string) error {
	for _, ns := range namespaces {
		lsCtx := &lsv1alpha1.Context{}
		lsCtx.Name = lsv1alpha1.DefaultContextName
		lsCtx.Namespace = ns
		if _, err := controllerutil.CreateOrUpdate(ctx, kubeClient, lsCtx, func() error {
			lsCtx.RepositoryContext = MakeRepositoryContext(repoCtx)
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

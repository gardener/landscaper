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

package ctf

import (
	"context"
	"errors"
	"fmt"
	"io"

	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

// ComponentDescriptorFileName is the name of the component-descriptor file.
const ComponentDescriptorFileName = "component-descriptor.yaml"

// ArtefactDescriptorFileName is the name of the artefact-descriptor file.
const ArtefactDescriptorFileName = "artefact-descriptor.yaml"

// ManifestFileName is the name of the manifest json file.
const ManifestFileName = "manifest.json"

// BlobsDirectoryName is the name of the blob directory in the tar.
const BlobsDirectoryName = "blobs"

var UnsupportedResolveType = errors.New("UnsupportedResolveType")

// ComponentResolver describes a general interface to resolve a component descriptor
type ComponentResolver interface {
	Resolve(ctx context.Context, repoCtx v2.RepositoryContext, name, version string) (*v2.ComponentDescriptor, BlobResolver, error)
}

// BlobResolver defines a resolver that can fetch
// blobs in a specific context defined in a component descriptor.
type BlobResolver interface {
	Info(ctx context.Context, res v2.Resource) (*BlobInfo, error)
	Resolve(ctx context.Context, res v2.Resource, writer io.Writer) (*BlobInfo, error)
}

// TypedBlobResolver defines a blob resolver
// that is able to resolves a set of access types.
type TypedBlobResolver interface {
	BlobResolver
	// CanResolve returns whether the resolver is able to resolve the
	// resource.
	CanResolve(resource v2.Resource) bool
}

// BlobInfo describes a blob.
type BlobInfo struct {
	// MediaType is the media type of the object this schema refers to.
	MediaType string `json:"mediaType,omitempty"`

	// Digest is the digest of the targeted content.
	Digest string `json:"digest"`

	// Size specifies the size in bytes of the blob.
	Size int64 `json:"size"`
}

// AggregatedBlobResolver combines multiple blob resolver.
// Is automatically picks the right resolver based on the resolvers type information.
// If multiple resolvers match, the first matching resolver is used.
type AggregatedBlobResolver struct {
	resolver []TypedBlobResolver
}

var _ BlobResolver = &AggregatedBlobResolver{}

// NewAggregatedBlobResolver creates a new aggregated resolver.
// Note that only typed resolvers can be added.
// An error is thrown if a resolver does not implement the supported types.
func NewAggregatedBlobResolver(resolvers ...BlobResolver) (*AggregatedBlobResolver, error) {
	agg := &AggregatedBlobResolver{
		resolver: make([]TypedBlobResolver, 0),
	}
	if err := agg.Add(resolvers...); err != nil {
		return nil, err
	}
	return agg, nil
}

// Add adds multiple resolvers to the aggregator.
// Only typed resolvers can be added.
// An error is thrown if a resolver does not implement the supported types function.
func (a *AggregatedBlobResolver) Add(resolvers ...BlobResolver) error {
	for i, resolver := range resolvers {
		typedResolver, ok := resolver.(TypedBlobResolver)
		if !ok {
			return fmt.Errorf("resolver %d does not implement supported types interface", i)
		}
		a.resolver = append(a.resolver, typedResolver)
	}
	return nil
}

func (a *AggregatedBlobResolver) Info(ctx context.Context, res v2.Resource) (*BlobInfo, error) {
	resolver, err := a.getResolver(res)
	if err != nil {
		return nil, err
	}
	return resolver.Info(ctx, res)
}

func (a *AggregatedBlobResolver) Resolve(ctx context.Context, res v2.Resource, writer io.Writer) (*BlobInfo, error) {
	resolver, err := a.getResolver(res)
	if err != nil {
		return nil, err
	}
	return resolver.Resolve(ctx, res, writer)
}

func (a *AggregatedBlobResolver) getResolver(res v2.Resource) (BlobResolver, error) {
	if res.Access == nil {
		return nil, fmt.Errorf("no access is defined")
	}

	for _, resolver := range a.resolver {
		if resolver.CanResolve(res) {
			return resolver, nil
		}
	}
	return nil, UnsupportedResolveType
}

// AggregateBlobResolvers aggregartes two resolvers to one by using aggregated blob resolver.
func AggregateBlobResolvers(a, b BlobResolver) (BlobResolver, error) {
	aggregated, ok := a.(*AggregatedBlobResolver)
	if ok {
		if err := aggregated.Add(b); err != nil {
			return nil, fmt.Errorf("unable to add second resolver to aggreagted first resolver: %w", err)
		}
		return aggregated, nil
	}

	aggregated, ok = b.(*AggregatedBlobResolver)
	if ok {
		if err := aggregated.Add(a); err != nil {
			return nil, fmt.Errorf("unable to add first resolver to aggreagted second resolver: %w", err)
		}
		return aggregated, nil
	}

	// create a new aggreagted resolver if neither a nor b are aggregations
	return NewAggregatedBlobResolver(a, b)
}

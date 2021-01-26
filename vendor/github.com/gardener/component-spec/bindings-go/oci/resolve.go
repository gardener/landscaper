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

package oci

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/gardener/component-spec/bindings-go/ctf"
)

type Client interface {
	// GetManifest returns the ocispec Manifest for a reference
	GetManifest(ctx context.Context, ref string) (*ocispecv1.Manifest, error)

	// Fetch fetches the blob for the given ocispec Descriptor.
	Fetch(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) error
}

// OCIRef generates the oci reference url from the repository context and a component name and version.
func OCIRef(repoCtx v2.RepositoryContext, name, version string) (string, error) {
	u, err := url.Parse(repoCtx.BaseURL)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, ComponentDescriptorNamespace, name)
	return fmt.Sprintf("%s:%s", u.String(), version), nil
}

// Resolver is a generic resolve to resolve a component descriptor from a oci registry
type Resolver struct {
	repoCtx v2.RepositoryContext
	client  Client
	decodeOpts []codec.DecodeOption
}

// NewResolver creates a new resolver.
func NewResolver(decodeOpts ...codec.DecodeOption) *Resolver {
	return &Resolver{
		decodeOpts: decodeOpts,
	}
}

// WithRepositoryContext sets the repository context of the resolver
func (r *Resolver) WithRepositoryContext(ctx v2.RepositoryContext) *Resolver {
	r.repoCtx = ctx
	return r
}

// WithOCIClient sets the oci client context of the resolver
func (r *Resolver) WithOCIClient(client Client) *Resolver {
	r.client = client
	return r
}

// Resolve resolves a component descriptor by name and version within the configured context.
func (r *Resolver) Resolve(ctx context.Context, name, version string) (*v2.ComponentDescriptor, ctf.BlobResolver, error) {
	if r.repoCtx.Type != v2.OCIRegistryType {
		return nil, nil, fmt.Errorf("unsupported type %s expected %s", r.repoCtx.Type, v2.OCIRegistryType)
	}
	ref, err := OCIRef(r.repoCtx, name, version)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to generate oci reference: %w", err)
	}

	manifest, err := r.client.GetManifest(ctx, ref)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to fetch manifest from ref %s: %w", ref, err)
	}

	componentConfig, err := r.getComponentConfig(ctx, ref, manifest)
	if err != nil {
		return nil, nil, err
	}

	componentDescriptorLayer := GetLayerWithDigest(manifest.Layers, componentConfig.ComponentDescriptorLayer.Digest)
	if componentDescriptorLayer == nil {
		return nil, nil, fmt.Errorf("no component descriptor layer defined")
	}

	var componentDescriptorLayerBytes bytes.Buffer
	if err := r.client.Fetch(ctx, ref, *componentDescriptorLayer, &componentDescriptorLayerBytes); err != nil {
		return nil, nil, fmt.Errorf("unable to fetch component descriptor layer: %w", err)
	}

	componentDescriptorBytes := componentDescriptorLayerBytes.Bytes()
	switch componentDescriptorLayer.MediaType {
	case ComponentDescriptorTarMimeType, LegacyComponentDescriptorTarMimeType:
		componentDescriptorBytes, err = ReadComponentDescriptorFromTar(&componentDescriptorLayerBytes)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to read component descriptor from tar: %w", err)
		}
	case ComponentDescriptorJSONMimeType:
	default:
		return nil, nil, fmt.Errorf("unsupported media type %q", componentDescriptorLayer.MediaType)
	}

	cd := &v2.ComponentDescriptor{}
	if err := codec.Decode(componentDescriptorBytes, cd, r.decodeOpts...); err != nil {
		return nil, nil, fmt.Errorf("unable to decode component descriptor: %w", err)
	}
	return cd, newBlobResolver(r.client, ref, manifest, cd), nil
}

// ToComponentArchive creates a tar archive in the CTF (Cnudie Transport Format) from the given component descriptor.
func (r *Resolver) ToComponentArchive(ctx context.Context, name, version string, writer io.Writer) error {
	cd, blobresolver, err := r.Resolve(ctx, name, version)
	if err != nil {
		return err
	}

	ca := ctf.NewComponentArchive(cd, memoryfs.New())
	for _, res := range cd.Resources {
		if err := ca.AddResourceFromResolver(ctx, &res, blobresolver); err != nil {
			return fmt.Errorf("unable to add resource %s to archive: %w", res.GetName(), err)
		}
	}

	return ca.WriteTar(writer)
}

func (r *Resolver) getComponentConfig(ctx context.Context, ref string, manifest *ocispecv1.Manifest) (*ComponentDescriptorConfig, error) {
	if manifest.Config.MediaType != ComponentDescriptorConfigMimeType &&
		manifest.Config.MediaType != ComponentDescriptorLegacyConfigMimeType {
		return nil, fmt.Errorf("unknown component config type '%s' expected '%s'", manifest.Config.MediaType, ComponentDescriptorConfigMimeType)
	}

	var data bytes.Buffer
	if err := r.client.Fetch(ctx, ref, manifest.Config, &data); err != nil {
		return nil, fmt.Errorf("unable to resolve component config: %w", err)
	}

	componentConfig := &ComponentDescriptorConfig{}
	if err := json.Unmarshal(data.Bytes(), componentConfig); err != nil {
		return nil, fmt.Errorf("unable to decode manifest config into component config: %w", err)
	}

	return componentConfig, nil
}

// blobResolver implements the BlobResolver interface
// and is returned when a component descriptor is resolved.
type blobResolver struct {
	client   Client
	ref      string
	manifest *ocispecv1.Manifest
	cd       *v2.ComponentDescriptor
}

func newBlobResolver(client Client, ref string, manifest *ocispecv1.Manifest, cd *v2.ComponentDescriptor) ctf.BlobResolver {
	return &blobResolver{
		client:   client,
		ref:      ref,
		manifest: manifest,
		cd:       cd,
	}
}

func (b *blobResolver) CanResolve(res v2.Resource) bool {
	return res.Access != nil && res.Access.GetType() == v2.LocalOCIBlobType || res.Access.GetType() == v2.OCIBlobType
}

func (b *blobResolver) Info(ctx context.Context, res v2.Resource) (*ctf.BlobInfo, error) {
	return b.resolve(ctx, res, nil)
}

func (b *blobResolver) Resolve(ctx context.Context, res v2.Resource, writer io.Writer) (*ctf.BlobInfo, error) {
	return b.resolve(ctx, res, writer)
}

func (b *blobResolver) resolve(ctx context.Context, res v2.Resource, writer io.Writer) (*ctf.BlobInfo, error) {
	switch res.Access.GetType() {
	case v2.LocalOCIBlobType:
		localOCIAccess := &v2.LocalOCIBlobAccess{}
		if err := v2.NewCodec(nil, nil, nil).Decode(res.Access.Raw, localOCIAccess); err != nil {
			return nil, fmt.Errorf("unable to decode access to type '%s': %w", res.Access.GetType(), err)
		}

		blobLayer := GetLayerWithDigest(b.manifest.Layers, localOCIAccess.Digest)
		if blobLayer == nil {
			return nil, fmt.Errorf("oci blob layer with digest %s not found in component descriptor manifest", localOCIAccess.Digest)
		}

		if writer != nil {
			if err := b.client.Fetch(ctx, b.ref, *blobLayer, writer); err != nil {
				return nil, err
			}
		}

		return &ctf.BlobInfo{
			MediaType: blobLayer.MediaType,
			Digest:    localOCIAccess.Digest,
			Size:      blobLayer.Size,
		}, nil
	case v2.OCIBlobType:
		ociBlobAccess := &v2.OCIBlobAccess{}
		if err := v2.NewCodec(nil, nil, nil).Decode(res.Access.Raw, ociBlobAccess); err != nil {
			return nil, fmt.Errorf("unable to decode access to type '%s': %w", res.Access.GetType(), err)
		}

		if writer != nil {
			if err := b.client.Fetch(ctx, b.ref, ocispecv1.Descriptor{
				MediaType: ociBlobAccess.MediaType,
				Digest:    digest.Digest(ociBlobAccess.Digest),
				Size:      ociBlobAccess.Size,
			}, writer); err != nil {
				return nil, err
			}
		}
		return &ctf.BlobInfo{
			MediaType: ociBlobAccess.MediaType,
			Digest:    ociBlobAccess.Digest,
			Size:      ociBlobAccess.Size,
		}, nil
	default:
		return nil, fmt.Errorf("unable to resolve access of type %s", res.Access.GetType())
	}
}

// GetLayerWithDigest returns the layer that matches the given digest.
// It returns nil if no layer matches the digest.
func GetLayerWithDigest(layers []ocispecv1.Descriptor, digest string) *ocispecv1.Descriptor {
	for _, layer := range layers {
		if layer.Digest.String() == digest {
			return &layer
		}
	}
	return nil
}

// ReadComponentDescriptorFromTar reads the component descriptor from a tar.
// The component is expected to be inside the tar at "/component-descriptor.yaml"
func ReadComponentDescriptorFromTar(r io.Reader) ([]byte, error) {
	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				return nil, errors.New("no component descriptor found in tar")
			}
			return nil, fmt.Errorf("unable to read tar: %w", err)
		}

		if strings.TrimLeft(header.Name, "/") != ctf.ComponentDescriptorFileName {
			continue
		}

		var data bytes.Buffer
		if _, err := io.Copy(&data, tr); err != nil {
			return nil, fmt.Errorf("erro while reading component descriptor file from tar: %w", err)
		}
		return data.Bytes(), err
	}
}

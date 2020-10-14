// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsregistry

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/go-logr/logr"

	"github.com/gardener/landscaper/pkg/apis/config"
	"github.com/gardener/landscaper/pkg/utils/oci"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

// ComponentDescriptorMediaType is the media type containing the component descriptor.
const ComponentDescriptorMediaType = "application/sap-cnudie+tar"

// ComponentDescriptorFileName is the name of the file
const ComponentDescriptorFileName = "component-descriptor.yaml"

// ociClient is a component descriptor repository implementation
// that resolves component references stored in an oci repository.
type ociClient struct {
	oci oci.Client
}

// NewOCIRegistry creates a new oci registry from a oci config.
func NewOCIRegistry(log logr.Logger, config *config.OCIConfiguration) (TypedRegistry, error) {
	client, err := oci.NewClient(log, oci.WithConfiguration(config))
	if err != nil {
		return nil, err
	}

	return &ociClient{
		oci: client,
	}, nil
}

// NewOCIRegistryWithOCIClient creates a new oci registry with a oci ociClient
func NewOCIRegistryWithOCIClient(log logr.Logger, client oci.Client) (TypedRegistry, error) {
	return &ociClient{
		oci: client,
	}, nil
}

// Type return the oci registry type that can be handled by this ociClient
func (r *ociClient) Type() string {
	return cdv2.OCIRegistryType
}

func (r *ociClient) InjectCache(c cache.Cache) error {
	return cache.InjectCacheInto(r.oci, c)
}

// Get resolves a reference and returns the component descriptor.
func (r *ociClient) Resolve(ctx context.Context, repoCtx cdv2.RepositoryContext, ref cdv2.ObjectMeta) (*cdv2.ComponentDescriptor, error) {
	if repoCtx.Type != cdv2.OCIRegistryType {
		return nil, fmt.Errorf("unsupported type %s expected %s", repoCtx.Type, cdv2.OCIRegistryType)
	}
	refPath, err := OCIRef(repoCtx, ref)
	if err != nil {
		return nil, err
	}

	manifest, err := r.oci.GetManifest(ctx, refPath)
	if err != nil {
		return nil, err
	}

	if len(manifest.Layers) != 1 {
		return nil, errors.New("manifest must contain 1 layer")
	}
	if manifest.Layers[0].MediaType != ComponentDescriptorMediaType {
		return nil, fmt.Errorf("unexpected media type %s, expected %s", manifest.Layers[0].MediaType, ComponentDescriptorMediaType)
	}

	var data bytes.Buffer
	if err := r.oci.Fetch(ctx, refPath, manifest.Layers[0], &data); err != nil {
		return nil, err
	}

	compDescData, err := readCompDescFromTar(&data)
	if err != nil {
		return nil, err
	}

	cd := &cdv2.ComponentDescriptor{}
	if err := codec.Decode(compDescData, cd); err != nil {
		return nil, err
	}

	return cd, nil
}

// OCIRef returns the oci artifacts uri for a repository context and object metadata
func OCIRef(repoCtx cdv2.RepositoryContext, ref cdv2.ObjectMeta) (string, error) {
	u, err := url.Parse(repoCtx.BaseURL)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, ref.Name)
	return fmt.Sprintf("%s:%s", u.String(), ref.Version), nil
}

func readCompDescFromTar(data io.Reader) ([]byte, error) {
	tr := tar.NewReader(data)
	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				return nil, cdv2.NotFound
			}
			return nil, err
		}

		if header.Name != ComponentDescriptorFileName {
			continue
		}

		if header.Typeflag == tar.TypeReg {
			var compDescData bytes.Buffer
			if _, err := io.Copy(&compDescData, tr); err != nil {
				return nil, err
			}
			return compDescData.Bytes(), nil
		}
	}
}

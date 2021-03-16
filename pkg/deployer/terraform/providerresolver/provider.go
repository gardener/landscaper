// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package providerresolver

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gardener/component-cli/ociclient"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"

	terraformv1alpha1 "github.com/gardener/landscaper/apis/deployer/terraform/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/pkg/utils"
)

// NoProviderDefinedError is the error that is returned if no provider was provided
var NoProviderDefinedError = errors.New("no provider was provided")

// WebMediaType is the expected media type for providers that are downloaded from a web server.
const WebMediaType = "application/zip"

// ResolveProviders is a helper function to resolve all providers of a providers list.
func ResolveProviders(ctx context.Context, resolver *ProviderResolver, providers terraformv1alpha1.TerraformProviders) error {
	for i, provider := range providers {
		if err := resolver.Resolve(ctx, provider); err != nil {
			return fmt.Errorf("unable to resolve provider %d: %w", i, err)
		}
	}
	return nil
}

// ProviderResolver contains all functionality to resolve providers stored on different locations.
type ProviderResolver struct {
	log          logr.Logger
	ociClient    ociclient.Client
	fs           vfs.FileSystem
	providersDir string
}

// NewProviderResolver creates a new provider resolver.
func NewProviderResolver(log logr.Logger, ociClient ociclient.Client) *ProviderResolver {
	return &ProviderResolver{
		log:          log,
		ociClient:    ociClient,
		fs:           osfs.New(),
		providersDir: "/tmp/terraform.d/plugins",
	}
}

// WithFs sets the filesystem that should be used for the provider.
func (r *ProviderResolver) WithFs(fs vfs.FileSystem) *ProviderResolver {
	r.fs = fs
	return r
}

// ProvidersDir sets the plugin cache path where the provider is downloaded to.
func (r *ProviderResolver) ProvidersDir(path string) *ProviderResolver {
	r.providersDir = path
	return r
}

// Resolve resolves the provider based on a chart access configuration.
func (r ProviderResolver) Resolve(ctx context.Context, providerConfig terraformv1alpha1.TerraformProvider) (err error) {
	// create directory if not exist
	_, err = r.fs.Stat(r.providersDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("unable to plugin directory info %q: %w", r.providersDir, err)
		}

		if err := r.fs.MkdirAll(r.providersDir, os.ModePerm); err != nil {
			return fmt.Errorf("unable to create plugin directory %q: %w", r.providersDir, err)
		}
	}

	providerPath := r.providerFilepath(providerConfig.Name, providerConfig.Version)
	f, err := r.fs.OpenFile(providerPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to open plugin file %q: %w", providerPath, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			r.log.Error(err, "unable to close file")
		}
		r.log.Info(fmt.Sprintf("Successfully downlaoded provider %q %q to %q",
			providerConfig.Name, providerConfig.Version, providerPath))
	}()

	if len(providerConfig.URL) != 0 {
		return r.fetchProviderFromUrl(f, providerConfig.URL)
	}

	// fetch the chart from a component descriptor defined resource
	if providerConfig.FromResource != nil {
		return r.fetchProviderFromResource(ctx, f, providerConfig.FromResource)
	}

	return NoProviderDefinedError
}

func (r *ProviderResolver) fetchProviderFromUrl(writer io.Writer, url string) error {
	res, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("unable to fetch provider from %q: %w", url, err)
	}
	contentTypes, ok := res.Header["Content-Type"]
	if !ok || len(contentTypes) == 0 || !utils.StringIsOneOf(WebMediaType, contentTypes...) {
		return fmt.Errorf("unexpected content type from %q expected %s", url, WebMediaType)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, res.Body); err != nil {
		return fmt.Errorf("unable to download provider: %w", err)
	}
	if err := res.Body.Close(); err != nil {
		return fmt.Errorf("unable to close remote stream from %q: %w", url, err)
	}

	if buf.Len() == 0 {
		return fmt.Errorf("no content downloaded form %q", url)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return fmt.Errorf("unable to read zip archive: %w", err)
	}
	if len(zr.File) != 1 {
		return fmt.Errorf("expected the provider zip archive to only contain one file but got %d", len(zr.File))
	}
	file, err := zr.File[0].Open()
	if err != nil {
		return fmt.Errorf("unable to read %q in zip archive: %w", zr.File[0].Name, err)
	}
	defer file.Close()
	if _, err := io.Copy(writer, file); err != nil {
		return fmt.Errorf("unable to download provider: %w", err)
	}
	return err
}

func (r *ProviderResolver) fetchProviderFromResource(ctx context.Context, writer io.Writer, ref *terraformv1alpha1.RemoteTerraformReference) error {
	// we also have to add a custom resolver for the "ociImage" resolver as we have to implement the
	// helm specific oci manifest structure
	compResolver, err := componentsregistry.NewOCIRegistryWithOCIClient(r.ociClient, ref.Inline)
	if err != nil {
		return fmt.Errorf("unable to build component resolver: %w", err)
	}

	cdRef := installations.GeReferenceFromComponentDescriptorDefinition(&ref.ComponentDescriptorDefinition)
	if cdRef == nil {
		return fmt.Errorf("no component descriptor reference found for %q", ref.ResourceName)
	}

	cd, blobResolver, err := compResolver.Resolve(ctx, *cdRef.RepositoryContext, cdRef.ComponentName, cdRef.Version)
	if err != nil {
		return fmt.Errorf("unable to get component descriptor for %q: %w", cdRef.ComponentName, err)
	}

	resources, err := cd.GetResourcesByName(ref.ResourceName)
	if err != nil {
		r.log.Error(err, "unable to find helm resource")
		return fmt.Errorf("unable to find resource with name %q in component descriptor", ref.ResourceName)
	}
	if len(resources) != 1 {
		return fmt.Errorf("resource with name %q cannot be uniquly identified", ref.ResourceName)
	}
	res := resources[0]

	if _, err := blobResolver.Resolve(ctx, res, writer); err != nil {
		return fmt.Errorf("unable to resolve chart from resource %q: %w", ref.ResourceName, err)
	}
	return err
}

// pluginFilename returns the path the the file of a terraform plugin.
func (r *ProviderResolver) providerFilepath(name, version string) string {
	return filepath.Join(r.providersDir, TerraformProviderName(name, version))
}

// TerraformProviderName generates the file name of a plugin.
// This is based on https://www.terraform.io/docs/configuration-0-11/providers.html#plugin-names-and-versions.
func TerraformProviderName(name, version string) string {
	return fmt.Sprintf("terraform-provider-%s_%s", name, version)
}

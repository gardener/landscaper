// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociclient

import (
	"context"
	"io"
	"net/http"

	"github.com/containerd/containerd/remotes"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/credentials"
)

type Client interface {
	// GetManifest returns the ocispec Manifest for a reference
	GetManifest(ctx context.Context, ref string) (*ocispecv1.Manifest, error)

	// Fetch fetches the blob for the given ocispec Descriptor.
	Fetch(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) error

	// PushManifest uploads the given ocispec Descriptor to the given ref.
	PushManifest(ctx context.Context, ref string, manifest *ocispecv1.Manifest) error
}

// ExtendedClient defines an oci client with extended functionality that may not work with all registries.
type ExtendedClient interface {
	Client
	// ListTags returns a list of all tags of the given ref.
	ListTags(ctx context.Context, ref string) ([]string, error)
	// ListRepositories lists all repositories for the given registry host.
	ListRepositories(ctx context.Context, registryHost string) ([]string, error)
}

// Resolver is a interface that should return a new resolver for a given ref if called.
type Resolver interface {
	// Resolver returns a new authenticated resolver.
	Resolver(ctx context.Context, ref string, client *http.Client, plainHTTP bool) (remotes.Resolver, error)
	// GetCredentials returns the username and password for a hostname if defined.
	GetCredentials(hostname string) (username, password string, err error)
}

// Options contains all client options to configure the oci client.
type Options struct {
	// Paths configures local paths to search for docker configuration files
	Paths []string

	// AllowPlainHttp allows the fallback to http if https is not supported by the registry.
	AllowPlainHttp bool

	// Resolver sets the used resolver.
	// A default resolver will be created if not given.
	Resolver Resolver

	// CacheConfig contains the cache configuration.
	// Tis configuration will automatically create a new cache based on that configuration.
	// This cache can be overwritten with the Cache property.
	CacheConfig *cache.Options

	// Cache is the oci cache to be used by the client
	Cache cache.Cache

	// CustomMediaTypes defines the custom known media types
	CustomMediaTypes sets.String

	HTTPClient *http.Client
}

// Option is the interface to specify different cache options
type Option interface {
	ApplyOption(options *Options)
}

// ApplyOptions applies the given list options on these options,
// and then returns itself (for convenient chaining).
func (o *Options) ApplyOptions(opts []Option) *Options {
	for _, opt := range opts {
		if opt != nil {
			opt.ApplyOption(o)
		}
	}
	return o
}

// WithCache configures a cache for the oci client
type WithCache struct {
	cache.Cache
}

func (c WithCache) ApplyOption(options *Options) {
	options.Cache = c.Cache
}

// WithKeyring return a option that configures the resolver to use the given oci keyring
func WithKeyring(ring credentials.OCIKeyring) Option {
	return WithKeyringOption{
		OCIKeyring: ring,
	}
}

// WithKeyringOption configures the resolver to use the given oci keyring
type WithKeyringOption struct {
	credentials.OCIKeyring
}

func (c WithKeyringOption) ApplyOption(options *Options) {
	options.Resolver = c.OCIKeyring
}

// WithResolver configures a resolver for the oci client
type WithResolver struct {
	Resolver
}

func (c WithResolver) ApplyOption(options *Options) {
	options.Resolver = c.Resolver
}

// WithKnownMediaType adds a known media types to the client
type WithKnownMediaType string

func (c WithKnownMediaType) ApplyOption(options *Options) {
	if options.CustomMediaTypes == nil {
		options.CustomMediaTypes = sets.NewString(string(c))
		return
	}

	options.CustomMediaTypes.Insert(string(c))
}

// AllowPlainHttp sets the allow plain http flag.
type AllowPlainHttp bool

func (c AllowPlainHttp) ApplyOption(options *Options) {
	options.AllowPlainHttp = bool(c)
}

// WithHTTPClient configures the http client.
type WithHTTPClient http.Client

func (c WithHTTPClient) ApplyOption(options *Options) {
	client := http.Client(c)
	options.HTTPClient = &client
}

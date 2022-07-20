// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociclient

import (
	"context"
	"io"
	"net/http"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/credentials"
	"github.com/gardener/component-cli/ociclient/oci"
)

type Client interface {
	Resolver

	// Fetch fetches the blob for the given ocispec Descriptor.
	Fetch(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) error

	// PushBlob uploads the blob for the given ocispec Descriptor to the given ref
	PushBlob(ctx context.Context, ref string, desc ocispecv1.Descriptor, opts ...PushOption) error

	// GetRawManifest returns the raw manifest for a reference.
	// The returned manifest can either be single arch or multi arch (image index/manifest list)
	GetRawManifest(ctx context.Context, ref string) (ocispecv1.Descriptor, []byte, error)

	// PushRawManifest uploads the given raw manifest to the given reference.
	// If the manifest is multi arch (image index/manifest list), only the multi arch manifest is pushed.
	// The referenced single arch manifests must be pushed individiually before.
	PushRawManifest(ctx context.Context, ref string, desc ocispecv1.Descriptor, rawManifest []byte, opts ...PushOption) error

	// GetManifest returns the ocispec Manifest for a reference
	// Deprecated: Please prefer GetRawManifest instead
	GetManifest(ctx context.Context, ref string) (*ocispecv1.Manifest, error)

	// PushManifest uploads the given Manifest to the given reference.
	// Deprecated: Please prefer PushRawManifest instead
	PushManifest(ctx context.Context, ref string, manifest *ocispecv1.Manifest, opts ...PushOption) error

	// GetOCIArtifact returns an OCIArtifact for a reference.
	// Deprecated: Please prefer GetRawManifest instead
	GetOCIArtifact(ctx context.Context, ref string) (*oci.Artifact, error)

	// PushOCIArtifact uploads the given OCIArtifact to the given ref.
	// Deprecated: Please prefer PushRawManifest instead
	PushOCIArtifact(ctx context.Context, ref string, artifact *oci.Artifact, opts ...PushOption) error
}

// ExtendedClient defines an oci client with extended functionality that may not work with all registries.
type ExtendedClient interface {
	Client
	// ListTags returns a list of all tags of the given ref.
	ListTags(ctx context.Context, ref string) ([]string, error)
	// ListRepositories lists all repositories for the given registry host.
	ListRepositories(ctx context.Context, registryHost string) ([]string, error)
}

// Resolver provides remotes based on a locator.
type Resolver interface {
	// Resolve attempts to resolve the reference into a name and descriptor.
	//
	// The argument `ref` should be a scheme-less URI representing the remote.
	// Structurally, it has a host and path. The "host" can be used to directly
	// reference a specific host or be matched against a specific handler.
	//
	// The returned name should be used to identify the referenced entity.
	// Depending on the remote namespace, this may be immutable or mutable.
	// While the name may differ from ref, it should itself be a valid ref.
	//
	// If the resolution fails, an error will be returned.
	Resolve(ctx context.Context, ref string) (name string, desc ocispecv1.Descriptor, err error)
}

// Store describes a store that returns a io reader for a descriptor
type Store interface {
	Get(desc ocispecv1.Descriptor) (io.ReadCloser, error)
}

// PushOption is the interface to specify different cache options
type PushOption interface {
	ApplyPushOption(options *PushOptions)
}

// PushOptions contains all oci push options.
type PushOptions struct {
	// Store is the oci cache to be used by the client
	Store Store
}

// ApplyOptions applies the given list options on these options,
// and then returns itself (for convenient chaining).
func (o *PushOptions) ApplyOptions(opts []PushOption) *PushOptions {
	for _, opt := range opts {
		if opt != nil {
			opt.ApplyPushOption(o)
		}
	}
	return o
}

// WithStore configures a store for the oci push.
func WithStore(store Store) WithStoreOption {
	return WithStoreOption{
		Store: store,
	}
}

// WithStoreOption configures a cache for the oci client
type WithStoreOption struct {
	Store
}

func (c WithStoreOption) ApplyPushOption(options *PushOptions) {
	options.Store = c.Store
}

// Options contains all client options to configure the oci client.
type Options struct {
	// Paths configures local paths to search for docker configuration files
	Paths []string

	// AllowPlainHttp allows the fallback to http if https is not supported by the registry.
	AllowPlainHttp bool

	// Keyring sets the used keyring.
	// A default keyring will be created if not given.
	Keyring credentials.OCIKeyring

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
func WithCache(c cache.Cache) WithCacheOption {
	return WithCacheOption{
		Cache: c,
	}
}

// WithCacheOption configures a cache for the oci client
type WithCacheOption struct {
	cache.Cache
}

func (c WithCacheOption) ApplyOption(options *Options) {
	options.Cache = c.Cache
}

func (c WithCacheOption) ApplyPushOption(options *PushOptions) {
	options.Store = c.Cache
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
	options.Keyring = c.OCIKeyring
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

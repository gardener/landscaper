// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/remotes"
	"github.com/go-logr/logr"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/credentials"
)

type client struct {
	log            logr.Logger
	resolver       Resolver
	cache          cache.Cache
	allowPlainHttp bool
	httpClient     *http.Client

	knownMediaTypes sets.String
}

// ResolverWrapperFunc returns a new authenticated resolver.
type ResolverWrapperFunc func(ctx context.Context, ref string, client *http.Client, plainHTTP bool) (remotes.Resolver, error)

func (f ResolverWrapperFunc) Resolver(ctx context.Context, ref string, client *http.Client, plainHTTP bool) (remotes.Resolver, error) {
	return f(ctx, ref, client, plainHTTP)
}

// NewClient creates a new OCI Client.
func NewClient(log logr.Logger, opts ...Option) (Client, error) {
	options := &Options{}
	options.ApplyOptions(opts)

	if options.Resolver == nil {
		resolver, err := credentials.NewBuilder(log.WithName("ociKeyring")).
			FromConfigFiles(options.Paths...).
			Build()
		if err != nil {
			return nil, err
		}
		options.Resolver = resolver
	}

	if options.Cache == nil {
		cacheOpts := make([]cache.Option, 0)
		if options.CacheConfig != nil {
			if len(options.CacheConfig.BasePath) != 0 {
				cacheOpts = append(cacheOpts, cache.WithBasePath(options.CacheConfig.BasePath))
			}
			cacheOpts = append(cacheOpts, cache.WithInMemoryOverlay(options.CacheConfig.InMemoryOverlay))
		}
		c, err := cache.NewCache(log, cacheOpts...)
		if err != nil {
			return nil, err
		}
		options.Cache = c
	}

	if options.HTTPClient == nil {
		options.HTTPClient = http.DefaultClient
	}

	return &client{
		log:             log,
		allowPlainHttp:  options.AllowPlainHttp,
		httpClient:      options.HTTPClient,
		resolver:        options.Resolver,
		cache:           options.Cache,
		knownMediaTypes: DefaultKnownMediaTypes.Union(options.CustomMediaTypes),
	}, nil
}

func (c *client) InjectCache(cache cache.Cache) error {
	c.cache = cache
	return nil
}

func (c *client) GetManifest(ctx context.Context, ref string) (*ocispecv1.Manifest, error) {
	resolver, err := c.resolver.Resolver(ctx, ref, c.httpClient, c.allowPlainHttp)
	if err != nil {
		return nil, err
	}
	_, desc, err := resolver.Resolve(ctx, ref)
	if err != nil {
		return nil, err
	}

	data := bytes.NewBuffer([]byte{})
	if err := c.Fetch(ctx, ref, desc, data); err != nil {
		return nil, err
	}

	var manifest ocispecv1.Manifest
	if err := json.Unmarshal(data.Bytes(), &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (c *client) Fetch(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) error {
	reader, err := c.getFetchReader(ctx, ref, desc)
	if err != nil {
		return err
	}
	defer func() {
		if err := reader.Close(); err != nil {
			c.log.Error(err, "failed closing reader", "ref", ref)
		}
	}()

	if _, err := io.Copy(writer, reader); err != nil {
		return err
	}
	return nil
}

func (c *client) getFetchReader(ctx context.Context, ref string, desc ocispecv1.Descriptor) (io.ReadCloser, error) {
	if c.cache != nil {
		reader, err := c.cache.Get(desc)
		if err != nil && err != cache.ErrNotFound {
			return nil, err
		}
		if err == nil {
			return reader, nil
		}
	}

	resolver, err := c.resolver.Resolver(context.Background(), ref, c.httpClient, c.allowPlainHttp)
	if err != nil {
		return nil, err
	}
	fetcher, err := resolver.Fetcher(ctx, ref)
	if err != nil {
		return nil, err
	}
	reader, err := fetcher.Fetch(ctx, desc)
	if err != nil {
		return nil, err
	}
	// try to cache
	if c.cache != nil {
		if err := c.cache.Add(desc, reader); err != nil {
			// do not throw an error as cache is just an optimization
			c.log.V(5).Info("unable to cache descriptor", "ref", ref, "error", err.Error())
		}
		return c.cache.Get(desc)
	}

	return reader, err
}

func (c *client) PushManifest(ctx context.Context, ref string, manifest *ocispecv1.Manifest) error {
	resolver, err := c.resolver.Resolver(context.Background(), ref, c.httpClient, c.allowPlainHttp)
	if err != nil {
		return err
	}
	pusher, err := resolver.Pusher(ctx, ref)
	if err != nil {
		return err
	}

	// add dummy config if it is not set
	if manifest.Config.Size == 0 {
		dummyConfig := []byte("{}")
		dummyDesc := ocispecv1.Descriptor{
			MediaType: "application/json",
			Digest:    digest.FromBytes(dummyConfig),
			Size:      int64(len(dummyConfig)),
		}
		if err := c.cache.Add(dummyDesc, ioutil.NopCloser(bytes.NewBuffer(dummyConfig))); err != nil {
			return fmt.Errorf("unable to add dummy config to cache: %w", err)
		}
	}
	if err := c.pushContent(ctx, pusher, manifest.Config); err != nil {
		return err
	}

	// last upload all layers
	for _, layer := range manifest.Layers {
		if err := c.pushContent(ctx, pusher, layer); err != nil {
			return err
		}
	}

	desc, err := c.createDescriptorFromManifest(manifest)
	if err != nil {
		return err
	}
	if err := c.pushContent(ctx, pusher, desc); err != nil {
		return err
	}

	return nil
}

func (c *client) createDescriptorFromManifest(manifest *ocispecv1.Manifest) (ocispecv1.Descriptor, error) {
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return ocispecv1.Descriptor{}, err
	}
	manifestDescriptor := ocispecv1.Descriptor{
		MediaType: ocispecv1.MediaTypeImageManifest,
		Digest:    digest.FromBytes(manifestBytes),
		Size:      int64(len(manifestBytes)),
	}

	manifestBuf := bytes.NewBuffer(manifestBytes)
	if err := c.cache.Add(manifestDescriptor, ioutil.NopCloser(manifestBuf)); err != nil {
		return ocispecv1.Descriptor{}, err
	}
	return manifestDescriptor, nil
}

func (c *client) pushContent(ctx context.Context, pusher remotes.Pusher, desc ocispecv1.Descriptor) error {
	if c.cache == nil {
		return errors.New("no cache defined. A cache is needed to upload content.")
	}
	r, err := c.cache.Get(desc)
	if err != nil {
		return err
	}
	defer r.Close()

	knownMediaTypes := append(c.knownMediaTypes.List(), desc.MediaType)
	writer, err := pusher.Push(AddKnownMediaTypesToCtx(ctx, knownMediaTypes), desc)
	if err != nil {
		if errdefs.IsAlreadyExists(err) {
			return nil
		}
		return err
	}
	defer writer.Close()
	return content.Copy(ctx, writer, r, desc.Size, desc.Digest)
}

// AddKnownMediaTypesToCtx adds a list of known media types to the context
func AddKnownMediaTypesToCtx(ctx context.Context, mediaTypes []string) context.Context {
	for _, mediaType := range mediaTypes {
		ctx = remotes.WithMediaTypeKeyPrefix(ctx, mediaType, "custom")
	}
	return ctx
}

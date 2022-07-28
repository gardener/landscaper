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
	"net/url"
	"path"
	"strings"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	containerdlog "github.com/containerd/containerd/log"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	distributionspecv1 "github.com/opencontainers/distribution-spec/specs-go/v1"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/credentials"
	"github.com/gardener/component-cli/ociclient/oci"
	"github.com/gardener/component-cli/pkg/utils"
)

type client struct {
	log            logr.Logger
	cache          cache.Cache
	keychain       credentials.Keyring
	httpClient     *http.Client
	transport      http.RoundTripper
	allowPlainHttp bool
	getHostConfig  docker.RegistryHosts

	knownMediaTypes sets.String
}

// NewClient creates a new OCI Client.
func NewClient(log logr.Logger, opts ...Option) (*client, error) {
	options := &Options{}
	options.ApplyOptions(opts)

	if options.Keyring == nil {
		keyring, err := credentials.NewBuilder(log.WithName("ociKeyring")).
			FromConfigFiles(options.Paths...).
			Build()
		if err != nil {
			return nil, err
		}
		options.Keyring = keyring
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
	trp := options.HTTPClient.Transport
	if trp == nil {
		trp = http.DefaultTransport
	}

	cLogger := logrus.New()
	cLogger.SetLevel(logrus.FatalLevel)
	if log.V(10).Enabled() {
		cLogger.SetLevel(logrus.TraceLevel)
	} else if log.V(7).Enabled() {
		cLogger.SetLevel(logrus.InfoLevel)
	} else if log.V(2).Enabled() {
		cLogger.SetLevel(logrus.ErrorLevel)
	}
	containerdlog.L = logrus.NewEntry(cLogger)

	return &client{
		log:            log,
		keychain:       options.Keyring,
		allowPlainHttp: options.AllowPlainHttp,
		httpClient:     options.HTTPClient,
		transport:      trp,
		cache:          options.Cache,
		getHostConfig: docker.ConfigureDefaultRegistries(
			docker.WithPlainHTTP(func(_ string) (bool, error) {
				return options.AllowPlainHttp, nil
			}),
		),
		knownMediaTypes: DefaultKnownMediaTypes.Union(options.CustomMediaTypes),
	}, nil
}

func (c *client) InjectCache(cache cache.Cache) error {
	c.cache = cache
	return nil
}

func (c *client) Resolve(ctx context.Context, ref string) (name string, desc ocispecv1.Descriptor, err error) {
	refspec, err := oci.ParseRef(ref)
	if err != nil {
		return "", ocispecv1.Descriptor{}, fmt.Errorf("unable to parse ref: %w", err)
	}
	ref = refspec.String()

	resolver, err := c.getResolverForRef(ctx, ref, transport.PullScope)
	if err != nil {
		return "", ocispecv1.Descriptor{}, err
	}
	return resolver.Resolve(ctx, ref)
}

func (c *client) GetOCIArtifact(ctx context.Context, ref string) (*oci.Artifact, error) {
	refspec, err := oci.ParseRef(ref)
	if err != nil {
		return nil, fmt.Errorf("unable to parse ref: %w", err)
	}
	ref = refspec.String()

	resolver, err := c.getResolverForRef(ctx, ref, transport.PullScope)
	if err != nil {
		return nil, err
	}
	_, desc, err := resolver.Resolve(ctx, ref)
	if err != nil {
		return nil, err
	}

	if desc.MediaType == MediaTypeDockerV2Schema1Manifest || desc.MediaType == MediaTypeDockerV2Schema1SignedManifest {
		c.log.V(7).Info("found v1 manifest -> convert to v2")
		convertedManifestDesc, err := ConvertV1ManifestToV2(ctx, c, c.cache, ref, desc)
		if err != nil {
			return nil, fmt.Errorf("unable to convert v1 manifest to v2: %w", err)
		}
		desc = convertedManifestDesc
	}

	data := bytes.NewBuffer([]byte{})
	if err := c.Fetch(ctx, ref, desc, data); err != nil {
		return nil, err
	}

	if IsMultiArchImage(desc.MediaType) {
		var index ocispecv1.Index
		if err := json.Unmarshal(data.Bytes(), &index); err != nil {
			return nil, err
		}

		i := oci.Index{
			Manifests:   []*oci.Manifest{},
			Annotations: index.Annotations,
		}

		indexArtifact, err := oci.NewIndexArtifact(&i)
		if err != nil {
			return nil, err
		}

		for _, mdesc := range index.Manifests {
			data := bytes.NewBuffer([]byte{})
			if err := c.Fetch(ctx, ref, mdesc, data); err != nil {
				return nil, err
			}

			var manifest ocispecv1.Manifest
			if err := json.Unmarshal(data.Bytes(), &manifest); err != nil {
				return nil, err
			}

			m := oci.Manifest{
				Descriptor: mdesc,
				Data:       &manifest,
			}

			indexArtifact.GetIndex().Manifests = append(indexArtifact.GetIndex().Manifests, &m)
		}

		return indexArtifact, nil
	} else if IsSingleArchImage(desc.MediaType) {
		var manifest ocispecv1.Manifest
		if err := json.Unmarshal(data.Bytes(), &manifest); err != nil {
			return nil, err
		}

		m := oci.Manifest{
			Descriptor: desc,
			Data:       &manifest,
		}

		manifestArtifact, err := oci.NewManifestArtifact(&m)
		if err != nil {
			return nil, err
		}

		return manifestArtifact, nil
	}

	return nil, fmt.Errorf("unable to handle mediatype: %s", desc.MediaType)
}

func (c *client) PushOCIArtifact(ctx context.Context, ref string, artifact *oci.Artifact, options ...PushOption) error {
	refspec, err := oci.ParseRef(ref)
	if err != nil {
		return fmt.Errorf("unable to parse ref: %w", err)
	}
	ref = refspec.String()

	opts := &PushOptions{}
	opts.Store = c.cache
	opts.ApplyOptions(options)

	tempCache := c.cache
	if tempCache == nil {
		tempCache = cache.NewInMemoryCache()
	}

	resolver, err := c.getResolverForRef(ctx, ref, transport.PushScope)
	if err != nil {
		return err
	}
	pusher, err := resolver.Pusher(ctx, ref)
	if err != nil {
		return err
	}

	if artifact.IsManifest() {
		_, err := c.pushManifest(ctx, artifact.GetManifest().Data, pusher, tempCache, opts)
		return err
	} else if artifact.IsIndex() {
		return c.pushImageIndex(ctx, artifact.GetIndex(), pusher, tempCache, opts)
	} else {
		// execution of this code should never happen
		// the oci artifact should always be of type manifest or index
		marshaledArtifact, err := artifact.MarshalJSON()
		if err != nil {
			c.log.Error(err, "unable to marshal oci artifact")
		}
		panic(fmt.Errorf("invalid oci artifact: %s", utils.SafeConvert(marshaledArtifact)))
	}
}

func (c *client) PushBlob(ctx context.Context, ref string, desc ocispecv1.Descriptor, options ...PushOption) error {
	refspec, err := oci.ParseRef(ref)
	if err != nil {
		return fmt.Errorf("unable to parse ref: %w", err)
	}
	ref = refspec.String()

	opts := &PushOptions{}
	opts.Store = c.cache
	opts.ApplyOptions(options)

	resolver, err := c.getResolverForRef(ctx, ref, transport.PushScope)
	if err != nil {
		return err
	}
	pusher, err := resolver.Pusher(ctx, ref)
	if err != nil {
		return err
	}

	if err := c.pushContent(ctx, opts.Store, pusher, desc); err != nil {
		return err
	}

	return nil
}

func (c *client) PushRawManifest(ctx context.Context, ref string, desc ocispecv1.Descriptor, rawManifest []byte, options ...PushOption) error {
	if !IsSingleArchImage(desc.MediaType) && !IsMultiArchImage(desc.MediaType) {
		return fmt.Errorf("media type is not an image manifest or image index: %s", desc.MediaType)
	}

	tempCache := c.cache
	if tempCache == nil {
		tempCache = cache.NewInMemoryCache()
	}

	opts := &PushOptions{}
	opts.ApplyOptions(options)
	if opts.Store == nil {
		opts.ApplyOptions([]PushOption{WithStore(tempCache)})
	}

	resolver, err := c.getResolverForRef(ctx, ref, transport.PushScope)
	if err != nil {
		return err
	}

	pusher, err := resolver.Pusher(ctx, ref)
	if err != nil {
		return err
	}

	if IsSingleArchImage(desc.MediaType) {
		manifest := ocispecv1.Manifest{}
		if err := json.Unmarshal(rawManifest, &manifest); err != nil {
			return fmt.Errorf("unable to unmarshal manifest: %w", err)
		}

		// add dummy config if it is not set
		if manifest.Config.Size == 0 {
			dummyConfig := []byte("{}")
			dummyDesc := ocispecv1.Descriptor{
				MediaType: "application/json",
				Digest:    digest.FromBytes(dummyConfig),
				Size:      int64(len(dummyConfig)),
			}
			if err := tempCache.Add(dummyDesc, ioutil.NopCloser(bytes.NewBuffer(dummyConfig))); err != nil {
				return fmt.Errorf("unable to add dummy config to cache: %w", err)
			}
			if err := c.pushContent(ctx, tempCache, pusher, dummyDesc); err != nil {
				return fmt.Errorf("unable to push dummy config: %w", err)
			}
		} else {
			if err := c.pushContent(ctx, opts.Store, pusher, manifest.Config); err != nil {
				return fmt.Errorf("unable to push config: %w", err)
			}
		}

		for _, layerDesc := range manifest.Layers {
			if err := c.pushContent(ctx, opts.Store, pusher, layerDesc); err != nil {
				return fmt.Errorf("unable to push layer: %w", err)
			}
		}
	}

	if err := tempCache.Add(desc, ioutil.NopCloser(bytes.NewBuffer(rawManifest))); err != nil {
		return fmt.Errorf("unable to add manifest to cache: %w", err)
	}

	if err := c.pushContent(ctx, tempCache, pusher, desc); err != nil {
		return fmt.Errorf("unable to push manifest: %w", err)
	}

	return nil
}

func (c *client) GetRawManifest(ctx context.Context, ref string) (ocispecv1.Descriptor, []byte, error) {
	refspec, err := oci.ParseRef(ref)
	if err != nil {
		return ocispecv1.Descriptor{}, nil, fmt.Errorf("unable to parse ref: %w", err)
	}
	ref = refspec.String()

	resolver, err := c.getResolverForRef(ctx, ref, transport.PullScope)
	if err != nil {
		return ocispecv1.Descriptor{}, nil, err
	}
	_, desc, err := resolver.Resolve(ctx, ref)
	if err != nil {
		return ocispecv1.Descriptor{}, nil, err
	}

	if desc.MediaType == MediaTypeDockerV2Schema1Manifest || desc.MediaType == MediaTypeDockerV2Schema1SignedManifest {
		c.log.V(7).Info("found v1 manifest -> convert to v2")
		convertedManifestDesc, err := ConvertV1ManifestToV2(ctx, c, c.cache, ref, desc)
		if err != nil {
			return ocispecv1.Descriptor{}, nil, fmt.Errorf("unable to convert v1 manifest to v2: %w", err)
		}
		desc = convertedManifestDesc
	}

	if !IsSingleArchImage(desc.MediaType) && !IsMultiArchImage(desc.MediaType) {
		return ocispecv1.Descriptor{}, nil, fmt.Errorf("media type is not an image manifest or image index: %s", desc.MediaType)
	}

	data := bytes.NewBuffer([]byte{})
	if err := c.Fetch(ctx, ref, desc, data); err != nil {
		return ocispecv1.Descriptor{}, nil, err
	}
	rawManifest := data.Bytes()

	return desc, rawManifest, nil
}

func (c *client) pushManifest(ctx context.Context, manifest *ocispecv1.Manifest, pusher remotes.Pusher, cache cache.Cache, opts *PushOptions) (ocispecv1.Descriptor, error) {
	// add dummy config if it is not set
	if manifest.Config.Size == 0 {
		dummyConfig := []byte("{}")
		dummyDesc := ocispecv1.Descriptor{
			MediaType: "application/json",
			Digest:    digest.FromBytes(dummyConfig),
			Size:      int64(len(dummyConfig)),
		}
		if err := cache.Add(dummyDesc, ioutil.NopCloser(bytes.NewBuffer(dummyConfig))); err != nil {
			return ocispecv1.Descriptor{}, fmt.Errorf("unable to add dummy config to cache: %w", err)
		}
		if err := c.pushContent(ctx, cache, pusher, dummyDesc); err != nil {
			return ocispecv1.Descriptor{}, fmt.Errorf("unable to push dummy config: %w", err)
		}
	} else {
		if err := c.pushContent(ctx, opts.Store, pusher, manifest.Config); err != nil {
			return ocispecv1.Descriptor{}, fmt.Errorf("unable to push config: %w", err)
		}
	}

	// last upload all layers
	for _, layer := range manifest.Layers {
		if err := c.pushContent(ctx, opts.Store, pusher, layer); err != nil {
			return ocispecv1.Descriptor{}, fmt.Errorf("unable to push layer: %w", err)
		}
	}

	manifestDesc, err := CreateDescriptorFromManifest(manifest)
	if err != nil {
		return ocispecv1.Descriptor{}, fmt.Errorf("unable to create manifest descriptor: %w", err)
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return ocispecv1.Descriptor{}, fmt.Errorf("unable to marshal manifest: %w", err)
	}

	if err := cache.Add(manifestDesc, ioutil.NopCloser(bytes.NewBuffer(manifestBytes))); err != nil {
		return ocispecv1.Descriptor{}, fmt.Errorf("unable to add manifest to cache: %w", err)
	}

	if err := c.pushContent(ctx, cache, pusher, manifestDesc); err != nil {
		return ocispecv1.Descriptor{}, fmt.Errorf("unable to push manifest: %w", err)
	}

	return manifestDesc, nil
}

func (c *client) pushImageIndex(ctx context.Context, indexArtifact *oci.Index, pusher remotes.Pusher, cache cache.Cache, opts *PushOptions) error {
	manifestDescs := []ocispecv1.Descriptor{}
	for _, manifest := range indexArtifact.Manifests {
		mdesc, err := c.pushManifest(ctx, manifest.Data, pusher, cache, opts)
		if err != nil {
			return fmt.Errorf("unable to upload manifest: %w", err)
		}
		mdesc.Platform = manifest.Descriptor.Platform
		mdesc.Annotations = manifest.Descriptor.Annotations
		manifestDescs = append(manifestDescs, mdesc)
	}

	index := ocispecv1.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		Manifests:   manifestDescs,
		Annotations: indexArtifact.Annotations,
	}

	indexBytes, err := json.Marshal(index)
	if err != nil {
		return err
	}
	indexDescriptor := ocispecv1.Descriptor{
		MediaType: ocispecv1.MediaTypeImageIndex,
		Digest:    digest.FromBytes(indexBytes),
		Size:      int64(len(indexBytes)),
	}

	manifestBuf := bytes.NewBuffer(indexBytes)
	if err := cache.Add(indexDescriptor, ioutil.NopCloser(manifestBuf)); err != nil {
		return err
	}

	if err := c.pushContent(ctx, cache, pusher, indexDescriptor); err != nil {
		return fmt.Errorf("unable to push image index: %w", err)
	}

	return nil
}

func (c *client) GetManifest(ctx context.Context, ref string) (*ocispecv1.Manifest, error) {
	desc, rawManifest, err := c.GetRawManifest(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("unable to get manifest: %w", err)
	}

	if desc.MediaType != ocispecv1.MediaTypeImageManifest && desc.MediaType != images.MediaTypeDockerSchema2Manifest {
		return nil, fmt.Errorf("media type is not an image manifest: %s", desc.MediaType)
	}

	var manifest ocispecv1.Manifest
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		return nil, fmt.Errorf("unable to unmarshal manifest: %w", err)
	}

	return &manifest, nil
}

func (c *client) Fetch(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) error {
	refspec, err := oci.ParseRef(ref)
	if err != nil {
		return fmt.Errorf("unable to parse ref: %w", err)
	}
	ref = refspec.String()

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

	resolver, err := c.getResolverForRef(ctx, ref, transport.PullScope)
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
			c.log.V(2).Info("unable to cache descriptor", "ref", ref, "error", err.Error())
			if err = reader.Close(); err != nil {
				c.log.V(2).Info("unable to close reader", "ref", ref, "error", err.Error())
			}
			return fetcher.Fetch(ctx, desc)
		}
		return c.cache.Get(desc)
	}

	return reader, err
}

func (c *client) PushManifest(ctx context.Context, ref string, manifest *ocispecv1.Manifest, options ...PushOption) error {
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("unable to marshal manifest: %w", err)
	}

	desc := ocispecv1.Descriptor{
		MediaType:   ocispecv1.MediaTypeImageManifest,
		Digest:      digest.FromBytes(manifestBytes),
		Size:        int64(len(manifestBytes)),
		Annotations: manifest.Annotations,
	}

	return c.PushRawManifest(ctx, ref, desc, manifestBytes, options...)
}

func (c *client) getHttpClient() *http.Client {
	return &http.Client{
		Transport:     c.httpClient.Transport,
		CheckRedirect: c.httpClient.CheckRedirect,
		Jar:           c.httpClient.Jar,
		Timeout:       c.httpClient.Timeout,
	}
}

// getRefParserOptions returns the options for reference parsing
func (c *client) getRefParserOptions(ref string) ([]name.Option, error) {
	hosts, err := c.getHostConfig(ref)
	if err != nil {
		return nil, fmt.Errorf("unable to find registry host: %w", err)
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no host configuration found: %w", err)
	}

	hostConfig := hosts[0]
	var options []name.Option
	if hostConfig.Scheme == "http" {
		options = []name.Option{
			name.Insecure,
		}
	}
	return options, nil
}

// getTransportForRef returns the authenticated transport for a reference.
func (c *client) getTransportForRef(ctx context.Context, ref string, scopes ...string) (http.RoundTripper, error) {
	parseOptions, err := c.getRefParserOptions(ref)
	if err != nil {
		return nil, fmt.Errorf("unable to get ref parser options: %w", err)
	}

	repo, err := name.ParseReference(ref, parseOptions...)
	if err != nil {
		return nil, fmt.Errorf("unable to parse ref: %w", err)
	}

	auth, err := c.keychain.ResolveWithContext(ctx, repo.Context())
	if err != nil {
		return nil, fmt.Errorf("unable to get authentication: %w", err)
	}

	for i, scope := range scopes {
		scopes[i] = repo.Scope(scope)
	}
	trp, err := transport.NewWithContext(ctx, repo.Context().Registry, auth, c.transport, scopes)
	if err != nil {
		return nil, fmt.Errorf("unable to create transport: %w", err)
	}
	return trp, nil
}

// getResolverForRef returns the authenticated resolver for a reference.
func (c *client) getResolverForRef(ctx context.Context, ref string, scopes ...string) (remotes.Resolver, error) {
	trp, err := c.getTransportForRef(ctx, ref, scopes...)
	if err != nil {
		return nil, fmt.Errorf("unable to create transport: %w", err)
	}
	httpClient := c.getHttpClient()
	httpClient.Transport = trp
	return docker.NewResolver(docker.ResolverOptions{
		Client: httpClient,
	}), nil
}

// ListTags lists all tags for a given ref.
// Implements the distribution spec defined in https://github.com/opencontainers/distribution-spec/blob/main/spec.md#api.
func (c *client) ListTags(ctx context.Context, ref string) ([]string, error) {
	refspec, err := oci.ParseRef(ref)
	if err != nil {
		return nil, fmt.Errorf("unable to parse ref: %w", err)
	}
	hosts, err := c.getHostConfig(refspec.Host)
	if err != nil {
		return nil, fmt.Errorf("unable to find registry host: %w", err)
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no host configuration found: %w", err)
	}
	hostConfig := hosts[0]

	trp, err := c.getTransportForRef(ctx, ref, transport.PullScope)
	if err != nil {
		return nil, fmt.Errorf("unable to create transport: %w", err)
	}
	httpClient := c.getHttpClient()
	httpClient.Transport = trp

	u := &url.URL{
		Scheme: hostConfig.Scheme,
		Host:   hostConfig.Host,
		Path:   path.Join(hostConfig.Path, refspec.Repository, "tags", "list"),
		// ECR returns an error if n > 1000:
		// https://github.com/google/go-containerregistry/issues/681
		RawQuery: "n=1000",
	}

	var tags []string
	err = doRequestWithPaging(ctx, u, func(ctx context.Context, u *url.URL) (*http.Response, error) {
		resp, err := c.doRequest(ctx, httpClient, u)
		if err != nil {
			return nil, err
		}

		var data bytes.Buffer
		if _, err := io.Copy(&data, resp.Body); err != nil {
			return nil, fmt.Errorf("unable to read response body: %w", err)
		}
		if err := resp.Body.Close(); err != nil {
			return nil, fmt.Errorf("unbale to close body reader: %w", err)
		}

		tagList := &distributionspecv1.TagList{}
		if err := json.Unmarshal(data.Bytes(), tagList); err != nil {
			return nil, fmt.Errorf("unable to decode tagList list: %w", err)
		}
		tags = append(tags, tagList.Tags...)
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// ListRepositories lists all repositories for the given registry host.
func (c *client) ListRepositories(ctx context.Context, ref string) ([]string, error) {
	parseOptions, err := c.getRefParserOptions(ref)
	if err != nil {
		return nil, fmt.Errorf("unable to get ref parser options: %w", err)
	}

	repo, err := name.ParseReference(ref, parseOptions...)
	if err != nil {
		return nil, fmt.Errorf("unable to parse ref: %w", err)
	}

	auth, err := c.keychain.ResolveWithContext(ctx, repo.Context())
	if err != nil {
		return nil, fmt.Errorf("unable to get authentication: %w", err)
	}

	trp, err := transport.New(repo.Context().Registry, auth, c.transport, []string{"registry:catalog:*"})
	if err != nil {
		return nil, fmt.Errorf("unable to create transport: %w", err)
	}
	httpClient := c.getHttpClient()
	httpClient.Transport = trp

	hosts, err := c.getHostConfig(repo.Context().RegistryStr())
	if err != nil {
		return nil, fmt.Errorf("unable to find registry host: %w", err)
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no host configuration found: %w", err)
	}
	hostConfig := hosts[0]

	u := &url.URL{
		Scheme: hostConfig.Scheme,
		Host:   hostConfig.Host,
		Path:   path.Join(hostConfig.Path, "_catalog"),
		// ECR returns an error if n > 1000:
		// https://github.com/google/go-containerregistry/issues/681
		RawQuery: "n=1000",
	}

	// parse registry to also support more specific credentials e.g. for gcr with gcr.io/my-project
	refspec, err := oci.ParseRef(ref)
	if err != nil {
		return nil, fmt.Errorf("unable to parse ref: %w", err)
	}
	repositories := make([]string, 0)
	err = doRequestWithPaging(ctx, u, func(ctx context.Context, u *url.URL) (*http.Response, error) {
		resp, err := c.doRequest(ctx, httpClient, u)
		if err != nil {
			return nil, err
		}

		var data bytes.Buffer
		if _, err := io.Copy(&data, resp.Body); err != nil {
			return nil, fmt.Errorf("unable to read response body: %w", err)
		}
		if err := resp.Body.Close(); err != nil {
			return nil, fmt.Errorf("unbale to close body reader: %w", err)
		}

		repositoryList := &distributionspecv1.RepositoryList{}
		if err := json.Unmarshal(data.Bytes(), repositoryList); err != nil {
			return nil, fmt.Errorf("unable to decode repository list: %w", err)
		}

		// the registry by default returns all repositories
		// lets filter the results if a repository path is provided
		if len(refspec.Repository) != 0 {
			name := refspec.Name()
			prefix := refspec.Repository
			for _, repo := range repositoryList.Repositories {
				if strings.HasPrefix(repo, prefix) || strings.HasPrefix(repo, name) {
					r := refspec.DeepCopy()
					r.Repository = repo
					repositories = append(repositories, r.Name())
				}
			}
			return resp, nil
		}
		repositories = append(repositories, repositoryList.Repositories...)
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	return repositories, nil
}

// doRequest does a authenticated request to the given oci registry
func (c *client) doRequest(ctx context.Context, httpClient *http.Client, url *url.URL) (*http.Response, error) {
	req := &http.Request{
		Method: http.MethodGet,
		URL:    url,
		Header: make(http.Header),
	}
	resp, err := httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("unable to get %q: %w", url.String(), err)
	}

	if resp.StatusCode != 200 {
		var data bytes.Buffer
		if _, err := io.Copy(&data, resp.Body); err != nil {
			return nil, fmt.Errorf("unable to read response body: %w", err)
		}
		if err := resp.Body.Close(); err != nil {
			return nil, fmt.Errorf("unbale to close body reader: %w", err)
		}
		// read error response
		errRes := &distributionspecv1.ErrorResponse{}
		if err := json.Unmarshal(data.Bytes(), errRes); err != nil {
			return nil, fmt.Errorf("unable to decode error response: %w", err)
		}
		errMsg := ""
		for _, err := range errRes.Detail() {
			errMsg = errMsg + fmt.Sprintf("; Code: %q, Message: %q, Detail: %q", err.Code, err.Message, err.Detail)
		}
		return nil, fmt.Errorf("error during list call to registry with status code %d: %v", resp.StatusCode, errMsg)
	}
	return resp, nil
}

type pagingFunc func(ctx context.Context, url *url.URL) (*http.Response, error)

// doRequestWithPaging implements the oci spec paging for repositories and tags.
func doRequestWithPaging(ctx context.Context, u *url.URL, pFunc pagingFunc) error {
	nextUrl := u
	for {
		resp, err := pFunc(ctx, nextUrl)
		if err != nil {
			return err
		}

		// parse next url
		link := resp.Header.Get("Link")
		if len(link) == 0 {
			return nil
		}
		splitLink := strings.Split(link, ";")
		next := strings.NewReplacer(">", "", "<", "").Replace(splitLink[0])
		nextUrl, err = url.Parse(next)
		if err != nil {
			return fmt.Errorf("unable to parse next url %q: %w", next, err)
		}
	}
}

func CreateDescriptorFromManifest(manifest *ocispecv1.Manifest) (ocispecv1.Descriptor, error) {
	if manifest.SchemaVersion == 0 {
		manifest.SchemaVersion = 2
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return ocispecv1.Descriptor{}, err
	}
	manifestDescriptor := ocispecv1.Descriptor{
		MediaType: ocispecv1.MediaTypeImageManifest,
		Digest:    digest.FromBytes(manifestBytes),
		Size:      int64(len(manifestBytes)),
	}

	return manifestDescriptor, nil
}

func (c *client) pushContent(ctx context.Context, store Store, pusher remotes.Pusher, desc ocispecv1.Descriptor) error {
	if store == nil {
		return errors.New("a store is needed to upload content but no store has been defined")
	}
	r, err := store.Get(desc)
	if err != nil {
		return err
	}
	defer r.Close()

	writer, err := pusher.Push(AddKnownMediaTypesToCtx(ctx, []string{desc.MediaType}), desc)
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

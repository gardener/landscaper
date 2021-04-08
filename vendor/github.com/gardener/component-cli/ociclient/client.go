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
	containerdlog "github.com/containerd/containerd/log"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/go-logr/logr"
	distributionspecv1 "github.com/opencontainers/distribution-spec/specs-go/v1"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/credentials"
	"github.com/gardener/component-cli/ociclient/oci"
)

type client struct {
	log                  logr.Logger
	resolver             Resolver
	cache                cache.Cache
	allowPlainHttp       bool
	httpClient           *http.Client
	defaultRegistryHosts docker.RegistryHosts

	knownMediaTypes sets.String
}

// ResolverWrapperFunc returns a new authenticated resolver.
type ResolverWrapperFunc func(ctx context.Context, ref string, client *http.Client, plainHTTP bool) (remotes.Resolver, error)

func (f ResolverWrapperFunc) Resolver(ctx context.Context, ref string, client *http.Client, plainHTTP bool) (remotes.Resolver, error) {
	return f(ctx, ref, client, plainHTTP)
}

// NewClient creates a new OCI Client.
func NewClient(log logr.Logger, opts ...Option) (*client, error) {
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

	authorizer := docker.NewDockerAuthorizer(
		docker.WithAuthClient(options.HTTPClient),
		docker.WithAuthCreds(options.Resolver.GetCredentials))

	cLogger := logrus.New()
	if log.V(5).Enabled() {
		cLogger.SetLevel(logrus.DebugLevel)
	}
	if log.V(7).Enabled() {
		cLogger.SetLevel(logrus.TraceLevel)
	}
	containerdlog.L = logrus.NewEntry(cLogger)

	return &client{
		log:            log,
		allowPlainHttp: options.AllowPlainHttp,
		httpClient:     options.HTTPClient,
		resolver:       options.Resolver,
		cache:          options.Cache,
		defaultRegistryHosts: docker.ConfigureDefaultRegistries(
			docker.WithPlainHTTP(func(_ string) (bool, error) {
				return options.AllowPlainHttp, nil
			}),
			docker.WithAuthorizer(authorizer),
			docker.WithClient(options.HTTPClient),
		),
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

// ListTags lists all tags for a given ref.
// Implements the distribution spec defined in https://github.com/opencontainers/distribution-spec/blob/main/spec.md#api.
// todo: do paging
func (c *client) ListTags(ctx context.Context, ref string) ([]string, error) {
	refspec, err := oci.ParseRef(ref)
	if err != nil {
		return nil, fmt.Errorf("unable to parse reference: %w", err)
	}
	hosts, err := c.defaultRegistryHosts(refspec.Host)
	if err != nil {
		return nil, fmt.Errorf("unable to find registry host: %w", err)
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no host configuration found: %w", err)
	}
	hostConfig := hosts[0]
	hostConfig.Authorizer = c.getAuthorizerForRef(ref)

	u := &url.URL{
		Scheme: hostConfig.Scheme,
		Host:   hostConfig.Host,
		Path:   path.Join(hostConfig.Path, refspec.Repository, "tags", "list"),
		// ECR returns an error if n > 1000:
		// https://github.com/google/go-containerregistry/issues/681
		RawQuery: "n=1000",
	}

	var tags []string
	err = doWithPaging(ctx, u, func(ctx context.Context, u *url.URL) (*http.Response, error) {
		resp, err := c.doRequest(ctx, hostConfig, u, "")
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
	// parse registry to also support more specific credentials e.g. for gcr with gcr.io/my-project
	refspec, err := oci.ParseRef(ref)
	if err != nil {
		return nil, fmt.Errorf("unable to parse reference: %w", err)
	}

	hosts, err := c.defaultRegistryHosts(refspec.Host)
	if err != nil {
		return nil, fmt.Errorf("unable to find registry host: %w", err)
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no host configuration found: %w", err)
	}
	hostConfig := hosts[0]
	hostConfig.Authorizer = c.getAuthorizerForRef(ref)

	u := &url.URL{
		Scheme: hostConfig.Scheme,
		Host:   hostConfig.Host,
		Path:   path.Join(hostConfig.Path, "_catalog"),
		// ECR returns an error if n > 1000:
		// https://github.com/google/go-containerregistry/issues/681
		RawQuery: "n=1000",
	}

	repositories := make([]string, 0)
	err = doWithPaging(ctx, u, func(ctx context.Context, u *url.URL) (*http.Response, error) {
		resp, err := c.doRequest(ctx, hostConfig, u, "registry:catalog:*")
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
					repositories = append(repositories, repo)
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
func (c *client) doRequest(ctx context.Context, registry docker.RegistryHost, url *url.URL, defaultScope string) (*http.Response, error) {
	req := &http.Request{
		Method: http.MethodGet,
		URL:    url,
		Header: make(http.Header),
	}
	if err := registry.Authorizer.Authorize(ctx, req); err != nil {
		return nil, fmt.Errorf("unable to authorize call: %w", err)
	}
	resp, err := registry.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to get %q: %w", url.String(), err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		if len(defaultScope) != 0 {
			// inject default scope if not requested by registry
			authHeader := resp.Header.Get("WWW-Authenticate")
			if len(authHeader) != 0 && !strings.Contains(authHeader, "scope") {
				resp.Header.Set("WWW-Authenticate",
					fmt.Sprintf("%s,scope=%q", authHeader, defaultScope))
			}
		}
		// do authorization if 401 is returned and retry the request
		if err := registry.Authorizer.AddResponses(ctx, []*http.Response{resp}); err != nil {
			return nil, fmt.Errorf("unable to authorize call: %w", err)
		}
		if err := registry.Authorizer.Authorize(ctx, req); err != nil {
			return nil, fmt.Errorf("unable to authorize call: %w", err)
		}
		resp, err = registry.Client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("unable to get %q: %w", url.String(), err)
		}
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

func doWithPaging(ctx context.Context, u *url.URL, pFunc pagingFunc) error {
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

func (c *client) getAuthorizerForRef(ref string) docker.Authorizer {
	u, p, err := c.resolver.GetCredentials(ref)
	if err != nil {
		return docker.NewDockerAuthorizer(
			docker.WithAuthClient(c.httpClient),
			docker.WithAuthCreds(c.resolver.GetCredentials))
	}
	return docker.NewDockerAuthorizer(
		docker.WithAuthClient(c.httpClient),
		docker.WithAuthCreds(func(s string) (string, string, error) {
			return u, p, nil
		}))
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

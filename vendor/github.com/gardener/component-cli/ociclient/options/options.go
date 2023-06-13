// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"crypto/tls"
	"fmt"
	"net/http"

	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/pflag"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/credentials"
	"github.com/gardener/component-cli/ociclient/credentials/secretserver"
)

// Options defines a set of options to create a oci client
type Options struct {
	// AllowPlainHttp allows the fallback to http if the oci registry does not support https
	AllowPlainHttp bool
	// SkipTLSVerify specifies if the server's certificate should be checked for validity.
	SkipTLSVerify bool
	// CacheDir defines the oci cache directory
	CacheDir string
	// RegistryConfigPath defines a path to the dockerconfig.json with the oci registry authentication.
	RegistryConfigPath string
	// ConcourseConfigPath is the path to the local concourse config file.
	ConcourseConfigPath string
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	if fs == nil {
		fs = pflag.CommandLine
	}

	fs.BoolVar(&o.AllowPlainHttp, "allow-plain-http", false, "allows the fallback to http if the oci registry does not support https")
	fs.BoolVar(&o.SkipTLSVerify, "insecure-skip-tls-verify", false, "If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure")
	fs.StringVar(&o.RegistryConfigPath, "registry-config", "", "path to the dockerconfig.json with the oci registry authentication information")
	fs.StringVar(&o.ConcourseConfigPath, "cc-config", "", "path to the local concourse config file")
}

// Build builds a new oci client based on the given options
func (o *Options) Build(log logr.Logger, fs vfs.FileSystem) (ociclient.ExtendedClient, cache.Cache, error) {
	cache, err := cache.NewCache(log, cache.WithBasePath(o.CacheDir))
	if err != nil {
		return nil, nil, err
	}

	ociOpts := []ociclient.Option{
		ociclient.WithCache(cache),
		ociclient.WithKnownMediaType(cdoci.ComponentDescriptorConfigMimeType),
		ociclient.WithKnownMediaType(cdoci.ComponentDescriptorTarMimeType),
		ociclient.WithKnownMediaType(cdoci.ComponentDescriptorJSONMimeType),
		ociclient.AllowPlainHttp(o.AllowPlainHttp),
	}

	if o.SkipTLSVerify {
		httpClient := http.Client{
			Transport: http.DefaultTransport,
		}
		httpClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
		ociOpts = append(ociOpts, ociclient.WithHTTPClient(httpClient))
	}

	keyring, err := credentials.NewBuilder(log).WithFS(fs).FromConfigFiles(o.RegistryConfigPath).Build()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create keyring for registry at %q: %w", o.RegistryConfigPath, err)
	}
	ociOpts = append(ociOpts, ociclient.WithKeyring(keyring))

	secretServerKeyring, err := secretserver.New().
		WithLog(log.WithName("secretserver")).
		WithFS(fs).
		FromPath(o.ConcourseConfigPath).
		WithMinPrivileges(secretserver.ReadWrite).
		Build()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get credentials from secret server: %s", err.Error())
	}
	if secretServerKeyring != nil {
		if err := credentials.Merge(keyring, secretServerKeyring); err != nil {
			return nil, nil, err
		}
	}

	ociClient, err := ociclient.NewClient(log, ociOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to build oci client: %w", err)
	}
	return ociClient, cache, nil
}

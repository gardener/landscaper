// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package envtest

import (
	"fmt"
	"net"
	"path/filepath"
)

// RegistryConfiguration contains the docker registry configuration.
// See https://docs.docker.com/registry/configuration/ for more configuration options
type RegistryConfiguration struct {
	Version    string     `json:"version"`
	Storage    Storage    `json:"storage,omitempty"`
	Auth       Auth       `json:"auth,omitempty"`
	HTTPConfig HTTPConfig `json:"http,omitempty"`
}

// Default applies defaults to the registry configuration.
func (cfg *RegistryConfiguration) Default(tmpDir string) error {
	cfg.Version = "0.1"
	if len(cfg.HTTPConfig.Addr) == 0 {
		// try to assign a random port
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
				return fmt.Errorf("failed to listen on a port: %w", err)
			}
		}
		cfg.HTTPConfig.Addr = l.Addr().String()
		if err := l.Close(); err != nil {
			return fmt.Errorf("failed to close temp listener: %w", err)
		}
	}

	if cfg.Storage.Filesystem == nil {
		cfg.Storage.Filesystem = &FilesystemStorage{
			RootDirectory: filepath.Join(tmpDir, "storage"),
			MaxThreads:    100,
		}
	}

	return nil
}

type Storage struct {
	Filesystem *FilesystemStorage `json:"filesystem,omitempty"`
}

type FilesystemStorage struct {
	RootDirectory string `json:"rootdirectory,omitempty"`
	MaxThreads    int    `json:"maxthreads,omitempty"`
}

// Auth describes the authentication mechanism that should be used for the registry.
// https://github.com/distribution/distribution/blob/main/docs/configuration.md#auth
type Auth struct {
	Silly     *SillyAuth     `json:"silly,omitempty"`
	Httpasswd *HttpasswdAuth `json:"htpasswd,omitempty"`
}

// SillyAuth is only appropriate for development.
// It simply checks for the existence of the Authorization header in the HTTP request.
// It does not check the header’s value.
// If the header does not exist, the silly auth responds with a challenge response, echoing back the realm, service, and scope for which access was denied.
type SillyAuth struct {
	Realm string `json:"realm"`
	Path  string `json:"service"`
}

// HttpasswdAuth allows you to configure basic authentication using an Apache htpasswd file.
// The only supported password format is bcrypt.
// Entries with other hash types are ignored.
// The htpasswd file is loaded once, at startup.
// If the file is invalid, the registry will display an error and will not start.
type HttpasswdAuth struct {
	Realm string `json:"realm"`
	Path  string `json:"path"`
}

type HTTPConfig struct {
	// Defaults to localhost:5000
	Addr   string         `json:"addr,omitempty"`
	Prefix string         `json:"prefix,omitempty"`
	Host   string         `json:"host,omitempty"`
	TLS    *HTTPTLSConfig `json:"tls,omitempty"`
}

type HTTPTLSConfig struct {
	// Cert is the path to the certificate file.
	Cert string `json:"certificate,omitempty"`
	// Key is the path to the key file.
	Key string `json:"key,omitempty"`
	// ClientCas are a list of ca file paths.
	ClientCas []string `json:"clientcas"`
}

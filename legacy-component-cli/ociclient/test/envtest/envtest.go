// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package envtest

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
)

type Environment struct {
	RegistryBinaryPath    string
	RegistryConfiguration RegistryConfiguration
	ReadinessTimeout      time.Duration
	Stdout                io.Writer
	Stderr                io.Writer

	// Contains the host information as soon as the registry is started.
	// The host is of the format "ip:port"
	Addr string
	// Transport to communicate with the registry.
	// Includes the trusted ca.
	Transport *http.Transport
	// BasicAuth defines the basic auth credentials to access the registry.
	BasicAuth *BasicAuth

	configDir  string
	configPath string

	cancelCtx context.CancelFunc
	cmd       *exec.Cmd
	mu        *sync.RWMutex
	stopped   bool
	err       error
}

// BasicAuth defines auth credentials that consists of a username and a password.
type BasicAuth struct {
	Username string
	Password string
}

const DefaultRegistryBinaryPath = "./tmp/test/bin/registry"

type Options struct {
	RegistryBinaryPath    string
	RegistryConfiguration RegistryConfiguration
	ReadinessTimeout      *time.Duration
	Stdout                io.Writer
	Stderr                io.Writer
}

func (opts *Options) Default() {
	if len(opts.RegistryBinaryPath) == 0 {
		opts.RegistryBinaryPath = DefaultRegistryBinaryPath
	}
	if opts.ReadinessTimeout == nil {
		d := 30 * time.Second
		opts.ReadinessTimeout = &d
	}
	if opts.Stdout == nil {
		opts.Stdout = bytes.NewBuffer(make([]byte, 0))
	}
	if opts.Stderr == nil {
		opts.Stderr = bytes.NewBuffer(make([]byte, 0))
	}
}

// New creates a new test registry environment
func New(opts Options) *Environment {
	opts.Default()
	return &Environment{
		RegistryBinaryPath:    opts.RegistryBinaryPath,
		RegistryConfiguration: opts.RegistryConfiguration,
		ReadinessTimeout:      *opts.ReadinessTimeout,
		Stdout:                opts.Stdout,
		Stderr:                opts.Stderr,
		mu:                    &sync.RWMutex{},
	}
}

func (e *Environment) Start(ctx context.Context) error {
	ctx, e.cancelCtx = context.WithCancel(ctx)
	if err := e.setup(); err != nil {
		return err
	}
	if err := e.runRegistry(ctx); err != nil {
		return err
	}
	// wait until registry is healthy
	if err := e.WaitForRegistryToBeHealthy(); err != nil {
		if buf, ok := e.Stdout.(*bytes.Buffer); ok {
			fmt.Println("Stdout:")
			fmt.Println(buf.String())
		}
		if buf, ok := e.Stderr.(*bytes.Buffer); ok {
			fmt.Println("Stderr:")
			fmt.Println(buf.String())
		}
		return err
	}
	return nil
}

func (e *Environment) Close() error {
	e.mu.RLock()
	if !e.stopped {
		e.cancelCtx()
	}
	e.mu.RUnlock()
	// wait until the process is stopped.
	for {
		e.mu.RLock()
		stopped := e.stopped
		e.mu.RUnlock()
		if stopped {
			break
		}
		time.Sleep(2 * time.Second)
	}

	if err := os.RemoveAll(e.configDir); err != nil {
		return fmt.Errorf("unable to remove config dir %q: %w", e.configDir, err)
	}
	return e.err
}

// GetConfigFileBytes returns the docker configfile containing the registry auth for the registry.
func (e *Environment) GetConfigFileBytes() ([]byte, error) {
	cf := configfile.ConfigFile{}
	cf.AuthConfigs = map[string]types.AuthConfig{
		e.Addr: {
			Username: e.BasicAuth.Username,
			Password: e.BasicAuth.Password,
		},
	}
	return json.Marshal(cf)
}

// setups creates all necessary files for the registry.
// This includes the configuration and the certificates.
func (e *Environment) setup() error {
	configPath, err := ioutil.TempDir(os.TempDir(), "registry-")
	if err != nil {
		return fmt.Errorf("unable to create temporary config path: %w", err)
	}
	e.configDir = configPath

	if err := e.RegistryConfiguration.Default(configPath); err != nil {
		return err
	}

	if e.RegistryConfiguration.HTTPConfig.TLS == nil {
		// create certificates
		cert, err := GenerateCertificates()
		if err != nil {
			return err
		}
		var (
			caPath   = filepath.Join(e.configDir, "ca.pem")
			certPath = filepath.Join(e.configDir, "cert.pem")
			keyPath  = filepath.Join(e.configDir, "key.pem")
		)
		if err := ioutil.WriteFile(caPath, cert.CA, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write ca to %q: %w", caPath, err)
		}
		if err := ioutil.WriteFile(certPath, cert.Cert, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write ca to %q: %w", certPath, err)
		}
		if err := ioutil.WriteFile(keyPath, cert.Key, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write ca to %q: %w", keyPath, err)
		}

		caCertPool, err := x509.SystemCertPool()
		if err != nil {
			caCertPool = x509.NewCertPool()
		}
		caCertPool.AppendCertsFromPEM(cert.CA)
		e.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		}
		e.RegistryConfiguration.HTTPConfig.TLS = &HTTPTLSConfig{
			Cert: certPath,
			Key:  keyPath,
		}
	}

	if e.RegistryConfiguration.Auth.Httpasswd == nil {
		httpasswdPath := filepath.Join(e.configDir, "httpasswd")
		e.BasicAuth = &BasicAuth{
			Username: "testuser",
			Password: RandString(10),
		}
		if err := ioutil.WriteFile(
			httpasswdPath,
			[]byte(CreateHtpasswd(e.BasicAuth.Username, e.BasicAuth.Password)),
			os.ModePerm); err != nil {
			return fmt.Errorf("unable to write httpasswd file to %q: %w", httpasswdPath, err)
		}
		e.RegistryConfiguration.Auth.Httpasswd = &HttpasswdAuth{
			Realm: "basic-realm",
			Path:  httpasswdPath,
		}
	}

	configBytes, err := json.Marshal(e.RegistryConfiguration)
	if err != nil {
		return fmt.Errorf("unable to marshal registry config: %w", err)
	}
	e.configPath = filepath.Join(e.configDir, "config.json")
	if err := ioutil.WriteFile(e.configPath, configBytes, os.ModePerm); err != nil {
		return fmt.Errorf("unable to write configuration file %q: %w", e.configPath, err)
	}
	return nil
}

func (e *Environment) runRegistry(ctx context.Context) error {
	e.cmd = exec.CommandContext(ctx, e.RegistryBinaryPath, "serve", e.configPath)
	e.cmd.Stdout = e.Stdout
	e.cmd.Stderr = e.Stderr
	if err := e.cmd.Start(); err != nil {
		return fmt.Errorf("unable to start registry: %w", err)
	}
	e.Addr = e.RegistryConfiguration.HTTPConfig.Addr
	go func() {
		defer func() {
			e.mu.Lock()
			e.stopped = true
			e.mu.Unlock()
		}()
		if err := e.cmd.Wait(); err != nil {
			if ctx.Err() == context.Canceled {
				return
			}
			e.err = fmt.Errorf("error while running %q: %w", e.cmd.String(), err)
		}
	}()
	return nil
}

func (e *Environment) WaitForRegistryToBeHealthy() error {
	start := time.Now()
	for {
		if e.err != nil {
			return e.err
		}
		err := e.doHealthCheck()
		if err == nil {
			return nil
		}
		if time.Now().After(start.Add(e.ReadinessTimeout)) {
			return fmt.Errorf("timed out waiting for the registry to become healthy: %w", err)
		}
		time.Sleep(5 * time.Second)
	}
}

func (e *Environment) doHealthCheck() error {
	if len(e.Addr) == 0 {
		return errors.New("no addr to perform a heath check defined")
	}

	client := http.DefaultClient
	client.Transport = e.Transport
	res, err := client.Get("https://" + e.Addr)
	if err != nil {
		return fmt.Errorf("error while doing health check request: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status code %d but expected 200", res.StatusCode)
	}
	return nil
}

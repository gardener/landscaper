// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

// taken from "github.com/containers/image/v5", unfortunately this is private and cannot be used via import

package docker

import (
	"os"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	cliflags "github.com/docker/cli/cli/flags"
	dockerclient "github.com/docker/docker/client"
	"github.com/spf13/pflag"
)

func newDockerClient(dockerhost string) (*dockerclient.Client, error) {
	if dockerhost == "" {
		opts := cliflags.NewClientOptions()
		// set defaults
		opts.SetDefaultOptions(pflag.NewFlagSet("", pflag.ContinueOnError))
		configfile := config.LoadDefaultConfigFile(os.Stderr)
		c, err := command.NewAPIClientFromFlags(opts, configfile)
		if err != nil {
			return nil, err
		}
		return c.(*dockerclient.Client), nil
	}
	c, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithHost(dockerhost))
	if err != nil {
		return nil, err
	}
	url, err := dockerclient.ParseHostURL(dockerhost)
	if err == nil && url.Scheme == "unix" {
		dockerclient.WithScheme(url.Scheme)(c)
	}
	return c, nil
}

/*

const (
	// The default API version to be used in case none is explicitly specified.
	defaultAPIVersion = "1.22"
)


// NewDockerClient initializes a new API client based on the passed SystemContext.
func newDockerClient2(host string) (*dockerclient.Client, error) {
	if host == "" {
		host = dockerclient.DefaultDockerHost
	}

	// Sadly, unix:// sockets don't work transparently with dockerclient.NewClient.
	// They work fine with a nil httpClient; with a non-nil httpClient, the transport’s
	// TLSClientConfig must be nil (or the client will try using HTTPS over the PF_UNIX socket
	// regardless of the values in the *tls.Config), and we would have to call sockets.ConfigureTransport.
	//
	// We don't really want to configure anything for unix:// sockets, so just pass a nil *http.Client.
	//
	// Similarly, if we want to communicate over plain HTTP on a TCP socket, we also need to set
	// TLSClientConfig to nil. This can be achieved by using the form `http://`
	url, err := dockerclient.ParseHostURL(host)
	if err != nil {
		return nil, err
	}
	var httpClient *http.Client
	if url.Scheme != "unix" {
		if url.Scheme == "http" {
			httpClient = httpConfig()
		} else {
			hc, err := tlsConfig(nil)
			if err != nil {
				return nil, err
			}
			httpClient = hc
		}
	}

	return dockerclient.NewClient(host, defaultAPIVersion, httpClient, nil)
}

func tlsConfig(sys *types.SystemContext) (*http.Client, error) {
	options := tlsconfig.Options{}
	if sys != nil && sys.DockerDaemonInsecureSkipTLSVerify {
		options.InsecureSkipVerify = true
	}

	if sys != nil && sys.DockerDaemonCertPath != "" {
		options.CAFile = filepath.Join(sys.DockerDaemonCertPath, "ca.pem")
		options.CertFile = filepath.Join(sys.DockerDaemonCertPath, "cert.pem")
		options.KeyFile = filepath.Join(sys.DockerDaemonCertPath, "key.pem")
	}

	tlsc, err := tlsconfig.Client(options)
	if err != nil {
		return nil, err
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsc,
		},
		CheckRedirect: dockerclient.CheckRedirect,
	}, nil
}

func httpConfig() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: nil,
		},
		CheckRedirect: dockerclient.CheckRedirect,
	}
}
*/

// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"github.com/containerd/containerd/remotes/docker"
)

var (
	ContextWithRepositoryScope           = docker.ContextWithRepositoryScope
	ContextWithAppendPullRepositoryScope = docker.ContextWithAppendPullRepositoryScope
	NewInMemoryTracker                   = docker.NewInMemoryTracker
	NewDockerAuthorizer                  = docker.NewDockerAuthorizer
	WithAuthClient                       = docker.WithAuthClient
	WithAuthHeader                       = docker.WithAuthHeader
	WithAuthCreds                        = docker.WithAuthCreds
)

type (
	Errors            = docker.Errors
	StatusTracker     = docker.StatusTracker
	Status            = docker.Status
	StatusTrackLocker = docker.StatusTrackLocker
)

func ConvertHosts(hosts docker.RegistryHosts) RegistryHosts {
	return func(host string) ([]RegistryHost, error) {
		list, err := hosts(host)
		if err != nil {
			return nil, err
		}
		result := make([]RegistryHost, len(list))
		for i, v := range list {
			result[i] = RegistryHost{
				Client:       v.Client,
				Authorizer:   v.Authorizer,
				Host:         v.Host,
				Scheme:       v.Scheme,
				Path:         v.Path,
				Capabilities: HostCapabilities(v.Capabilities),
				Header:       v.Header,
			}
		}
		return result, nil
	}
}

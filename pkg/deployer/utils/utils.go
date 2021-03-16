// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/credentials"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/vfs"
	corev1 "k8s.io/api/core/v1"
)

// CreateOciClientFromDockerAuthConfig reads registry pull secrets from a directory and builds a new oci client with all found secrets.
func CreateOciClientFromDockerAuthConfig(ctx context.Context, log logr.Logger, fs vfs.FileSystem, registryPullSecretsDir string) (ociclient.Client, error) {
	var authConfig []string
	err := vfs.Walk(fs, registryPullSecretsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Name() != corev1.DockerConfigJsonKey {
			return nil
		}

		authConfig = append(authConfig, path)

		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to add local registry pull authConfig: %w", err)
	}

	keyring, err := credentials.NewBuilder(log).FromConfigFiles(authConfig...).Build()
	if err != nil {
		return nil, err
	}

	ociClient, err := ociclient.NewClient(log, ociclient.WithResolver{Resolver: keyring})
	if err != nil {
		return nil, err
	}

	return ociClient, err
}

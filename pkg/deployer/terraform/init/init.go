// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/gardener/landscaper/pkg/deployer/terraform/providerresolver"
	"github.com/gardener/landscaper/pkg/deployer/utils"
)

// Run runs as init container for the terraform deployer and
// downloads all configured providers.
func Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	opts := &Options{}
	opts.Complete(ctx)
	if err := opts.Validate(); err != nil {
		return err
	}

	return opts.Run(ctx, log, fs)
}

func (o *Options) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	ociClient, err := utils.CreateOciClientFromDockerAuthConfig(ctx, log, fs, o.RegistrySecretBasePath)
	if err != nil {
		return fmt.Errorf("unable to build oci client: %w", err)
	}
	resolver := providerresolver.NewProviderResolver(log, ociClient).
		WithFs(fs).
		ProvidersDir(o.ProvidersDir)
	if err := providerresolver.ResolveProviders(ctx, resolver, o.Configuration.TerraformProviders); err != nil {
		return fmt.Errorf("unable to resolve providers: %w", err)
	}
	return nil
}

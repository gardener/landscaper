// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"fmt"

	"github.com/gardener/component-cli/ociclient"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/osfs"
	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/landscaper/operation"

	"github.com/gardener/component-cli/ociclient/credentials"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/pkg/utils"
)

// SetupRegistries sets up components and blueprints registries for the current reconcile
func (c *Controller) SetupRegistries(ctx context.Context, op *operation.Operation, pullSecrets []lsv1alpha1.ObjectReference, installation *lsv1alpha1.Installation) error {
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	// resolve all pull secrets
	secrets, err := c.resolveSecrets(ctx, pullSecrets)
	if err != nil {
		return err
	}

	compRegistry, err := componentsregistry.New(c.SharedCache)
	if err != nil {
		return fmt.Errorf("unable to create component registry manager: %w", err)
	}
	if c.LsConfig.Registry.Local != nil {
		componentsOCIRegistry, err := componentsregistry.NewLocalClient(c.LsConfig.Registry.Local.RootPath)
		if err != nil {
			return err
		}
		if err := compRegistry.Set(componentsOCIRegistry); err != nil {
			return err
		}
	}

	// always add an oci client to support unauthenticated requests
	ociConfigFiles := make([]string, 0)
	if c.LsConfig.Registry.OCI != nil {
		ociConfigFiles = c.LsConfig.Registry.OCI.ConfigFiles
	}
	ociKeyring, err := credentials.NewBuilder(logger.Logr()).DisableDefaultConfig().
		WithFS(osfs.New()).
		FromConfigFiles(ociConfigFiles...).
		FromPullSecrets(secrets...).
		Build()
	if err != nil {
		return err
	}
	ociClient, err := ociclient.NewClient(logger.Logr(),
		utils.WithConfiguration(c.LsConfig.Registry.OCI),
		ociclient.WithKeyring(ociKeyring),
		ociclient.WithCache(c.SharedCache),
	)
	if err != nil {
		return err
	}

	var inlineCd *cdv2.ComponentDescriptor = nil
	if installation.Spec.ComponentDescriptor != nil {
		inlineCd = installation.Spec.ComponentDescriptor.Inline
	}

	componentsOCIRegistry, err := componentsregistry.NewOCIRegistryWithOCIClient(logger, ociClient, inlineCd)
	if err != nil {
		return err
	}
	if err := compRegistry.Set(componentsOCIRegistry); err != nil {
		return err
	}

	op.SetComponentsRegistry(compRegistry)
	return nil
}

func (c *Controller) resolveSecrets(ctx context.Context, secretRefs []lsv1alpha1.ObjectReference) ([]corev1.Secret, error) {
	secrets := make([]corev1.Secret, len(secretRefs))
	for i, secretRef := range secretRefs {
		secret := corev1.Secret{}
		// todo: check for cache
		if err := c.Client().Get(ctx, secretRef.NamespacedName(), &secret); err != nil {
			return nil, err
		}
		secrets[i] = secret
	}
	return secrets, nil
}

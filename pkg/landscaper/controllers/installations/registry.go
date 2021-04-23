// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"

	"github.com/gardener/component-cli/ociclient"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/osfs"
	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/component-cli/ociclient/credentials"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/pkg/utils"
)

// SetupRegistries sets up components and blueprints registries for the current reconcile
func (c *Controller) SetupRegistries(ctx context.Context, pullSecrets []lsv1alpha1.ObjectReference, installation *lsv1alpha1.Installation) error {
	// resolve all pull secrets
	secrets, err := c.resolveSecrets(ctx, pullSecrets)
	if err != nil {
		return err
	}

	if c.LsConfig.Registry.Local != nil {
		componentsOCIRegistry, err := componentsregistry.NewLocalClient(c.Log(), c.LsConfig.Registry.Local.RootPath)
		if err != nil {
			return err
		}
		if err := c.ComponentsRegistryMgr.Set(componentsOCIRegistry); err != nil {
			return err
		}
	}

	// always add c oci client to support unauthenticated requests
	ociConfigFiles := make([]string, 0)
	if c.LsConfig.Registry.OCI != nil {
		ociConfigFiles = c.LsConfig.Registry.OCI.ConfigFiles
	}
	ociKeyring, err := credentials.NewBuilder(c.Log()).DisableDefaultConfig().
		WithFS(osfs.New()).
		FromConfigFiles(ociConfigFiles...).
		FromPullSecrets(secrets...).
		Build()
	if err != nil {
		return err
	}
	ociClient, err := ociclient.NewClient(c.Log(),
		utils.WithConfiguration(c.LsConfig.Registry.OCI),
		ociclient.WithResolver{Resolver: ociKeyring},
		ociclient.WithCache{Cache: c.ComponentsRegistryMgr.SharedCache()},
	)
	if err != nil {
		return err
	}

	var inlineCd *cdv2.ComponentDescriptor = nil
	if installation.Spec.ComponentDescriptor != nil {
		inlineCd = installation.Spec.ComponentDescriptor.Inline
	}

	componentsOCIRegistry, err := componentsregistry.NewOCIRegistryWithOCIClient(ociClient, inlineCd)
	if err != nil {
		return err
	}
	if err := c.ComponentsRegistryMgr.Set(componentsOCIRegistry); err != nil {
		return err
	}

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

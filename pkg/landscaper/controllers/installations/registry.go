// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"

	"github.com/gardener/component-cli/ociclient"
	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/component-cli/ociclient/credentials"

	confighelper "github.com/gardener/landscaper/pkg/apis/config/helper"
	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
)

// SetupRegistries sets up components and blueprints registries for the current reconcile
func (a *actuator) SetupRegistries(ctx context.Context, pullSecrets []lsv1alpha1.ObjectReference) error {
	// resolve all pull secrets
	secrets, err := a.resolveSecrets(ctx, pullSecrets)
	if err != nil {
		return err
	}

	if a.lsConfig.Registry.Local != nil {
		componentsOCIRegistry, err := componentsregistry.NewLocalClient(a.Log(), a.lsConfig.Registry.Local.RootPath)
		if err != nil {
			return err
		}
		if err := a.componentsRegistryMgr.Set(componentsOCIRegistry); err != nil {
			return err
		}
	}

	// always add a oci client to support unauthenticated requests
	ociConfigFiles := make([]string, 0)
	if a.lsConfig.Registry.OCI != nil {
		ociConfigFiles = a.lsConfig.Registry.OCI.ConfigFiles
	}
	ociKeyring, err := credentials.CreateOCIRegistryKeyring(secrets, ociConfigFiles)
	if err != nil {
		return err
	}
	ociClient, err := ociclient.NewClient(a.Log(),
		confighelper.WithConfiguration(a.lsConfig.Registry.OCI),
		ociclient.WithResolver{Resolver: ociKeyring})
	if err != nil {
		return err
	}
	componentsOCIRegistry, err := componentsregistry.NewOCIRegistryWithOCIClient(ociClient)
	if err != nil {
		return err
	}
	if err := a.componentsRegistryMgr.Set(componentsOCIRegistry); err != nil {
		return err
	}

	return nil
}

func (a *actuator) resolveSecrets(ctx context.Context, secretRefs []lsv1alpha1.ObjectReference) ([]corev1.Secret, error) {
	secrets := make([]corev1.Secret, len(secretRefs))
	for i, secretRef := range secretRefs {
		secret := corev1.Secret{}
		// todo: check for cache
		if err := a.Client().Get(ctx, secretRef.NamespacedName(), &secret); err != nil {
			return nil, err
		}
		secrets[i] = secret
	}
	return secrets, nil
}

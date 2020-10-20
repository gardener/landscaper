// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	corev1 "k8s.io/api/core/v1"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	blueprintsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/pkg/utils/oci"
	"github.com/gardener/landscaper/pkg/utils/oci/credentials"
)

// SetupRegistries sets up components and blueprints registries for the current reconcile
func (a *actuator) SetupRegistries(ctx context.Context, pullSecrets []lsv1alpha1.ObjectReference) error {
	// resolve all pull secrets
	secrets, err := a.resolveSecrets(ctx, pullSecrets)
	if err != nil {
		return err
	}

	if a.lsConfig.Registries.Components.Local != nil {
		componentsOCIRegistry, err := componentsregistry.NewLocalClient(a.Log(), a.lsConfig.Registries.Components.Local.ConfigPaths...)
		if err != nil {
			return err
		}
		if err := a.componentsRegistryMgr.Set(componentsOCIRegistry); err != nil {
			return err
		}
	}

	// always add a oci client to support unauthenticated requests
	ociConfigFiles := make([]string, 0)
	if a.lsConfig.Registries.Artifacts.OCI != nil {
		ociConfigFiles = a.lsConfig.Registries.Components.OCI.ConfigFiles
	}
	ociKeyring, err := credentials.CreateOCIRegistryKeyring(secrets, ociConfigFiles)
	if err != nil {
		return err
	}
	ociClient, err := oci.NewClient(a.Log(), oci.WithConfiguration(a.lsConfig.Registries.Components.OCI), oci.WithResolver{Resolver: ociKeyring})
	if err != nil {
		return err
	}
	componentsOCIRegistry, err := componentsregistry.NewOCIRegistryWithOCIClient(a.Log(), ociClient)
	if err != nil {
		return err
	}
	if err := a.componentsRegistryMgr.Set(componentsOCIRegistry); err != nil {
		return err
	}

	if a.lsConfig.Registries.Artifacts.Local != nil {
		blueprintsOCIRegistry, err := blueprintsregistry.NewLocalRegistry(a.Log(), a.lsConfig.Registries.Artifacts.Local.ConfigPaths...)
		if err != nil {
			return err
		}
		if err := a.blueprintRegistryMgr.Set(blueprintsregistry.LocalAccessType, blueprintsregistry.LocalAccessCodec, blueprintsOCIRegistry); err != nil {
			return err
		}

	}

	// always add a oci client to support unauthenticated requests
	ociConfigFiles = make([]string, 0)
	if a.lsConfig.Registries.Artifacts.OCI != nil {
		ociConfigFiles = a.lsConfig.Registries.Artifacts.OCI.ConfigFiles
	}
	ociKeyring, err = credentials.CreateOCIRegistryKeyring(secrets, ociConfigFiles)
	if err != nil {
		return err
	}
	ociClient, err = oci.NewClient(a.Log(), oci.WithConfiguration(a.lsConfig.Registries.Artifacts.OCI), oci.WithResolver{Resolver: ociKeyring})
	if err != nil {
		return err
	}
	blueprintsOCIRegistry, err := blueprintsregistry.NewOCIRegistryWithOCIClient(a.Log(), ociClient)
	if err != nil {
		return err
	}
	if err := a.InjectUniversalOCIClient(ociClient); err != nil {
		return err
	}

	return a.blueprintRegistryMgr.Set(cdv2.OCIRegistryType, cdv2.KnownAccessTypes[cdv2.OCIRegistryType], blueprintsOCIRegistry)
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

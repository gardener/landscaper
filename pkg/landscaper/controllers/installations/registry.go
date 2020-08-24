// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package installations

import (
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	corev1 "k8s.io/api/core/v1"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	blueprintsoci "github.com/gardener/landscaper/pkg/landscaper/registry/blueprints/oci"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/pkg/utils/oci"
	"github.com/gardener/landscaper/pkg/utils/oci/credentials"
)

// setupRegistries sets up components and blueprints registries for the current reconcile
func (a *actuator) setupRegistries(ctx context.Context, pullSecrets []lsv1alpha1.ObjectReference) error {
	// resolve all pull secrets
	secrets, err := a.resolveSecrets(ctx, pullSecrets)
	if err != nil {
		return err
	}

	ociKeyring, err := credentials.CreateOCIRegistryKeyring(secrets, a.registriesConfig.Components.OCI.ConfigFiles)
	if err != nil {
		return err
	}

	ociClient, err := oci.NewClient(a.Log(), oci.WithConfiguration(a.registriesConfig.Components.OCI), oci.WithResolver{Resolver: ociKeyring})
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

	ociKeyring, err = credentials.CreateOCIRegistryKeyring(secrets, a.registriesConfig.Blueprints.OCI.ConfigFiles)
	if err != nil {
		return err
	}
	ociClient, err = oci.NewClient(a.Log(), oci.WithConfiguration(a.registriesConfig.Blueprints.OCI), oci.WithResolver{Resolver: ociKeyring})
	if err != nil {
		return err
	}
	blueprintsOCIRegistry, err := blueprintsoci.NewWithOCIClient(a.Log(), ociClient)
	if err != nil {
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

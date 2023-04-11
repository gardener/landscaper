// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"fmt"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/dockerconfig"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/pkg/errors"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	corev1 "k8s.io/api/core/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/cnudie"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	ocmadapter "github.com/gardener/landscaper/pkg/components/ocm"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
)

// SetupRegistries sets up components and blueprints registries for the current reconcile
func (c *Controller) SetupRegistries(ctx context.Context, op *operation.Operation, pullSecrets []lsv1alpha1.ObjectReference,
	installation *lsv1alpha1.Installation) error {

	// resolve all pull secrets
	secrets, err := c.resolveSecrets(ctx, pullSecrets)
	if err != nil {
		return err
	}

	var inlineCd *cdv2.ComponentDescriptor = nil
	if installation.Spec.ComponentDescriptor != nil {
		inlineCd = installation.Spec.ComponentDescriptor.Inline
	}

	registry, err := cnudie.NewRegistry(ctx, secrets, c.SharedCache, c.LsConfig.Registry.Local, c.LsConfig.Registry.OCI, inlineCd)
	if err != nil {
		return err
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	octx := ocm.DefaultContext()

	//var spec *dockerconfig.RepositorySpec
	//for _, path := range ociConfigFiles {
	//	spec = dockerconfig.NewRepositorySpec(path, true)
	//	_, err = octx.CredentialsContext().RepositoryForSpec(spec)
	//	if err != nil {
	//		return errors.Wrapf(err, "cannot access %v", path)
	//	}
	//}

	for _, secret := range secrets {
		if secret.Type != corev1.SecretTypeDockerConfigJson {
			continue
		}
		dockerConfigBytes, ok := secret.Data[corev1.DockerConfigJsonKey]
		if !ok {
			continue
		}
		spec := dockerconfig.NewRepositorySpecForConfig(dockerConfigBytes)
		_, err = octx.CredentialsContext().RepositoryForSpec(spec)
		if err != nil {
			return errors.Wrapf(err, "cannot create credentials from secret")
		}
	}

	registry = ocmadapter.NewRegistry(octx)
	op.SetComponentsRegistry(registry)
	return nil
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	//registry := cnudie.NewRegistry(compRegistry)
	//op.SetComponentsRegistry(registry)
	//return nil
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

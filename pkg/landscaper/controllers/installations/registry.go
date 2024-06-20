// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"

	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/utils/cache"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/landscaper/pkg/components/registries"

	corev1 "k8s.io/api/core/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
)

// SetupRegistries sets up components and blueprints registries for the current reconcile
func (c *Controller) SetupRegistries(ctx context.Context, op *operation.Operation, contextObj lsv1alpha1.Context,
	pullSecrets []lsv1alpha1.ObjectReference, installation *lsv1alpha1.Installation) error {

	logger, ctx := logging.FromContextOrNew(ctx, nil)
	pm := utils.StartPerformanceMeasurement(&logger, "SetupRegistries")
	defer pm.StopDebug()

	// resolve all pull secrets
	secrets, err := c.resolveSecrets(ctx, pullSecrets)
	if err != nil {
		return err
	}

	var ocmConfig *corev1.ConfigMap
	if contextObj.OCMConfig != nil {
		ocmConfig = &corev1.ConfigMap{}
		if err := c.LsUncachedClient().Get(ctx, client.ObjectKey{
			Namespace: contextObj.Namespace,
			Name:      contextObj.OCMConfig.Name,
		}, ocmConfig); err != nil {
			return err
		}
	}

	var inlineCd *types.ComponentDescriptor = nil
	if installation.Spec.ComponentDescriptor != nil {
		inlineCd = installation.Spec.ComponentDescriptor.Inline
	}

	var registry model.RegistryAccess
	if inlineCd == nil {
		registry = cache.GetOCMContextCache().GetRegistryAccess(ctx, installation.Status.JobID)
	}

	if registry == nil {
		registry, err = registries.GetFactory(contextObj.UseOCM).NewRegistryAccess(ctx, nil, ocmConfig, secrets, c.LsConfig.Registry.Local, c.LsConfig.Registry.OCI, inlineCd)
		if err != nil {
			return err
		}

		if inlineCd == nil {
			cache.GetOCMContextCache().AddRegistryAccess(ctx, installation.Status.JobID, registry)
		}
	}

	op.SetComponentsRegistry(registry)
	return nil
}

func (c *Controller) resolveSecrets(ctx context.Context, secretRefs []lsv1alpha1.ObjectReference) ([]corev1.Secret, error) {
	secrets := make([]corev1.Secret, len(secretRefs))
	for i, secretRef := range secretRefs {
		secret := corev1.Secret{}
		// todo: check for cache
		if err := c.LsUncachedClient().Get(ctx, secretRef.NamespacedName(), &secret); err != nil {
			return nil, err
		}
		secrets[i] = secret
	}
	return secrets, nil
}

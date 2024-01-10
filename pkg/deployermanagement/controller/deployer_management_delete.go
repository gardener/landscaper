// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployers

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// Delete removes a deployer installation given a deployer registration and a environment.
func (dm *DeployerManagement) Delete(ctx context.Context, registration *lsv1alpha1.DeployerRegistration, env *lsv1alpha1.Environment) error {
	inst, err := dm.getInstallation(ctx, registration, env)
	if err != nil {
		return err
	}
	instKey := kutil.ObjectKeyFromObject(inst)

	if err := read_write_layer.GetInstallation(ctx, dm.lsUncachedClient, instKey, inst, read_write_layer.R000013); err != nil {
		if apierrors.IsNotFound(err) {
			// installation is already deleted
			// nothing to do.
			return nil
		}
		return fmt.Errorf("unable to get istallation: %w", err)
	}

	// trigger deletion
	if err := dm.Writer().DeleteInstallation(ctx, read_write_layer.W000020, inst); err != nil {
		return fmt.Errorf("unable to delete client: %w", err)
	}
	// wait for installation deletion.
	return wait.PollUntilContextTimeout(ctx, 20*time.Second, 5*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		if err := read_write_layer.GetInstallation(ctx, dm.lsUncachedClient, instKey, inst, read_write_layer.R000007); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, fmt.Errorf("unable to get installation while waiting for deletion: %w", err)
		}
		return false, nil
	})
}

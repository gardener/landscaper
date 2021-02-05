// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"errors"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// WaitForInstallationToBeInPhase waits until the given installation is in the expected phase
func WaitForInstallationToBeInPhase(
	ctx context.Context,
	kubeClient client.Client,
	inst *lsv1alpha1.Installation,
	phase lsv1alpha1.ComponentInstallationPhase,
	timeout time.Duration) error {

	err := wait.Poll(5*time.Second, timeout, func() (bool, error) {
		updated := &lsv1alpha1.Installation{}
		if err := kubeClient.Get(ctx, kutil.ObjectKey(inst.Name, inst.Namespace), updated); err != nil {
			return false, err
		}
		*inst = *updated
		if inst.Status.Phase == phase {
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		return fmt.Errorf("error while waiting for installation to be in phase %q: %w", phase, err)
	}
	return nil
}

// WaitForObjectDeletion waits until the given object is deleted
func WaitForObjectDeletion(
	ctx context.Context,
	kubeClient client.Client,
	obj client.Object,
	timeout time.Duration) error {
	err := wait.Poll(5*time.Second, timeout, func() (bool, error) {
		if err := kubeClient.Get(ctx, kutil.ObjectKey(obj.GetName(), obj.GetNamespace()), obj); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("error while waiting for installation to be deleted: %w", err)
	}
	return nil
}

// GetDeployItemsOfInstallation returns all direct deploy items of the installation.
// It does not return deploy items of subinstllations
// todo: for further tests create recursive installation navigator
// e.g. Navigator(inst).GetSubinstallation(name).GetDeployItems()
func GetDeployItemsOfInstallation(ctx context.Context, kubeClient client.Client, inst *lsv1alpha1.Installation) ([]*lsv1alpha1.DeployItem, error) {
	if inst.Status.ExecutionReference == nil {
		return nil, errors.New("no execution reference defined for the installation")
	}
	exec := &lsv1alpha1.Execution{}
	if err := kubeClient.Get(ctx, inst.Status.ExecutionReference.NamespacedName(), exec); err != nil {
		return nil, err
	}

	items := make([]*lsv1alpha1.DeployItem, 0)
	for _, ref := range exec.Status.DeployItemReferences {
		item := &lsv1alpha1.DeployItem{}
		if err := kubeClient.Get(ctx, ref.Reference.NamespacedName(), item); err != nil {
			return nil, fmt.Errorf("unable to find deploy item %q: %w", ref.Name, err)
		}
		items = append(items, item)
	}
	return items, nil
}

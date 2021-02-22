// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
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

// WaitForDeployItemToSucceed waits for a DeployItem to be in phase Succeeded
func WaitForDeployItemToSucceed(
	ctx context.Context,
	kubeClient client.Client,
	obj *lsv1alpha1.DeployItem,
	timeout time.Duration) error {
	return WaitForDeployItemToBeInPhase(ctx, kubeClient, obj, lsv1alpha1.ExecutionPhaseSucceeded, timeout)
}

// WaitForDeployItemToBeInPhase waits until the given deploy item is in the expected phase
func WaitForDeployItemToBeInPhase(
	ctx context.Context,
	kubeClient client.Client,
	deployItem *lsv1alpha1.DeployItem,
	phase lsv1alpha1.ExecutionPhase,
	timeout time.Duration) error {

	err := wait.Poll(5*time.Second, timeout, func() (bool, error) {
		updated := &lsv1alpha1.DeployItem{}
		if err := kubeClient.Get(ctx, kutil.ObjectKey(deployItem.Name, deployItem.Namespace), updated); err != nil {
			return false, err
		}
		*deployItem = *updated
		if deployItem.Status.Phase == phase {
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		return fmt.Errorf("error while waiting for deploy item to be in phase %q: %w", phase, err)
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

// GetSubInstallationsOfInstallation returns the direct subinstallations of a installation.
func GetSubInstallationsOfInstallation(ctx context.Context, kubeClient client.Client, inst *lsv1alpha1.Installation) ([]*lsv1alpha1.Installation, error) {
	list := make([]*lsv1alpha1.Installation, 0)
	if len(inst.Status.InstallationReferences) == 0 {
		return list, nil
	}

	for _, ref := range inst.Status.InstallationReferences {
		inst := &lsv1alpha1.Installation{}
		if err := kubeClient.Get(ctx, ref.Reference.NamespacedName(), inst); err != nil {
			return nil, fmt.Errorf("unable to find installation %q: %w", ref.Name, err)
		}
		list = append(list, inst)
	}
	return list, nil
}

// GetDeployItemExport returns the exports for a deploy item
func GetDeployItemExport(ctx context.Context, kubeClient client.Client, di *lsv1alpha1.DeployItem) ([]byte, error) {
	if di.Status.ExportReference == nil {
		return nil, errors.New("no export defined")
	}
	secret := &corev1.Secret{}
	if err := kubeClient.Get(ctx, di.Status.ExportReference.NamespacedName(), secret); err != nil {
		return nil, fmt.Errorf("unable to get export from %q: %w", di.Status.ExportReference.NamespacedName(), err)
	}

	return secret.Data[lsv1alpha1.DataObjectSecretDataKey], nil
}

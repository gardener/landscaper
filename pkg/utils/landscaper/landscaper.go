// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package landscaper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// WaitForInstallationToFinish waits until the given installation has finished with the given phase
func WaitForInstallationToFinish(
	ctx context.Context,
	kubeClient client.Reader,
	inst *lsv1alpha1.Installation,
	phase lsv1alpha1.InstallationPhase,
	timeout time.Duration) error {

	err := WaitForInstallationToHaveCondition(ctx, kubeClient, inst, func(installation *lsv1alpha1.Installation) (bool, error) {
		return IsInstallationFinished(installation, phase)
	}, timeout)
	if err != nil {
		return fmt.Errorf("error while waiting for installation to finish: %w", err)
	}
	return nil
}

func IsInstallationFinished(inst *lsv1alpha1.Installation, phase lsv1alpha1.InstallationPhase) (bool, error) {
	if inst.Status.JobIDFinished != inst.Status.JobID || helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ReconcileOperation) ||
		inst.Status.InstallationPhase.IsEmpty() {
		return false, nil
	} else if inst.Status.InstallationPhase != phase {
		return false, fmt.Errorf("installation has finish with unexpected phase: %s, expected: %s", inst.Status.InstallationPhase, phase)
	}
	return true, nil
}

// WaitForInstallationToBeDeleted waits until the given installation has finished with the given phase
func WaitForInstallationToBeDeleted(
	ctx context.Context,
	kubeClient client.Reader,
	inst *lsv1alpha1.Installation,
	timeout time.Duration) error {

	pollErr := wait.PollUntilContextTimeout(ctx, 1*time.Second, timeout, true, func(ctx context.Context) (done bool, err error) {
		updated := &lsv1alpha1.Installation{}
		getErr := read_write_layer.GetInstallation(ctx, kubeClient, kutil.ObjectKey(inst.Name, inst.Namespace),
			updated, read_write_layer.R000012)
		return getErr != nil && apierrors.IsNotFound(getErr), nil
	})

	if pollErr != nil {
		return fmt.Errorf("error while waiting for installation to be deleted: %w", pollErr)
	}
	return nil
}

// InstallationConditionFunc defines a condition function that is used to in the wait helper function.
type InstallationConditionFunc func(installation *lsv1alpha1.Installation) (bool, error)

// WaitForInstallationToHaveCondition waits until the given installation fulfills the given condition.
func WaitForInstallationToHaveCondition(
	ctx context.Context,
	kubeClient client.Reader,
	inst *lsv1alpha1.Installation,
	cond InstallationConditionFunc,
	timeout time.Duration) error {

	return wait.PollUntilContextTimeout(ctx, 1*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		updated := &lsv1alpha1.Installation{}
		if err := read_write_layer.GetInstallation(ctx, kubeClient, kutil.ObjectKey(inst.Name, inst.Namespace), updated,
			read_write_layer.R000015); err != nil {
			return false, err
		}
		*inst = *updated
		return cond(inst)
	})
}

// WaitForDeployItemToFinish waits until the given deploy item has finished with the given phase
func WaitForDeployItemToFinish(
	ctx context.Context,
	kubeClient client.Reader,
	deployItem *lsv1alpha1.DeployItem,
	phase lsv1alpha1.DeployItemPhase,
	timeout time.Duration) error {

	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, timeout, false, func(ctx context.Context) (bool, error) {
		updated := &lsv1alpha1.DeployItem{}
		if err := read_write_layer.GetDeployItem(ctx, kubeClient, kutil.ObjectKey(deployItem.Name, deployItem.Namespace),
			updated, read_write_layer.R000029); err != nil {
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

// GetSubInstallationsOfInstallation returns the direct subinstallations of a installation.
func GetSubInstallationsOfInstallation(ctx context.Context, kubeClient client.Client, inst *lsv1alpha1.Installation) ([]*lsv1alpha1.Installation, error) {
	installationList := &lsv1alpha1.InstallationList{}

	if err := read_write_layer.ListInstallations(ctx, kubeClient, installationList, read_write_layer.R000033,
		client.InNamespace(inst.Namespace),
		client.MatchingLabels{
			lsv1alpha1.EncompassedByLabel: inst.Name,
		}); err != nil {
		return nil, err
	}

	list := make([]*lsv1alpha1.Installation, 0)

	for i := range installationList.Items {
		inst := &installationList.Items[i]
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
	if err := read_write_layer.GetSecret(ctx, kubeClient, di.Status.ExportReference.NamespacedName(), secret, read_write_layer.R000056); err != nil {
		return nil, fmt.Errorf("unable to get export from %q: %w", di.Status.ExportReference.NamespacedName(), err)
	}

	return secret.Data[lsv1alpha1.DataObjectSecretDataKey], nil
}

// CreateKubernetesTarget creates a new target of type kubernetes
func CreateKubernetesTarget(namespace, name string, restConfig *rest.Config) (*lsv1alpha1.Target, error) {
	target := &lsv1alpha1.Target{}
	target.Name = name
	target.Namespace = namespace
	if err := BuildKubernetesTarget(target, restConfig); err != nil {
		return nil, err
	}
	return target, nil
}

// BuildKubernetesTarget adds all kubernetes type related attributes to the target
func BuildKubernetesTarget(target *lsv1alpha1.Target, restConfig *rest.Config) error {
	data, err := kutil.GenerateKubeconfigBytes(restConfig)
	if err != nil {
		return err
	}

	config := targettypes.KubernetesClusterTargetConfig{
		Kubeconfig: targettypes.ValueRef{
			StrVal: pointer.String(string(data)),
		},
	}
	data, err = json.Marshal(config)
	if err != nil {
		return err
	}

	target.Spec.Type = targettypes.KubernetesClusterTargetType
	target.Spec.Configuration = lsv1alpha1.NewAnyJSONPointer(data)

	return nil
}

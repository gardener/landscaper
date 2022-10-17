// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package lib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	lsutil "github.com/gardener/landscaper/pkg/utils"

	"k8s.io/client-go/tools/record"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/pkg/api"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

// GetKubeconfigFromTargetConfig fetches the kubeconfig from a given config.
// If the config defines the target from a secret that secret is read from all provided clients.
func GetKubeconfigFromTargetConfig(ctx context.Context, config *lsv1alpha1.KubernetesClusterTargetConfig, kubeClients ...client.Client) ([]byte, error) {
	if config.Kubeconfig.StrVal != nil {
		return []byte(*config.Kubeconfig.StrVal), nil
	}
	if config.Kubeconfig.SecretRef == nil {
		return nil, errors.New("kubeconfig not defined")
	}

	return GetKubeconfigFromSecretRef(ctx, config.Kubeconfig.SecretRef, kubeClients...)
}

func GetKubeconfigFromSecretRef(ctx context.Context, ref *lsv1alpha1.SecretReference, kubeClients ...client.Client) ([]byte, error) {
	var errList []error
	for _, kubeClient := range kubeClients {
		secret := &corev1.Secret{}
		if err := kubeClient.Get(ctx, ref.NamespacedName(), secret); err != nil {
			if !apierrors.IsNotFound(err) {
				errList = append(errList, err)
			}
			continue
		}

		if len(ref.Key) == 0 {
			ref.Key = lsv1alpha1.DefaultKubeconfigKey
		}

		kubeconfig, ok := secret.Data[ref.Key]
		if !ok {
			errList = append(errList, fmt.Errorf("secret found but key %q not found", ref.Key))
			continue
		}
		return kubeconfig, nil
	}

	if len(errList) != 0 {
		return nil, utilerrors.NewAggregate(errList)
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{
		Resource: "secret",
	}, ref.Name)
}

// SetProviderStatus sets the provider specific status for a deploy item.
func SetProviderStatus(di *lsv1alpha1.DeployItem, status runtime.Object, scheme *runtime.Scheme) error {
	rawStatus, err := kutil.ConvertToRawExtension(status, scheme)
	if err != nil {
		return err
	}
	di.Status.ProviderStatus = rawStatus
	return nil
}

// CreateOrUpdateExport creates or updates the export of a deploy item.
func CreateOrUpdateExport(ctx context.Context, kubeWriter *read_write_layer.Writer, kubeClient client.Client, deployItem *lsv1alpha1.DeployItem, values interface{}) error {
	if values == nil {
		return nil
	}
	const currOp = "CreateExports"
	data, err := json.Marshal(values)
	if err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "MarshalExportData", err.Error())
	}

	secret := &corev1.Secret{}
	secret.Name = fmt.Sprintf("%s-export", deployItem.Name)
	secret.Namespace = deployItem.Namespace
	if deployItem.Status.ExportReference != nil {
		secret.Name = deployItem.Status.ExportReference.Name
		secret.Namespace = deployItem.Status.ExportReference.Namespace
	}

	_, err = controllerutil.CreateOrUpdate(ctx, kubeClient, secret, func() error {
		secret.Data = map[string][]byte{
			lsv1alpha1.DataObjectSecretDataKey: data,
		}
		return controllerutil.SetOwnerReference(deployItem, secret, api.LandscaperScheme)
	})
	if err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "CreateOrUpdateSecret", err.Error())
	}

	deployItem.Status.ExportReference = &lsv1alpha1.ObjectReference{
		Name:      secret.Name,
		Namespace: secret.Namespace,
	}

	if err := kubeWriter.UpdateDeployItemStatus(ctx, read_write_layer.W000060, deployItem); err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "Update DeployItem", err.Error())
	}
	return nil
}

// GetRegistryPullSecretsFromContext returns the object references to
// registry pull secrets defined by the landscaper context.
func GetRegistryPullSecretsFromContext(lsCtx *lsv1alpha1.Context) []lsv1alpha1.ObjectReference {
	refs := make([]lsv1alpha1.ObjectReference, len(lsCtx.RegistryPullSecrets))
	for i, r := range lsCtx.RegistryPullSecrets {
		refs[i] = lsv1alpha1.ObjectReference{
			Name:      r.Name,
			Namespace: lsCtx.Namespace,
		}
	}
	return refs
}

func HandleReconcileResult(ctx context.Context, err lserrors.LsError, oldDeployItem, deployItem *lsv1alpha1.DeployItem,
	lsClient client.Client, lsEventRecorder record.EventRecorder) error {

	logger, ctx := logging.FromContextOrNew(ctx, nil)
	lsutil.SetLastError(&deployItem.Status, lserrors.TryUpdateLsError(deployItem.Status.GetLastError(), err))

	if deployItem.Status.GetLastError() != nil {
		if lserrors.ContainsAnyErrorCode(deployItem.Status.GetLastError().Codes, lsv1alpha1.UnrecoverableErrorCodes) {
			deployItem.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
		}

		lastErr := deployItem.Status.GetLastError()
		lsEventRecorder.Event(deployItem, corev1.EventTypeWarning, lastErr.Reason, lastErr.Message)
	}

	if deployItem.Status.Phase == lsv1alpha1.ExecutionPhaseFailed {
		deployItem.Status.DeployItemPhase = lsv1alpha1.DeployItemPhaseFailed
	} else if deployItem.Status.Phase == lsv1alpha1.ExecutionPhaseSucceeded {
		deployItem.Status.DeployItemPhase = lsv1alpha1.DeployItemPhaseSucceeded
	}

	if deployItem.Status.DeployItemPhase == lsv1alpha1.DeployItemPhaseSucceeded ||
		deployItem.Status.DeployItemPhase == lsv1alpha1.DeployItemPhaseFailed {
		deployItem.Status.JobIDFinished = deployItem.Status.GetJobID()
	}

	if !reflect.DeepEqual(oldDeployItem.Status, deployItem.Status) {
		if err2 := read_write_layer.NewWriter(lsClient).UpdateDeployItemStatus(ctx, read_write_layer.W000092, deployItem); err2 != nil {
			if !deployItem.DeletionTimestamp.IsZero() {
				// recheck if already deleted
				diRecheck := &lsv1alpha1.DeployItem{}
				errRecheck := read_write_layer.GetDeployItem(ctx, lsClient, kutil.ObjectKey(deployItem.Name, deployItem.Namespace), diRecheck)
				if errRecheck != nil && apierrors.IsNotFound(errRecheck) {
					return nil
				}
			}

			if apierrors.IsConflict(err2) { // reduce logging
				logger.Debug("Unable to update status", lc.KeyError, err2.Error())
			} else {
				logger.Error(err2, "Unable to update status")
			}
			if err == nil {
				return err2
			}
		}
	}

	return err
}

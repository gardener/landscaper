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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/landscaper/targetresolver"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/deployer/lib/targetselector"
	lsutil "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// GetKubeconfigFromTargetConfig fetches the kubeconfig from a given config.
// If the config defines the target from a secret that secret is read from all provided clients.
func GetKubeconfigFromTargetConfig(ctx context.Context, config *targettypes.KubernetesClusterTargetConfig,
	targetNamespace string, lsClient client.Client) ([]byte, error) {
	if config.Kubeconfig.StrVal != nil {
		return []byte(*config.Kubeconfig.StrVal), nil
	}
	if config.Kubeconfig.SecretRef == nil {
		return nil, errors.New("kubeconfig not defined")
	}

	return GetKubeconfigFromSecretRef(ctx, config.Kubeconfig.SecretRef, targetNamespace, lsClient)
}

func GetKubeconfigFromSecretRef(ctx context.Context, ref *lsv1alpha1.SecretReference, targetNamespace string,
	lsClient client.Client) ([]byte, error) {

	if len(ref.Namespace) > 0 && ref.Namespace != targetNamespace {
		return nil, fmt.Errorf("namespace of secret ref %s differs from target namespace %s",
			ref.Namespace, targetNamespace)
	}

	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{Name: ref.Name, Namespace: targetNamespace}
	if err := read_write_layer.GetSecret(ctx, lsClient, secretKey, secret, read_write_layer.R000050); err != nil {
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Resource: "secret",
		}, ref.Name)
	}

	if len(ref.Key) == 0 {
		ref.Key = targettypes.DefaultKubeconfigKey
	}

	kubeconfig, ok := secret.Data[ref.Key]
	if !ok {
		return nil, fmt.Errorf("secret found but key %q not found", ref.Key)
	}
	return kubeconfig, nil
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
	lsClient client.Client, lsEventRecorder record.EventRecorder, finishedObjectCache *lsutil.FinishedObjectCache) error {

	logger, ctx := logging.FromContextOrNew(ctx, nil)
	lsutil.SetLastError(&deployItem.Status, lserrors.TryUpdateLsError(deployItem.Status.GetLastError(), err))

	if deployItem.Status.GetLastError() != nil {
		if lserrors.ContainsAnyErrorCode(deployItem.Status.GetLastError().Codes, lsv1alpha1.UnrecoverableErrorCodes) {
			lsv1alpha1helper.SetDeployItemToFailed(deployItem)
		}

		lastErr := deployItem.Status.GetLastError()
		lsEventRecorder.Event(deployItem, corev1.EventTypeWarning, lastErr.Reason, lastErr.Message)
	}

	// if a reconciliation ends in a final phase, the current job is done
	if deployItem.Status.Phase.IsFinal() {
		deployItem.Status.JobIDFinished = deployItem.Status.GetJobID()
		deployItem.Status.TransitionTimes = lsutil.SetFinishedTransitionTime(deployItem.Status.TransitionTimes)
	}

	if !reflect.DeepEqual(&oldDeployItem.Status, &deployItem.Status) {
		if err2 := read_write_layer.NewWriter(lsClient).UpdateDeployItemStatus(ctx, read_write_layer.W000092, deployItem); err2 != nil {
			if !deployItem.DeletionTimestamp.IsZero() {
				// recheck if already deleted
				diRecheck := &lsv1alpha1.DeployItem{}
				errRecheck := read_write_layer.GetDeployItem(ctx, lsClient, kutil.ObjectKey(deployItem.Name, deployItem.Namespace),
					diRecheck, read_write_layer.R000030)
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
		} else if finishedObjectCache != nil && IsDeployItemFinished(deployItem) {
			finishedObjectCache.AddSynchonized(&deployItem.ObjectMeta)
		}
	}

	return err
}

func CheckResponsibility(ctx context.Context, lsClient client.Client, obj *metav1.PartialObjectMetadata,
	deployerType lsv1alpha1.DeployItemType, targetSelectors []lsv1alpha1.TargetSelector) (*lsv1alpha1.ResolvedTarget, bool, bool, lserrors.LsError) {

	annotatedDeployerType, found := obj.GetAnnotations()[lsv1alpha1.DeployerTypeAnnotation]
	targetName := obj.GetAnnotations()[lsv1alpha1.DeployerTargetNameAnnotation]
	if !found {
		// deploy item in old version
		di := &lsv1alpha1.DeployItem{}
		if err := read_write_layer.GetDeployItem(ctx, lsClient, client.ObjectKeyFromObject(obj), di, read_write_layer.R000032); err != nil {
			return nil, false, false, lserrors.NewWrappedError(err, "CheckResponsibility", "fetchDeployItem",
				"fetching deploy item failed")
		}

		annotatedDeployerType = string(di.Spec.Type)

		targetName = lsv1alpha1.NoTargetNameValue
		if di.Spec.Target != nil && di.Spec.Target.Name != "" {
			targetName = di.Spec.Target.Name
		}
	}

	if annotatedDeployerType != string(deployerType) {
		return nil, false, false, nil
	}

	return checkTargetResponsibilityAndResolve(ctx, lsClient, obj.Namespace, targetName, targetSelectors)
}

func checkTargetResponsibilityAndResolve(ctx context.Context, lsClient client.Client,
	targetNamespace, targetName string, targetSelectors []lsv1alpha1.TargetSelector) (*lsv1alpha1.ResolvedTarget, bool, bool, lserrors.LsError) {

	target, responsible, targetNotFound, lsError := checkTargetResponsibility(ctx, lsClient, targetNamespace, targetName, targetSelectors)
	if lsError != nil {
		return nil, false, false, lsError
	}

	if targetNotFound {
		return nil, responsible, targetNotFound, nil // = nil, true, true, nil
	}

	if !responsible {
		return nil, responsible, targetNotFound, nil // = nil, false, false, nil
	}

	// resolve Target reference, if any
	var rt *lsv1alpha1.ResolvedTarget
	var err error
	if target != nil {
		rt, err = targetresolver.Resolve(ctx, target, lsClient)
		if err != nil {
			lsError = lserrors.NewWrappedError(err, "checkTargetResponsibilityAndResolve", "resolveTarget", err.Error())
			return nil, false, false, lsError
		}
	}

	return rt, true, false, nil
}

func checkTargetResponsibility(ctx context.Context, lsClient client.Client,
	targetNamespace, targetName string, targetSelectors []lsv1alpha1.TargetSelector) (*lsv1alpha1.Target, bool, bool, lserrors.LsError) {

	logger, ctx := logging.FromContextOrNew(ctx, nil)

	op := "checkTargetResponsibility"

	if targetName == lsv1alpha1.NoTargetNameValue {
		logger.Debug("No target defined")
		return nil, true, false, nil
	}

	logger.Debug("Found target. Checking responsibility")
	target := &lsv1alpha1.Target{}
	if err := read_write_layer.GetTarget(ctx, lsClient, client.ObjectKey{Namespace: targetNamespace, Name: targetName},
		target, read_write_layer.R000051); err != nil {
		lsError := lserrors.NewWrappedError(err, op, "FetchTarget", "unable to get target for deploy item - other error")
		if apierrors.IsNotFound(err) {
			return nil, true, true, nil
		}
		return nil, false, false, lsError
	}
	if len(targetSelectors) == 0 {
		logger.Debug("No target selectors defined")
		return target, true, false, nil
	}
	matched, err := targetselector.MatchOne(target, targetSelectors)
	if err != nil {
		lsError := lserrors.NewWrappedError(err, op, "MatchOne", "unable to match target selector")
		return nil, false, false, lsError
	}
	if !matched {
		logger.Debug("The deployitem's target has not matched the given target selector",
			"target", target.Name)
		return nil, false, false, nil
	}
	return target, true, false, nil
}

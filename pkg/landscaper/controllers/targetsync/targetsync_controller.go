// SPDX-FileCopyrightText: 2022 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package targetsync

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/utils/pointer"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutils "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
)

// AddControllerToManagerForTargetSyncs adds the controller to the manager
func AddControllerToManagerForTargetSyncs(logger logging.Logger, mgr manager.Manager) error {
	log := logger.Reconciles("targetSync", "TargetSync")
	ctrl := NewTargetSyncController(log, mgr.GetClient(), NewDefaultSourceClientProvider())

	predicates := builder.WithPredicates(predicate.Or(predicate.LabelChangedPredicate{},
		predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{}))

	return builder.ControllerManagedBy(mgr).
		For(&lsv1alpha1.TargetSync{}, predicates).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(ctrl)
}

// TargetSyncController is the TargetSync controller
type TargetSyncController struct {
	log                  logging.Logger
	targetClient         client.Client
	sourceClientProvider SourceClientProvider
}

// NewTargetSyncController returns a new TargetSync controller
func NewTargetSyncController(logger logging.Logger, targetClient client.Client, p SourceClientProvider) reconcile.Reconciler {
	return &TargetSyncController{
		log:                  logger,
		targetClient:         targetClient,
		sourceClientProvider: p,
	}
}

// Reconcile reconciles requests for TargetSyncs
func (c *TargetSyncController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger, ctx := c.log.StartReconcileAndAddToContext(ctx, req)

	targetSync := &lsv1alpha1.TargetSync{}
	if err := c.targetClient.Get(ctx, req.NamespacedName, targetSync); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info(err.Error())
			return reconcile.Result{}, nil
		}
		logger.Error(err, "fetching targetsync object failed")
		return reconcile.Result{}, err
	}

	// set finalizer
	if targetSync.DeletionTimestamp.IsZero() && !kutils.HasFinalizer(targetSync, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(targetSync, lsv1alpha1.LandscaperFinalizer)
		if err := c.targetClient.Update(ctx, targetSync); err != nil {
			logger.Error(err, "adding finalizer to targetsync object failed")
			return reconcile.Result{}, err
		}
		// do not return here because the controller only watches for particular events and setting a finalizer is not part of this
	}

	if targetSync.DeletionTimestamp.IsZero() {
		if err := c.handleReconcile(ctx, targetSync); err != nil {
			logger.Error(err, "reconciling targetsync object failed")
			return reconcile.Result{}, err
		}
	} else {
		if err := c.handleDelete(ctx, targetSync); err != nil {
			logger.Error(err, "deleting targetsync object failed")
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: requeueInterval,
	}, nil
}

func (c *TargetSyncController) handleReconcile(ctx context.Context, targetSync *lsv1alpha1.TargetSync) error {
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyReconciledResource, client.ObjectKeyFromObject(targetSync).String()})

	errors := []error{}

	targetSyncs, err := c.fetchTargetSyncs(ctx, targetSync)

	if err != nil {
		errors = append(errors, err)
	} else if len(targetSyncs.Items) > 1 {
		err = fmt.Errorf("more than one TargetSync object in the same namespace is not allowed")
		errors = append(errors, err)
	} else {
		sourceClient, restConfig, err := c.sourceClientProvider.GetSourceClient(ctx, targetSync, c.targetClient, nil)
		if err != nil {
			logger.Error(err, "fetching source client for targetsync object failed")
			errors = append(errors, err)
		} else {
			err = c.refreshToken(ctx, targetSync, restConfig)
			if err != nil {
				logger.Error(err, "refreshing token failed")
				errors = append(errors, err)
			} else {
				errors = c.handleSecretsAndShoots(ctx, targetSync, sourceClient)
			}
		}
	}

	errorStrings := []string{}
	for _, nextError := range errors {
		errorStrings = append(errorStrings, nextError.Error())
	}

	now := metav1.Now()
	targetSync.Status.LastErrors = errorStrings
	targetSync.Status.ObservedGeneration = targetSync.GetGeneration()
	targetSync.Status.LastUpdateTime = &now

	if err = c.targetClient.Status().Update(ctx, targetSync); err != nil {
		logger.Error(err, "updating status at the end of reconcile of targetsync object failed")
		return err
	}

	if len(errors) > 0 {
		return errors[0]
	}
	return nil
}

func (c *TargetSyncController) handleDelete(ctx context.Context, targetSync *lsv1alpha1.TargetSync) error {
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyReconciledResource, client.ObjectKeyFromObject(targetSync).String()})

	errorStrings := []string{}

	err := c.removeTargetsAndSecrets(ctx, targetSync)
	if err != nil {
		errorStrings = append(errorStrings, err.Error())

		now := metav1.Now()
		targetSync.Status.LastErrors = errorStrings
		targetSync.Status.ObservedGeneration = targetSync.GetGeneration()
		targetSync.Status.LastUpdateTime = &now

		if internalErr := c.targetClient.Status().Update(ctx, targetSync); err != nil {
			logger.Error(err, "updating status with error for deleting targetsync object failed")
			return internalErr
		}

		return err
	}

	logger.Info("deleting targetsync object: removing finalizer")
	controllerutil.RemoveFinalizer(targetSync, lsv1alpha1.LandscaperFinalizer)
	if err := c.targetClient.Update(ctx, targetSync); err != nil {
		logger.Error(err, "removing finalizer for deleting targetsync object failed")
		return lserrors.NewWrappedError(err, "handleDelete", "RemoveFinalizer", err.Error())
	}

	return nil
}

func (c *TargetSyncController) handleSecretsAndShoots(ctx context.Context, targetSync *lsv1alpha1.TargetSync,
	sourceClient client.Client) []error {

	logger, ctx := logging.FromContextOrNew(ctx, nil)
	errors := []error{}

	if targetSync.Spec.SecretNameExpression != "" && targetSync.Spec.ShootNameExpression != "" {
		msg := "a targetsync object with both, secretNameExpression and shootNameExpression, is not allowed"
		logger.Error(nil, msg)
		errors = append(errors, fmt.Errorf(msg))
		return errors
	}

	oldTargets, err := c.fetchOldTargets(ctx, targetSync)
	if err != nil {
		errors = append(errors, err)
		return errors
	}

	if targetSync.Spec.SecretNameExpression != "" {
		secrFilter, err := newNameFilter(targetSync.Spec.SecretNameExpression)
		if err != nil {
			logger.Error(err, "building secret name filter of targetsync object failed: "+targetSync.Spec.SecretNameExpression)
			errors = append(errors, err)
			return errors
		}

		secrets := &corev1.SecretList{}
		if err = sourceClient.List(ctx, secrets, client.InNamespace(targetSync.Spec.SourceNamespace)); err != nil {
			logger.Error(err, "fetching secret list for targetsync object failed")
			errors = append(errors, err)
			return errors
		}

		for _, secret := range secrets.Items {
			if secrFilter.shouldBeProcessed(&secret) {
				secretLogger := logger.WithValues(lc.KeyResource, client.ObjectKeyFromObject(&secret).String())
				secretCtx := logging.NewContext(ctx, secretLogger)

				delete(oldTargets, secret.Name)

				if err = c.handleSecret(secretCtx, targetSync, &secret); err != nil {
					msg := fmt.Sprintf("handling secret %s of targetsync object failed", client.ObjectKeyFromObject(&secret).String())
					secretLogger.Error(err, msg)
					errors = append(errors, err)
				}
			}
		}
	}

	if targetSync.Spec.ShootNameExpression != "" {
		shootFilter, err := newNameFilter(targetSync.Spec.ShootNameExpression)
		if err != nil {
			logger.Error(err, "building shoot name filter of targetsync object failed: "+targetSync.Spec.ShootNameExpression)
			errors = append(errors, err)
			return errors
		}

		shootClient, err := c.sourceClientProvider.GetUnstructuredSourceClient(ctx, targetSync, c.targetClient, shootGVR)
		if err != nil {
			logger.Error(err, "failed to get shoot client for targetsync")
			errors = append(errors, err)
			return errors
		}

		shootList, err := shootClient.List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Error(err, "failed to list shoots for targetsync")
			errors = append(errors, err)
			return errors
		}

		for _, shoot := range shootList.Items {
			if shootFilter.shouldBeProcessed(&shoot) {
				shootLogger := logger.WithValues(lc.KeyResource, client.ObjectKeyFromObject(&shoot).String())
				shootCtx := logging.NewContext(ctx, shootLogger)

				targetName := c.deriveTargetNameFromShootName(shoot.GetName())
				delete(oldTargets, targetName)

				if err = c.handleShoot(shootCtx, targetSync, shootClient, &shoot); err != nil {
					msg := fmt.Sprintf("handling shoot %s of targetsync object failed", client.ObjectKeyFromObject(&shoot).String())
					shootLogger.Error(err, msg)
					errors = append(errors, err)
				}
			}
		}
	}

	if targetSync.Spec.CreateTargetToSource {
		targetName := targetSync.Spec.TargetToSourceName
		if targetName == "" {
			targetName = targetSync.Spec.SourceNamespace
		}
		delete(oldTargets, targetName)
		if err := c.createOrUpdateTarget(ctx, targetSync, targetName, targetSync.Spec.SecretRef.Name,
			targetSync.Spec.SecretRef.Key, false); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) == 0 {
		for key := range oldTargets {
			nextOldTarget := oldTargets[key]
			// do not delete the secret to the source namespace
			if !c.isTargetSyncSecret(nextOldTarget.Spec.SecretRef.Name, targetSync) {
				secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: targetSync.Namespace, Name: key}}
				if err := c.targetClient.Delete(ctx, &secret); err != nil {
					msg := fmt.Sprintf("deleting old secret %s of targetsync object failed", client.ObjectKeyFromObject(&secret).String())
					logger.Error(err, msg)
					errors = append(errors, err)
				}
			}

			target := lsv1alpha1.Target{ObjectMeta: metav1.ObjectMeta{Namespace: targetSync.Namespace, Name: key}}
			if err := c.targetClient.Delete(ctx, &target); err != nil {
				msg := fmt.Sprintf("deleting old target %s of targetsync object failed", client.ObjectKeyFromObject(&target).String())
				logger.Error(err, msg)
				errors = append(errors, err)
			}
		}
	}

	return errors
}

func (c *TargetSyncController) handleSecret(ctx context.Context, targetSync *lsv1alpha1.TargetSync, secret *corev1.Secret) error {
	targetName := secret.GetName()
	err := c.createOrUpdateTarget(ctx, targetSync, targetName, "", "", false)
	if err != nil {
		return err
	}

	err = c.createOrUpdateSecret(ctx, targetSync, secret)
	return err
}

func (c *TargetSyncController) handleShoot(ctx context.Context, targetSync *lsv1alpha1.TargetSync,
	shootClient dynamic.ResourceInterface, shoot *unstructured.Unstructured) error {
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	targetName := c.deriveTargetNameFromShootName(shoot.GetName())

	due, err := c.isRenewalOfShortLivedKubeconfigDue(ctx, targetName, targetSync.Namespace)
	if err != nil {
		return err
	} else if !due {
		return nil
	}

	adminKubeconfigRequest := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "authentication.gardener.cloud/v1alpha1",
			"kind":       "AdminKubeconfigRequest",
			"metadata": map[string]interface{}{
				"namespace": shoot.GetNamespace(),
				"name":      shoot.GetName(),
			},
			"spec": map[string]interface{}{
				"expirationSeconds": kubeconfigExpirationSeconds,
			},
		},
	}

	result, err := shootClient.Create(ctx, &adminKubeconfigRequest, metav1.CreateOptions{}, subresourceAdminkubeconfig)
	if err != nil {
		msg := "targetsync for shoot failed at adminkubeconfigrequest"
		logger.Error(err, msg)
		return fmt.Errorf("%s; target: %s, error: %w", msg, targetName, err)
	}

	kubeconfig, found, err := unstructured.NestedString(result.Object, "status", "kubeconfig")
	if err != nil {
		msg := "targetsync for shoot failed: could not get kubeconfig from adminkubeconfig subresource"
		logger.Error(err, msg)
		return fmt.Errorf("%s; target: %s, error: %w", msg, targetName, err)
	} else if !found {
		msg := "targetsync for shoot failed: could not find kubeconfig in adminkubeconfig subresource"
		logger.Error(nil, msg)
		return fmt.Errorf("%s; target: %s", msg, targetName)
	}

	err = c.createOrUpdateSecretForShoot(ctx, targetSync, targetName, kubeconfig)
	if err != nil {
		msg := "targetsync for shoot failed: could not create or update secret"
		logger.Error(err, msg)
		return fmt.Errorf("%s; target: %s, error: %w", msg, targetName, err)
	}

	err = c.createOrUpdateTarget(ctx, targetSync, targetName, "", "", true)
	if err != nil {
		msg := "targetsync for shoot failed: could not create or update target"
		logger.Error(err, msg)
		return fmt.Errorf("%s; target: %s, error: %w", msg, targetName, err)
	}

	return nil
}

func (c *TargetSyncController) isRenewalOfShortLivedKubeconfigDue(ctx context.Context, targetName, targetNamespace string) (due bool, err error) {
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	oldTarget := &lsv1alpha1.Target{
		ObjectMeta: controllerruntime.ObjectMeta{Name: targetName, Namespace: targetNamespace},
	}

	err = c.targetClient.Get(ctx, client.ObjectKeyFromObject(oldTarget), oldTarget)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("targetsync for shoot is due for the first time")
			return true, nil
		}

		msg := "targetsync for shoot failed because synced target could not be read"
		logger.Error(err, msg)
		return false, fmt.Errorf("%s; target: %s, error: %w", msg, targetName, err)
	}

	lastTargetSync, err := helper.GetTimestampAnnotation(oldTarget.ObjectMeta, annotationKeyLastTargetSync)
	if err != nil {
		logger.Error(err, "targetsync for shoot is due because last targetsync time could not be retrieved")
		return true, nil

	} else if time.Since(lastTargetSync) > kubeconfigRenewalSeconds*time.Second {
		logger.Info("targetsync for shoot is due because kubeconfig will expire soon")
		return true, nil
	}

	return false, nil
}

func (c *TargetSyncController) createOrUpdateTarget(ctx context.Context, targetSync *lsv1alpha1.TargetSync,
	targetName, alternativeSecretName, alternativeKubeconfigKey string, addLastTargetSyncAnnotation bool) error {

	newTarget := &lsv1alpha1.Target{
		ObjectMeta: controllerruntime.ObjectMeta{Name: targetName, Namespace: targetSync.Namespace},
	}

	_, err := controllerruntime.CreateOrUpdate(ctx, c.targetClient, newTarget, func() error {
		newTarget.ObjectMeta.Labels = map[string]string{
			labelKeyTargetSync: labelValueOk,
		}
		if addLastTargetSyncAnnotation {
			helper.SetTimestampAnnotationNow(&newTarget.ObjectMeta, annotationKeyLastTargetSync)
		}

		secretName := targetName
		if alternativeSecretName != "" {
			secretName = alternativeSecretName
		}

		key := kubeconfigKey
		if alternativeKubeconfigKey != "" {
			key = alternativeKubeconfigKey
		}

		newTarget.Spec = lsv1alpha1.TargetSpec{
			Type: targettypes.KubernetesClusterTargetType,
			SecretRef: &lsv1alpha1.LocalSecretReference{
				Name: secretName,
				Key:  key,
			},
		}
		return nil
	})

	return err
}

func (c *TargetSyncController) createOrUpdateSecret(ctx context.Context, targetSync *lsv1alpha1.TargetSync, secret *corev1.Secret) error {
	newSecret := &corev1.Secret{
		ObjectMeta: controllerruntime.ObjectMeta{Name: secret.Name, Namespace: targetSync.Namespace},
	}

	_, err := controllerruntime.CreateOrUpdate(ctx, c.targetClient, newSecret, func() error {
		newSecret.ObjectMeta.Labels = map[string]string{
			labelKeyTargetSync: labelValueOk,
		}
		newSecret.Data = secret.Data
		newSecret.Type = secret.Type
		return nil
	})

	return err
}

func (c *TargetSyncController) createOrUpdateSecretForShoot(ctx context.Context, targetSync *lsv1alpha1.TargetSync,
	targetName string, kubeconfig string) error {

	newSecret := &corev1.Secret{
		ObjectMeta: controllerruntime.ObjectMeta{Name: targetName, Namespace: targetSync.Namespace},
	}

	kubeconfigBytes, err := base64.StdEncoding.DecodeString(kubeconfig)
	if err != nil {
		return err
	}

	_, err = controllerruntime.CreateOrUpdate(ctx, c.targetClient, newSecret, func() error {
		newSecret.ObjectMeta.Labels = map[string]string{
			labelKeyTargetSync: labelValueOk,
		}
		newSecret.Data = map[string][]byte{
			kubeconfigKey: kubeconfigBytes,
		}
		newSecret.Type = corev1.SecretTypeOpaque
		return nil
	})

	return err
}

func (c *TargetSyncController) removeTargetsAndSecrets(ctx context.Context, targetSync *lsv1alpha1.TargetSync) error {
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	secrets := &corev1.SecretList{}
	if err := c.targetClient.List(ctx, secrets, client.InNamespace(targetSync.Namespace),
		client.MatchingLabels{labelKeyTargetSync: labelValueOk}); err != nil {
		logger.Error(err, "listing secrets for deleting targetsync object failed")
		return err
	}

	for _, secret := range secrets.Items {
		if !c.isTargetSyncSecret(secret.Name, targetSync) {
			secretLogger := logger.WithValues(lc.KeyResource, client.ObjectKeyFromObject(&secret).String())
			secretLogger.Info("deleting secret whose targetsync object is being deleted")
			if err := c.targetClient.Delete(ctx, &secret); err != nil {
				secretLogger.Error(err, "failed to delete secret whose targetsync object is being deleted")
				return err
			}
		}
	}

	targets := &lsv1alpha1.TargetList{}
	if err := c.targetClient.List(ctx, targets, client.InNamespace(targetSync.Namespace),
		client.MatchingLabels{labelKeyTargetSync: labelValueOk}); err != nil {
		logger.Error(err, "listing targets for deleting targetsync object failed")
		return err
	}

	for _, target := range targets.Items {
		targetLogger := logger.WithValues(lc.KeyResource, client.ObjectKeyFromObject(&target).String())
		targetLogger.Info("deleting target whose targetsync object is being deleted")
		if err := c.targetClient.Delete(ctx, &target); err != nil {
			targetLogger.Error(err, "failed to delete target whose targetsync object is being deleted")
			return err
		}
	}

	return nil
}

func (c *TargetSyncController) fetchTargetSyncs(ctx context.Context, targetSync *lsv1alpha1.TargetSync) (*lsv1alpha1.TargetSyncList, error) {
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	targetSyncs := &lsv1alpha1.TargetSyncList{}
	if err := c.targetClient.List(ctx, targetSyncs, client.InNamespace(targetSync.Namespace)); err != nil {
		logger.Error(err, "targetsync failed: could not fetch targetsync list")
		return nil, err
	}

	return targetSyncs, nil
}

func (c *TargetSyncController) fetchOldTargets(ctx context.Context, targetSync *lsv1alpha1.TargetSync) (map[string]*lsv1alpha1.Target, error) {
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	targets := &lsv1alpha1.TargetList{}
	if err := c.targetClient.List(ctx, targets, client.InNamespace(targetSync.Namespace), client.MatchingLabels{labelKeyTargetSync: labelValueOk}); err != nil {
		logger.Error(err, "targetsync failed: old targets could not be fetched")
		return nil, err
	}

	targetMap := map[string]*lsv1alpha1.Target{}
	for i := range targets.Items {
		next := &targets.Items[i]
		targetMap[next.Name] = next
	}

	return targetMap, nil
}

func (c *TargetSyncController) refreshToken(ctx context.Context, targetSync *lsv1alpha1.TargetSync, restConfig *rest.Config) error {
	if c.isTokenRotationEnabled(targetSync) && c.isTokenRotationDue(targetSync) {
		logger, ctx := logging.FromContextOrNew(ctx, nil)

		secret, kubeconfigObject, err := c.fetchSecretAndKubeconfigObject(ctx, targetSync)
		if err != nil {
			logger.Error(err, "fetching secret and kubeconfig failed for sync object")
			return err
		}

		serviceAccountName, user, err := c.getServiceAccountNameAndAuthInfo(kubeconfigObject)
		if err != nil {
			logger.Error(err, "fetching service account name and user failed for sync object")
			return err
		}

		// fetch new token
		newToken, err := c.fetchNewToken(ctx, targetSync.Spec.SourceNamespace, serviceAccountName, restConfig)
		if err != nil {
			logger.Error(err, "fetching new token for sync object")
			return err
		}

		user.Token = newToken

		err = c.rotateTokenInSecret(ctx, targetSync, secret, kubeconfigObject)
		if err != nil {
			logger.Error(err, "rotating token in secret failed for sync object")
			return err
		}
	}

	return nil
}

func (c *TargetSyncController) isTokenRotationEnabled(targetSync *lsv1alpha1.TargetSync) bool {
	return targetSync.Spec.TokenRotation != nil && targetSync.Spec.TokenRotation.Enabled
}

func (c *TargetSyncController) isTokenRotationDue(targetSync *lsv1alpha1.TargetSync) bool {
	return metav1.HasAnnotation(targetSync.ObjectMeta, lsv1alpha1.RotateTokenAnnotation) ||
		targetSync.Status.LastTokenRotationTime == nil ||
		time.Since(targetSync.Status.LastTokenRotationTime.Time) > tokenRotationInterval
}

func (c *TargetSyncController) fetchSecretAndKubeconfigObject(ctx context.Context,
	targetSync *lsv1alpha1.TargetSync) (*corev1.Secret, *clientcmdapi.Config, error) {

	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: targetSync.Namespace,
		Name:      targetSync.Spec.SecretRef.Name,
	}

	if err := c.targetClient.Get(ctx, secretKey, secret); err != nil {
		return nil, nil, err
	}

	kubeconfigBytes, ok := secret.Data[targetSync.Spec.SecretRef.Key]
	if !ok || len(kubeconfigBytes) == 0 {
		return nil, nil, fmt.Errorf("no kubeconfig in secret to rotate for sync object")
	}

	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfigBytes)
	if err != nil {
		return nil, nil, err
	}

	kubeConfig, err := clientConfig.RawConfig()
	if err != nil {
		return nil, nil, err
	}

	return secret, &kubeConfig, nil
}

func (c *TargetSyncController) rotateTokenInSecret(ctx context.Context, targetSync *lsv1alpha1.TargetSync, secret *corev1.Secret,
	kubeConfigObject *clientcmdapi.Config) error {

	kubeConfigObjectV1 := clientcmdapiv1.Config{}
	if err := clientcmdapiv1.Convert_api_Config_To_v1_Config(kubeConfigObject, &kubeConfigObjectV1, nil); err != nil {
		return err
	}

	kubeconfigBytes, err := yaml.Marshal(kubeConfigObjectV1)
	if err != nil {
		return err
	}

	secret.Data[targetSync.Spec.SecretRef.Key] = kubeconfigBytes

	if err := c.targetClient.Update(ctx, secret); err != nil {
		return err
	}

	return nil
}

func (c *TargetSyncController) getServiceAccountNameAndAuthInfo(
	kubeConfigObject *clientcmdapi.Config) (string, *clientcmdapi.AuthInfo, error) {

	authInfos := kubeConfigObject.AuthInfos
	if len(authInfos) != 1 {
		return "", nil, fmt.Errorf("authInfos in kubeconfig invalid for sync object")
	}

	serviceAccountName := ""
	var authInfo *clientcmdapi.AuthInfo

	for k, v := range authInfos {
		serviceAccountName = k
		authInfo = v
		break
	}

	return serviceAccountName, authInfo, nil
}

func (c *TargetSyncController) fetchNewToken(ctx context.Context, namespace, serviceAccountName string, restConfig *rest.Config) (string, error) {
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	treq := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: pointer.Int64(tokenExpirationSeconds),
		},
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		logger.Error(err, "fetching client set for refreshing token failed for sync object")
		return "", err
	}

	treq, err = clientset.CoreV1().ServiceAccounts(namespace).CreateToken(ctx,
		serviceAccountName, treq, metav1.CreateOptions{})
	if err != nil {
		logger.Error(err, "fetching token failed for sync object")
		return "", err
	}

	return treq.Status.Token, nil
}

func (c *TargetSyncController) deriveTargetNameFromShootName(shootName string) string {
	return shootName
}

func (c *TargetSyncController) isTargetSyncSecret(secretName string, targetSync *lsv1alpha1.TargetSync) bool {
	return secretName == targetSync.Spec.SecretRef.Name
}

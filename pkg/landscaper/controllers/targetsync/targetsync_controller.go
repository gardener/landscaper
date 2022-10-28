// SPDX-FileCopyrightText: 2022 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package targetsync

import (
	"context"
	"fmt"
	"time"

	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	"sigs.k8s.io/controller-runtime/pkg/predicate"

	lserrors "github.com/gardener/landscaper/apis/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kutils "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

// AddControllerToManagerForTargetSyncs adds the controller to the manager
func AddControllerToManagerForTargetSyncs(logger logging.Logger, mgr manager.Manager) error {
	log := logger.Reconciles("targetSync", "TargetSync")
	ctrl, err := NewTargetSyncController(log, mgr.GetClient())
	if err != nil {
		return err
	}

	predicates := builder.WithPredicates(predicate.Or(predicate.LabelChangedPredicate{},
		predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{}))

	return builder.ControllerManagedBy(mgr).
		For(&lsv1alpha1.TargetSync{}, predicates).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(ctrl)
}

// Controller is the TargetSync controller
type TargetSyncController struct {
	log      logging.Logger
	lsClient client.Client
}

// NewController returns a new TargetSync controller
func NewTargetSyncController(logger logging.Logger, c client.Client) (reconcile.Reconciler, error) {
	ctrl := &TargetSyncController{
		log:      logger,
		lsClient: c,
	}

	return ctrl, nil
}

// Reconcile reconciles requests for TargetSyncs
func (c *TargetSyncController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger, ctx := c.log.StartReconcileAndAddToContext(ctx, req)

	targetSync := &lsv1alpha1.TargetSync{}
	if err := c.lsClient.Get(ctx, req.NamespacedName, targetSync); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info(err.Error())
			return reconcile.Result{}, nil
		}
		logger.Error(err, "fetching targetSync object failed")
		return reconcile.Result{}, err
	}

	// set finalizer
	if targetSync.DeletionTimestamp.IsZero() && !kutils.HasFinalizer(targetSync, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(targetSync, lsv1alpha1.LandscaperFinalizer)
		if err := c.lsClient.Update(ctx, targetSync); err != nil {
			logger.Error(err, "adding finalizer to targetSync object failed")
			return reconcile.Result{}, err
		}
		// do not return here because the controller only watches for particular events and setting a finalizer is not part of this
	}

	if targetSync.DeletionTimestamp.IsZero() {
		if err := c.handleReconcile(ctx, targetSync); err != nil {
			logger.Error(err, "reconciling targetSync object failed")
			return reconcile.Result{}, err
		}
	} else {
		if err := c.handleDelete(ctx, targetSync); err != nil {
			logger.Error(err, "deleting targetSync object failed")
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: time.Minute * 5,
	}, nil
}

func (c *TargetSyncController) handleReconcile(ctx context.Context, targetSync *lsv1alpha1.TargetSync) error {
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyReconciledResource, client.ObjectKeyFromObject(targetSync).String()})

	errors := []error{}

	targetSyncs, oldTargets, err := c.fetchTargetSyncsAndOldTargets(ctx, targetSync)

	if err != nil {
		errors = append(errors, err)
	} else if len(targetSyncs.Items) > 1 {
		err = fmt.Errorf("more than one TargetSync object in the same namespace is not allowed")
		errors = append(errors, err)
	} else {
		sourceClient, err := getSourceClient(ctx, targetSync, c.lsClient, nil)
		if err != nil {
			logger.Error(err, "fetching source client for target sync object failed")
			errors = append(errors, err)
		} else {
			errors = c.handleSecrets(ctx, targetSync, sourceClient, oldTargets)
		}
	}

	errorStrings := []string{}
	for _, nextError := range errors {
		errorStrings = append(errorStrings, nextError.Error())
	}

	targetSync.Status.LastErrors = errorStrings
	targetSync.Status.ObservedGeneration = targetSync.GetGeneration()
	targetSync.Status.LastUpdateTime = metav1.Now()

	if err = c.lsClient.Status().Update(ctx, targetSync); err != nil {
		logger.Error(err, "updating status at the end of reconcile of target sync object failed")
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

		targetSync.Status.LastErrors = errorStrings
		targetSync.Status.ObservedGeneration = targetSync.GetGeneration()
		targetSync.Status.LastUpdateTime = metav1.Now()

		if internalErr := c.lsClient.Status().Update(ctx, targetSync); err != nil {
			logger.Error(err, "updating status with error for deleting target sync object failed")
			return internalErr
		}

		return err
	}

	controllerutil.RemoveFinalizer(targetSync, lsv1alpha1.LandscaperFinalizer)
	if err := c.lsClient.Update(ctx, targetSync); err != nil {
		logger.Error(err, "removing finalizer for deleting target sync object failed")
		return lserrors.NewWrappedError(err, "handleDelete", "RemoveFinalizer", err.Error())
	}

	return nil
}

func (c *TargetSyncController) handleSecrets(ctx context.Context, targetSync *lsv1alpha1.TargetSync,
	sourceClient client.Client, oldTargets map[string]*lsv1alpha1.Target) []error {
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	errors := []error{}

	secrFilter, err := newSecretFilter(targetSync.Spec.SecretNameExpression)
	if err != nil {
		logger.Error(err, "building secret filter of target sync object failed: "+targetSync.Spec.SecretNameExpression)
		errors = append(errors, err)
		return errors
	}

	secrets := &corev1.SecretList{}
	if err = sourceClient.List(ctx, secrets, client.InNamespace(targetSync.Spec.SourceNamespace)); err != nil {
		logger.Error(err, "fetching secret list for target sync object failed")
		errors = append(errors, err)
		return errors
	}

	for _, nextSecret := range secrets.Items {
		if secrFilter.shouldBeProcessed(&nextSecret) {
			delete(oldTargets, nextSecret.Name)

			if err = c.handleSecret(ctx, targetSync, &nextSecret); err != nil {
				msg := fmt.Sprintf("handling secret %s of target sync object failed", client.ObjectKeyFromObject(&nextSecret).String())
				logger.Error(err, msg)
				errors = append(errors, err)
			}
		}
	}

	if len(errors) == 0 {
		for key := range oldTargets {
			secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: targetSync.Namespace, Name: key}}
			if err = c.lsClient.Delete(ctx, &secret); err != nil {
				msg := fmt.Sprintf("deleting old secret %s of target sync object failed", client.ObjectKeyFromObject(&secret).String())
				logger.Error(err, msg)
				errors = append(errors, err)
			}

			target := lsv1alpha1.Target{ObjectMeta: metav1.ObjectMeta{Namespace: targetSync.Namespace, Name: key}}
			if err = c.lsClient.Delete(ctx, &target); err != nil {
				msg := fmt.Sprintf("deleting old target %s of target sync object failed", client.ObjectKeyFromObject(&target).String())
				logger.Error(err, msg)
				errors = append(errors, err)
			}
		}
	}

	return errors
}

func (c *TargetSyncController) handleSecret(ctx context.Context, targetSync *lsv1alpha1.TargetSync, secret *corev1.Secret) error {
	err := c.createOrUpdateTarget(ctx, targetSync, secret)
	if err != nil {
		return err
	}

	err = c.createOrUpdateSecret(ctx, targetSync, secret)
	return err
}

func (c *TargetSyncController) createOrUpdateTarget(ctx context.Context, targetSync *lsv1alpha1.TargetSync, secret *corev1.Secret) error {
	targetSpec := lsv1alpha1.TargetSpec{
		Type: lsv1alpha1.KubernetesClusterTargetType,
		SecretRef: &lsv1alpha1.SecretReference{
			ObjectReference: lsv1alpha1.ObjectReference{
				Name:      secret.Name,
				Namespace: targetSync.Namespace,
			},
			Key: lsv1alpha1.DefaultKubeconfigKey,
		},
	}

	newTarget := &lsv1alpha1.Target{
		ObjectMeta: controllerruntime.ObjectMeta{Name: secret.Name, Namespace: targetSync.Namespace},
	}

	_, err := controllerruntime.CreateOrUpdate(ctx, c.lsClient, newTarget, func() error {
		newTarget.Spec = targetSpec
		newTarget.ObjectMeta.Labels = map[string]string{
			labelKeyTargetSync: labelValueOk,
		}
		return nil
	})

	return err
}

func (c *TargetSyncController) createOrUpdateSecret(ctx context.Context, targetSync *lsv1alpha1.TargetSync, secret *corev1.Secret) error {
	newSecret := &corev1.Secret{
		ObjectMeta: controllerruntime.ObjectMeta{Name: secret.Name, Namespace: targetSync.Namespace},
	}

	_, err := controllerruntime.CreateOrUpdate(ctx, c.lsClient, newSecret, func() error {
		newSecret.ObjectMeta.Labels = map[string]string{
			labelKeyTargetSync: labelValueOk,
		}
		newSecret.Data = secret.Data
		newSecret.Type = secret.Type
		return nil
	})

	return err
}

func (c *TargetSyncController) removeTargetsAndSecrets(ctx context.Context, targetSync *lsv1alpha1.TargetSync) error {
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	secrets := &corev1.SecretList{}
	if err := c.lsClient.List(ctx, secrets, client.InNamespace(targetSync.Namespace),
		client.MatchingLabels{labelKeyTargetSync: labelValueOk}); err != nil {
		logger.Error(err, "listing secrets for deleting target sync object failed")
		return err
	}

	for _, nextSecret := range secrets.Items {
		if err := c.lsClient.Delete(ctx, &nextSecret); err != nil {
			logger.Error(err, "deleting secret for deleting target sync object failed")
			return err
		}
	}

	targets := &lsv1alpha1.TargetList{}
	if err := c.lsClient.List(ctx, targets, client.InNamespace(targetSync.Namespace),
		client.MatchingLabels{labelKeyTargetSync: labelValueOk}); err != nil {
		logger.Error(err, "listing targets for deleting target sync object failed")
		return err
	}

	for _, nextTarget := range targets.Items {
		if err := c.lsClient.Delete(ctx, &nextTarget); err != nil {
			logger.Error(err, "deleting target for deleting target sync object failed")
			return err
		}
	}

	return nil
}

func (c *TargetSyncController) fetchTargetSyncsAndOldTargets(ctx context.Context,
	targetSync *lsv1alpha1.TargetSync) (*lsv1alpha1.TargetSyncList, map[string]*lsv1alpha1.Target, error) {
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	targetSyncs := &lsv1alpha1.TargetSyncList{}
	if err := c.lsClient.List(ctx, targetSyncs, client.InNamespace(targetSync.Namespace)); err != nil {
		logger.Error(err, "fetching target sync list failed")
		return nil, nil, err
	}

	targets := &lsv1alpha1.TargetList{}
	if err := c.lsClient.List(ctx, targets, client.InNamespace(targetSync.Namespace),
		client.MatchingLabels{labelKeyTargetSync: labelValueOk}); err != nil {
		logger.Error(err, "fetching target list for target sync object failed")
		return nil, nil, err
	}

	targetMap := map[string]*lsv1alpha1.Target{}
	for i := range targets.Items {
		next := &targets.Items[i]
		targetMap[next.Name] = next
	}

	return targetSyncs, targetMap, nil
}

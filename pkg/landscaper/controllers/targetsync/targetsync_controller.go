// SPDX-FileCopyrightText: 2022 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package targetsync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kutils "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

// AddControllerToManagerForTargetSyncs adds the controller to the manager
func AddControllerToManagerForTargetSyncs(logger logging.Logger, mgr manager.Manager) error {
	log := logger.Reconciles("targetSync", "TargetSync")
	ctrl, err := NewTargetSyncController(log, mgr.GetClient(), mgr.GetScheme())
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
	log    logging.Logger
	client client.Client
}

// NewController returns a new TargetSync controller
func NewTargetSyncController(logger logging.Logger, c client.Client, scheme *runtime.Scheme) (reconcile.Reconciler, error) {
	ctrl := &TargetSyncController{
		log:    logger,
		client: c,
	}

	return ctrl, nil
}

// Reconcile reconciles requests for TargetSyncs
func (c *TargetSyncController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger, ctx := c.log.StartReconcileAndAddToContext(ctx, req)

	targetSync := &lsv1alpha1.TargetSync{}
	if err := c.client.Get(ctx, req.NamespacedName, targetSync); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// set finalizer
	if targetSync.DeletionTimestamp.IsZero() && !kutils.HasFinalizer(targetSync, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(targetSync, lsv1alpha1.LandscaperFinalizer)
		if err := c.client.Update(ctx, targetSync); err != nil {
			return reconcile.Result{}, err
		}
		// do not return here because the controller only watches for particular events and setting a finalizer is not part of this
	}

	if targetSync.DeletionTimestamp.IsZero() {
		if err := c.handleReconcile(ctx, targetSync); err != nil {
			return reconcile.Result{}, err
		}
	} else {
		if err := c.handleDelete(ctx, targetSync); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: time.Minute * 5,
	}, nil
}

func (c *TargetSyncController) handleReconcile(ctx context.Context, targetSync *lsv1alpha1.TargetSync) error {
	errors := []error{}

	targetSyncs, oldTargets, err := c.fetchTargetSyncsAndSyncedTargets(ctx, targetSync)

	if err != nil {
		errors = append(errors, err)
	} else if len(targetSyncs.Items) > 1 {
		err = fmt.Errorf("more than one TargetSync object in the same namespace is not allowed")
		errors = append(errors, err)
	} else {
		sourceClient, err := getSourceClient(ctx, targetSync, c.client, nil)
		if err != nil {
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

	if err = c.client.Status().Update(ctx, targetSync); err != nil {
		return err
	}

	if len(errors) > 0 {
		return errors[0]
	}
	return nil
}

func (c *TargetSyncController) handleDelete(ctx context.Context, targetSync *lsv1alpha1.TargetSync) error {
	errorStrings := []string{}

	err := c.removeTargetsAndSecrets(ctx, targetSync)
	if err != nil {
		errorStrings = append(errorStrings, err.Error())

		targetSync.Status.LastErrors = errorStrings
		targetSync.Status.ObservedGeneration = targetSync.GetGeneration()

		if internalErr := c.client.Status().Update(ctx, targetSync); err != nil {
			return internalErr
		}

		return err
	}

	controllerutil.RemoveFinalizer(targetSync, lsv1alpha1.LandscaperFinalizer)
	if err := c.client.Update(ctx, targetSync); err != nil {
		return lserrors.NewWrappedError(err, "handleDelete", "RemoveFinalizer", err.Error())
	}

	return nil
}

func (c *TargetSyncController) handleSecrets(ctx context.Context, targetSync *lsv1alpha1.TargetSync,
	sourceClient client.Client, oldTargets map[string]*lsv1alpha1.Target) []error {
	errors := []error{}

	secrFilter, err := newSecretFilter(targetSync.Spec.SecretNameExpression)
	if err != nil {
		errors = append(errors, err)
		return errors
	}

	secrets := &corev1.SecretList{}
	if err = sourceClient.List(ctx, secrets, client.InNamespace(targetSync.Spec.SourceNamespace)); err != nil {
		errors = append(errors, err)
		return errors
	}

	for _, nextSecret := range secrets.Items {
		if secrFilter.shouldBeProcessed(&nextSecret) {
			delete(oldTargets, nextSecret.Name)

			if err = c.handleSecret(ctx, targetSync, &nextSecret); err != nil {
				errors = append(errors, err)
			}
		}
	}

	if len(errors) == 0 {
		for key := range oldTargets {
			secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: targetSync.Namespace, Name: key}}
			if err = c.client.Delete(ctx, &secret); err != nil {
				errors = append(errors, err)
			}

			target := lsv1alpha1.Target{ObjectMeta: metav1.ObjectMeta{Namespace: targetSync.Namespace, Name: key}}
			if err = c.client.Delete(ctx, &target); err != nil {
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
	config := lsv1alpha1.KubernetesClusterTargetConfig{
		Kubeconfig: lsv1alpha1.ValueRef{
			SecretRef: &lsv1alpha1.SecretReference{
				ObjectReference: lsv1alpha1.ObjectReference{
					Name:      secret.Name,
					Namespace: targetSync.Namespace,
				},
				Key: lsv1alpha1.DefaultKubeconfigKey,
			},
		},
	}

	configBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}

	targetSpec := lsv1alpha1.TargetSpec{
		Configuration: lsv1alpha1.NewAnyJSON(configBytes),
		Type:          lsv1alpha1.KubernetesClusterTargetType,
	}

	newTarget := &lsv1alpha1.Target{
		ObjectMeta: controllerruntime.ObjectMeta{Name: secret.Name, Namespace: targetSync.Namespace},
	}

	_, err = controllerruntime.CreateOrUpdate(ctx, c.client, newTarget, func() error {
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

	_, err := controllerruntime.CreateOrUpdate(ctx, c.client, newSecret, func() error {
		newSecret.ObjectMeta.Labels = map[string]string{
			labelKeyTargetSync: labelValueOk,
		}
		newSecret.Data = map[string][]byte{
			lsv1alpha1.DefaultKubeconfigKey: secret.Data[lsv1alpha1.DefaultKubeconfigKey],
		}
		newSecret.Type = secret.Type
		return nil
	})

	return err
}

func (c *TargetSyncController) removeTargetsAndSecrets(ctx context.Context, targetSync *lsv1alpha1.TargetSync) error {
	secrets := &corev1.SecretList{}
	if err := c.client.List(ctx, secrets, client.InNamespace(targetSync.Namespace),
		client.MatchingLabels{labelKeyTargetSync: labelValueOk}); err != nil {
		return err
	}

	for _, nextSecret := range secrets.Items {
		if err := c.client.Delete(ctx, &nextSecret); err != nil {
			return err
		}
	}

	targets := &lsv1alpha1.TargetList{}
	if err := c.client.List(ctx, targets, client.InNamespace(targetSync.Namespace),
		client.MatchingLabels{labelKeyTargetSync: labelValueOk}); err != nil {
		return err
	}

	for _, nextTarget := range targets.Items {
		if err := c.client.Delete(ctx, &nextTarget); err != nil {
			return err
		}
	}

	return nil
}

func (c *TargetSyncController) fetchTargetSyncsAndSyncedTargets(ctx context.Context,
	targetSync *lsv1alpha1.TargetSync) (*lsv1alpha1.TargetSyncList, map[string]*lsv1alpha1.Target, error) {

	targetSyncs := &lsv1alpha1.TargetSyncList{}
	if err := c.client.List(ctx, targetSyncs, client.InNamespace(targetSync.Namespace)); err != nil {
		return nil, nil, err
	}

	targets := &lsv1alpha1.TargetList{}
	if err := c.client.List(ctx, targets, client.InNamespace(targetSync.Namespace),
		client.MatchingLabels{labelKeyTargetSync: labelValueOk}); err != nil {
		return nil, nil, err
	}

	targetMap := map[string]*lsv1alpha1.Target{}
	for i := range targets.Items {
		next := &targets.Items[i]
		targetMap[next.Name] = next
	}

	return targetSyncs, targetMap, nil
}

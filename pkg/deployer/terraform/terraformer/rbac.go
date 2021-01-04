// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package terraformer

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kutils "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// EnsureRBAC ensures that all the RBAC resources are created.
func (t *Terraformer) EnsureRBAC(ctx context.Context) error {
	if err := t.createOrUpdateRBAC(ctx); err != nil {
		return err
	}
	return t.waitForRBAC(ctx)
}

// createOrUpdateRBAC creates or updates the ServiceAccount, Role and Rolebinding.
func (t *Terraformer) createOrUpdateRBAC(ctx context.Context) error {
	t.log.Info("Creating Terraformer RBAC")
	if err := t.createOrUpdateServiceAccount(ctx); err != nil {
		return err
	}
	if err := t.createOrUpdateRole(ctx); err != nil {
		return err
	}
	return t.createOrUpdateRoleBinding(ctx)
}

// createOrUpdateServiceAccount creates or updates the ServiceAccount.
func (t *Terraformer) createOrUpdateServiceAccount(ctx context.Context) error {
	t.log.V(1).Info("Creating Terraformer ServiceAccount", "name", t.Name)
	serviceAccount := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: t.Namespace, Name: t.Name, Labels: t.Labels}}
	_, err := kutils.CreateOrUpdate(ctx, t.kubeClient, serviceAccount, func() error {
		return nil
	})
	return err
}

// createOrUpdateRole creates or updates the Role.
func (t *Terraformer) createOrUpdateRole(ctx context.Context) error {
	t.log.V(1).Info("Creating Terraformer Role", "name", t.Name)
	role := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Namespace: t.Namespace, Name: t.Name, Labels: t.Labels}}
	_, err := kutils.CreateOrUpdate(ctx, t.kubeClient, role, func() error {
		role.Rules = []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"configmaps", "secrets"},
			Verbs:     []string{"*"},
		}}
		return nil
	})
	return err
}

// createOrUpdateRoleBinding creates or updates the and Rolebinding.
func (t *Terraformer) createOrUpdateRoleBinding(ctx context.Context) error {
	t.log.V(1).Info("Creating Terraformer RoleBinding", "name", t.Name)
	roleBinding := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: t.Namespace, Name: t.Name, Labels: t.Labels}}
	_, err := kutils.CreateOrUpdate(ctx, t.kubeClient, roleBinding, func() error {
		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     t.Name,
		}
		roleBinding.Subjects = []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      t.Name,
			Namespace: t.Namespace,
		}}
		return nil
	})
	return err
}

// waitForRBAC waits for the RBAC resources to be created in the cluster.
func (t *Terraformer) waitForRBAC(ctx context.Context) error {
	pollCtx, cancel := context.WithTimeout(ctx, DeadlineCleaning)
	defer cancel()

	return wait.PollImmediateUntil(10*time.Second, func() (done bool, err error) {
		t.log.Info("Waiting for RBAC to be created...")
		key := kutils.ObjectKey(t.Name, t.Namespace)
		if err = t.kubeClient.Get(pollCtx, key, &corev1.ServiceAccount{}); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			t.log.Error(err, "unable to get service account", "serviceaccount", key.String())
			return false, err
		}

		if err = t.kubeClient.Get(pollCtx, key, &rbacv1.Role{}); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			t.log.Error(err, "unable to get role", "role", key.String())
			return false, err
		}

		if err = t.kubeClient.Get(pollCtx, key, &rbacv1.RoleBinding{}); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			t.log.Error(err, "unable to get rolebinding", "rolebinding", key.String())
			return false, err
		}
		return true, nil
	}, pollCtx.Done())
}

// cleanUpRBAC deletes all the RBAC resources.
func (t *Terraformer) cleanUpRBAC(ctx context.Context) error {
	t.log.Info("Cleaning up Terraformer RBAC")
	t.log.V(1).Info("Deleting Terraformer ServiceAccount", "name", t.Name)
	err := t.kubeClient.Delete(ctx, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: t.Namespace, Name: t.Name, Labels: t.Labels}})
	if client.IgnoreNotFound(err) != nil {
		return err
	}
	t.log.V(1).Info("Deleting Terraformer Role", "name", t.Name)
	err = t.kubeClient.Delete(ctx, &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Namespace: t.Namespace, Name: t.Name, Labels: t.Labels}})
	if client.IgnoreNotFound(err) != nil {
		return err
	}
	t.log.V(1).Info("Deleting Terraformer RoleBinding", "name", t.Name)
	err = t.kubeClient.Delete(ctx, &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Namespace: t.Namespace, Name: t.Name, Labels: t.Labels}})
	if client.IgnoreNotFound(err) != nil {
		return err
	}
	return nil
}

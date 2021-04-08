// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/deployer/lib/targetselector"
	lsversion "github.com/gardener/landscaper/pkg/version"
)

func GetAndCheckReconcile(log logr.Logger, lsClient client.Client, config *containerv1alpha1.Configuration) func(ctx context.Context, req reconcile.Request) (*lsv1alpha1.DeployItem, error) {
	return func(ctx context.Context, req reconcile.Request) (*lsv1alpha1.DeployItem, error) {
		logger := log.WithValues("resource", req.NamespacedName)
		logger.V(7).Info("Reconcile deploy item")

		deployItem := &lsv1alpha1.DeployItem{}
		if err := lsClient.Get(ctx, req.NamespacedName, deployItem); err != nil {
			if apierrors.IsNotFound(err) {
				logger.V(5).Info(err.Error())
				return nil, nil
			}
			return nil, err
		}

		if deployItem.Spec.Type != Type {
			logger.V(7).Info("DeployItem is of wrong type", "type", deployItem.Spec.Type)
			return nil, nil
		}

		if deployItem.Spec.Target != nil {
			target := &lsv1alpha1.Target{}
			if err := lsClient.Get(ctx, deployItem.Spec.Target.NamespacedName(), target); err != nil {
				return nil, fmt.Errorf("unable to get target for deploy item: %w", err)
			}
			if len(config.TargetSelector) != 0 {
				matched, err := targetselector.Match(target, config.TargetSelector)
				if err != nil {
					return nil, fmt.Errorf("unable to match target selector: %w", err)
				}
				if !matched {
					logger.V(5).Info("The deploy item's target does not match the given target selector")
					return nil, nil
				}
			}
		}

		return deployItem, nil
	}
}

// DefaultConfiguration sets the defaults for the container deployer configuration.
func DefaultConfiguration(obj *containerv1alpha1.Configuration) {
	containerv1alpha1.SetDefaults_Configuration(obj)
	if len(obj.InitContainer.Image) == 0 {
		version := lsversion.Get().GitVersion
		// default lsversion to latest if the gitversion is 0.0.0-dev
		if version == "0.0.0-dev" {
			version = "latest"
		}
		obj.InitContainer.Image = fmt.Sprintf("eu.gcr.io/gardener-project/landscaper/container-deployer-init:%s", version)
	}
	if len(obj.WaitContainer.Image) == 0 {
		version := lsversion.Get().GitVersion
		// default lsversion to latest if the gitversion is 0.0.0-dev
		if version == "0.0.0-dev" {
			version = "latest"
		}
		obj.WaitContainer.Image = fmt.Sprintf("eu.gcr.io/gardener-project/landscaper/container-deployer-wait:%s", version)
	}
}

// DecodeProviderStatus decodes a RawExtension to a container status.
func DecodeProviderStatus(raw *runtime.RawExtension) (*containerv1alpha1.ProviderStatus, error) {
	status := &containerv1alpha1.ProviderStatus{}
	if raw != nil {
		if _, _, err := serializer.NewCodecFactory(api.LandscaperScheme).UniversalDecoder().Decode(raw.Raw, nil, status); err != nil {
			return nil, err
		}
	}
	return status, nil
}

// EncodeProviderStatus encodes a container status to a RawExtension.
func EncodeProviderStatus(status *containerv1alpha1.ProviderStatus) (*runtime.RawExtension, error) {
	status.TypeMeta = metav1.TypeMeta{
		APIVersion: containerv1alpha1.SchemeGroupVersion.String(),
		Kind:       "ProviderStatus",
	}

	raw := &runtime.RawExtension{}
	obj := status.DeepCopyObject()
	if err := runtime.Convert_runtime_Object_To_runtime_RawExtension(&obj, raw, nil); err != nil {
		return &runtime.RawExtension{}, err
	}
	return raw, nil
}

// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"fmt"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/deployer/lib/targetselector"
	lsversion "github.com/gardener/landscaper/pkg/version"
)

func getAndCheckReconcile(ctx context.Context, lsClient client.Client, config containerv1alpha1.Configuration, key client.ObjectKey) (*lsv1alpha1.DeployItem, error) {
	logger, ctx := logging.FromContextOrNew(ctx, nil, lc.KeyResource, key.String())

	deployItem := &lsv1alpha1.DeployItem{}
	if err := read_write_layer.GetDeployItem(ctx, lsClient, key, deployItem); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Debug(err.Error())
			return nil, nil
		}
		return nil, err
	}

	if deployItem.Spec.Type != Type {
		logger.Debug("DeployItem is of wrong type", lc.KeyDeployItemType, deployItem.Spec.Type)
		return nil, nil
	}

	if deployItem.Spec.Target != nil {
		target := &lsv1alpha1.Target{}
		if err := lsClient.Get(ctx, deployItem.Spec.Target.NamespacedName(), target); err != nil {
			return nil, fmt.Errorf("unable to get target for deploy item: %w", err)
		}
		if len(config.TargetSelector) != 0 {
			matched, err := targetselector.MatchOne(target, config.TargetSelector)
			if err != nil {
				return nil, fmt.Errorf("unable to match target selector: %w", err)
			}
			if !matched {
				logger.Debug("The deploy item's target does not match the given target selector")
				return nil, nil
			}
		}
	}

	return deployItem, nil
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

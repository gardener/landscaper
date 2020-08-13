// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package container

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
)

// Delete handles the delete flow for container deploy item.
func (c *Container) Delete(ctx context.Context) error {
	c.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseDeleting
	if len(c.ProviderStatus.PodName) != 0 || !lsv1alpha1helper.IsCompletedExecutionPhase(c.DeployItem.Status.Phase) {
		// do default reconcile until the pod has finished
		return c.Reconcile(ctx)
	}

	controllerutil.RemoveFinalizer(c.DeployItem, lsv1alpha1.LandscaperFinalizer)
	return c.kubeClient.Update(ctx, c.DeployItem)
}

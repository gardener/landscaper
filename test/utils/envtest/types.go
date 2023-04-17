// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package envtest

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
)

var (
	InstallationGVK  schema.GroupVersionKind
	ExecutionGVK     schema.GroupVersionKind
	DeployItemGVK    schema.GroupVersionKind
	DataObjectGVK    schema.GroupVersionKind
	TargetGVK        schema.GroupVersionKind
	TargetSyncGVK    schema.GroupVersionKind
	ContextGVK       schema.GroupVersionKind
	SecretGVK        schema.GroupVersionKind
	ConfigMapGVK     schema.GroupVersionKind
	DeploymentGVK    schema.GroupVersionKind
	LSHealthCheckGVK schema.GroupVersionKind
)

func init() {
	var err error
	InstallationGVK, err = apiutil.GVKForObject(&lsv1alpha1.Installation{}, api.LandscaperScheme)
	utilruntime.Must(err)
	ExecutionGVK, err = apiutil.GVKForObject(&lsv1alpha1.Execution{}, api.LandscaperScheme)
	utilruntime.Must(err)
	DeployItemGVK, err = apiutil.GVKForObject(&lsv1alpha1.DeployItem{}, api.LandscaperScheme)
	utilruntime.Must(err)
	DataObjectGVK, err = apiutil.GVKForObject(&lsv1alpha1.DataObject{}, api.LandscaperScheme)
	utilruntime.Must(err)
	TargetGVK, err = apiutil.GVKForObject(&lsv1alpha1.Target{}, api.LandscaperScheme)
	utilruntime.Must(err)
	TargetSyncGVK, err = apiutil.GVKForObject(&lsv1alpha1.TargetSync{}, api.LandscaperScheme)
	utilruntime.Must(err)
	ContextGVK, err = apiutil.GVKForObject(&lsv1alpha1.Context{}, api.LandscaperScheme)
	utilruntime.Must(err)
	SecretGVK, err = apiutil.GVKForObject(&corev1.Secret{}, api.LandscaperScheme)
	utilruntime.Must(err)
	ConfigMapGVK, err = apiutil.GVKForObject(&corev1.ConfigMap{}, api.LandscaperScheme)
	utilruntime.Must(err)
	DeploymentGVK, err = apiutil.GVKForObject(&appsv1.Deployment{}, api.LandscaperScheme)
	utilruntime.Must(err)
	LSHealthCheckGVK, err = apiutil.GVKForObject(&lsv1alpha1.LsHealthCheck{}, api.LandscaperScheme)
	utilruntime.Must(err)
}

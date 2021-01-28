// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package envtest

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
)

var (
	InstallationGVK schema.GroupVersionKind
	ExecutionGVK    schema.GroupVersionKind
	DeployItemGVK   schema.GroupVersionKind
	DataObjectGVK   schema.GroupVersionKind
	TargetGVK       schema.GroupVersionKind
	SecretGVK       schema.GroupVersionKind
	ConfigMapGVK    schema.GroupVersionKind
)

func init() {
	var err error
	InstallationGVK, err = apiutil.GVKForObject(&lsv1alpha1.Installation{}, kubernetes.LandscaperScheme)
	utilruntime.Must(err)
	ExecutionGVK, err = apiutil.GVKForObject(&lsv1alpha1.Execution{}, kubernetes.LandscaperScheme)
	utilruntime.Must(err)
	DeployItemGVK, err = apiutil.GVKForObject(&lsv1alpha1.DeployItem{}, kubernetes.LandscaperScheme)
	utilruntime.Must(err)
	DataObjectGVK, err = apiutil.GVKForObject(&lsv1alpha1.DataObject{}, kubernetes.LandscaperScheme)
	utilruntime.Must(err)
	TargetGVK, err = apiutil.GVKForObject(&lsv1alpha1.Target{}, kubernetes.LandscaperScheme)
	utilruntime.Must(err)
	SecretGVK, err = apiutil.GVKForObject(&corev1.Secret{}, kubernetes.LandscaperScheme)
	utilruntime.Must(err)
	ConfigMapGVK, err = apiutil.GVKForObject(&corev1.ConfigMap{}, kubernetes.LandscaperScheme)
	utilruntime.Must(err)
}

// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package terraform

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	terraformv1alpha1 "github.com/gardener/landscaper/apis/deployer/terraform/v1alpha1"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/deployer/terraform/terraformer"
)

// AddControllerToManager adds the terraform deployer controller to the manager.
func AddControllerToManager(hostMgr manager.Manager, lsMgr manager.Manager, config *terraformv1alpha1.Configuration) error {
	c := NewController(
		ctrl.Log.WithName("controllers").WithName("TerraformDeployer"),
		lsMgr.GetClient(),
		hostMgr.GetClient(),
		hostMgr.GetConfig(),
		lsMgr.GetScheme(),
		config,
	)

	src := source.NewKindWithCache(&corev1.Pod{}, hostMgr.GetCache())
	return ctrl.NewControllerManagedBy(lsMgr).
		For(&lsv1alpha1.DeployItem{}).
		Watches(src, deployerlib.NewPodEventHandler(ConstructReconcileDeployItemRequest)).
		Complete(c)
}

func ConstructReconcileDeployItemRequest(obj metav1.Object) (reconcile.Request, bool) {
	var (
		req = reconcile.Request{}
		ok  bool
	)
	req.Name, ok = obj.GetLabels()[terraformer.LabelKeyItemName]
	if !ok {
		return req, false
	}
	req.Namespace, ok = obj.GetLabels()[terraformer.LabelKeyItemNamespace]
	if !ok {
		return req, false
	}
	return req, true
}

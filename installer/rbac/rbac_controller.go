package rbac

import (
	"github.com/gardener/landscaper/installer/resources"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
)

const (
	controllerServiceAccountName = "landscaper-controller"
)

func newControllerServiceAccountMutator(h *valuesHelper) resources.Mutator[*core.ServiceAccount] {
	return &resources.ServiceAccountMutator{
		Name:      controllerServiceAccountName,
		Namespace: h.resourceNamespace(),
		Labels:    h.rbacComponent.Labels(),
	}
}

func newControllerClusterRoleBindingMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRoleBinding] {
	return &resources.ClusterRoleBindingMutator{
		ClusterRoleBindingName:  controllerClusterRoleName(h),
		ClusterRoleName:         controllerClusterRoleName(h),
		ServiceAccountName:      controllerServiceAccountName,
		ServiceAccountNamespace: h.resourceNamespace(),
		Labels:                  h.rbacComponent.Labels(),
	}
}

func newControllerClusterRoleMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRole] {
	return &resources.ClusterRoleMutator{
		Name:   controllerClusterRoleName(h),
		Labels: h.rbacComponent.Labels(),
		Rules: []rbac.PolicyRule{
			{
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"customresourcedefinitions"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{"landscaper.gardener.cloud"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets", "configmaps"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts/token"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"clusterroles", "clusterrolebindings"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"get", "watch", "create", "update", "patch"},
			},
		},
	}
}

func controllerClusterRoleName(h *valuesHelper) string {
	return h.rbacComponent.ClusterScopedResourceName("controller")
}

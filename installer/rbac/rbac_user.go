package rbac

import (
	"github.com/gardener/landscaper/installer/resources"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
)

const (
	userServiceAccountName = "landscaper-user"
)

func newUserServiceAccountMutator(h *valuesHelper) resources.Mutator[*core.ServiceAccount] {
	return &resources.ServiceAccountMutator{
		Name:      userServiceAccountName,
		Namespace: h.resourceNamespace(),
		Labels:    h.rbacComponent.Labels(),
	}
}

func newUserClusterRoleBindingMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRoleBinding] {
	return &resources.ClusterRoleBindingMutator{
		ClusterRoleBindingName:  userClusterRoleName(h),
		ClusterRoleName:         userClusterRoleName(h),
		ServiceAccountName:      userServiceAccountName,
		ServiceAccountNamespace: h.resourceNamespace(),
		Labels:                  h.rbacComponent.Labels(),
	}
}

func newUserClusterRoleMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRole] {
	return &resources.ClusterRoleMutator{
		Name:   userClusterRoleName(h),
		Labels: h.rbacComponent.Labels(),
		Rules: []rbac.PolicyRule{
			{
				APIGroups: []string{"landscaper.gardener.cloud"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces", "secrets", "configmaps"},
				Verbs:     []string{"*"},
			},
		},
	}
}

func userClusterRoleName(h *valuesHelper) string {
	return h.rbacComponent.ClusterScopedResourceName("user")
}

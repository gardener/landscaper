package helmdeployer

import (
	"github.com/gardener/landscaper/installer/resources"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
)

func newServiceAccountMutator(h *valuesHelper) resources.Mutator[*core.ServiceAccount] {
	return &resources.ServiceAccountMutator{
		Name:      h.deployerFullName(),
		Namespace: h.hostNamespace(),
		Labels:    h.helmDeployerComponent.Labels(),
	}
}

func newClusterRoleBindingMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRoleBinding] {
	return &resources.ClusterRoleBindingMutator{
		ClusterRoleBindingName:  h.clusterRoleName(),
		ClusterRoleName:         h.clusterRoleName(),
		ServiceAccountName:      h.deployerFullName(),
		ServiceAccountNamespace: h.hostNamespace(),
		Labels:                  h.helmDeployerComponent.Labels(),
	}
}

func newClusterRoleMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRole] {
	return &resources.ClusterRoleMutator{
		Name:   h.clusterRoleName(),
		Labels: h.helmDeployerComponent.Labels(),
		Rules: []rbac.PolicyRule{
			{
				APIGroups: []string{"landscaper.gardener.cloud"},
				Resources: []string{"deployitems", "deployitems/status"},
				Verbs:     []string{"get", "list", "watch", "update"},
			},
			{
				APIGroups: []string{"landscaper.gardener.cloud"},
				Resources: []string{"targets", "contexts"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"landscaper.gardener.cloud"},
				Resources: []string{"syncobjects", "criticalproblems"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces", "pods", "configmaps"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts/token"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"get", "watch", "create", "update", "patch"},
			},
		},
	}
}

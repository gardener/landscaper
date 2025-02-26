package landscaper

import (
	"github.com/gardener/landscaper/installer/resources"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
)

func newServiceAccountMutator(h *valuesHelper) resources.Mutator[*core.ServiceAccount] {
	return &resources.ServiceAccountMutator{
		Name:      h.landscaperFullName(),
		Namespace: h.hostNamespace(),
		Labels:    h.landscaperLabels(),
	}
}

func newClusterRoleBindingMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRoleBinding] {
	return &resources.ClusterRoleBindingMutator{
		ClusterRoleBindingName:  h.clusterRoleName(),
		ClusterRoleName:         h.clusterRoleName(),
		ServiceAccountName:      h.landscaperFullName(),
		ServiceAccountNamespace: h.hostNamespace(),
		Labels:                  h.landscaperLabels(),
	}
}

func newClusterRoleMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRole] {
	return &resources.ClusterRoleMutator{
		Name:   h.clusterRoleName(),
		Labels: h.landscaperLabels(),
		Rules: []rbac.PolicyRule{
			{
				APIGroups: []string{"landscaper.gardener.cloud"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
			{
				// The agent contains a helm deployer to deploy other deployers.
				// Its unknown what deployers might need we have to give the agent all possible permissions for resources.
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}
}

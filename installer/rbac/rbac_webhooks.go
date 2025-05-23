package rbac

import (
	"github.com/gardener/landscaper/installer/resources"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
)

const (
	WebhooksServiceAccountName = "landscaper-webhooks"
)

func newWebhooksServiceAccountMutator(h *valuesHelper) resources.Mutator[*core.ServiceAccount] {
	return &resources.ServiceAccountMutator{
		Name:      WebhooksServiceAccountName,
		Namespace: h.resourceNamespace(),
		Labels:    h.rbacComponent.Labels(),
	}
}

func newWebhooksClusterRoleBindingMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRoleBinding] {
	return &resources.ClusterRoleBindingMutator{
		ClusterRoleBindingName:  webhooksClusterRoleName(h),
		ClusterRoleName:         webhooksClusterRoleName(h),
		ServiceAccountName:      WebhooksServiceAccountName,
		ServiceAccountNamespace: h.resourceNamespace(),
		Labels:                  h.rbacComponent.Labels(),
	}
}

func newWebhooksClusterRoleMutator(h *valuesHelper) resources.Mutator[*rbac.ClusterRole] {
	return &resources.ClusterRoleMutator{
		Name:   webhooksClusterRoleName(h),
		Labels: h.rbacComponent.Labels(),
		Rules: []rbac.PolicyRule{
			{
				APIGroups: []string{"landscaper.gardener.cloud"},
				Resources: []string{"installations"},
				Verbs:     []string{"list"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{"admissionregistration.k8s.io"},
				Resources: []string{"validatingwebhookconfigurations"},
				Verbs:     []string{"*"},
			},
		},
	}
}

func webhooksClusterRoleName(h *valuesHelper) string {
	return h.rbacComponent.ClusterScopedResourceName("webhooks")
}

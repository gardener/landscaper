package manifestdeployer

import (
	"fmt"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type clusterRoleDefinition struct {
	*valuesHelper
}

var _ ResourceDefinition[*v1.ClusterRole] = &clusterRoleDefinition{}

func newClusterRoleDefinition(b *valuesHelper) ResourceDefinition[*v1.ClusterRole] {
	return &clusterRoleDefinition{valuesHelper: b}
}

func (d *clusterRoleDefinition) String() string {
	return fmt.Sprintf("clusterrole %s", d.clusterRoleName())
}

func (d *clusterRoleDefinition) Empty() *v1.ClusterRole {
	return &v1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: d.clusterRoleName(),
		},
	}
}

func (b *clusterRoleDefinition) Mutate(r *v1.ClusterRole) error {
	r.ObjectMeta.Labels = b.deployerLabels()
	r.Rules = []v1.PolicyRule{
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
			Resources: []string{"namespaces", "pods"},
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
	}
	return nil
}

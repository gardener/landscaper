package manifestdeployer

import (
	"fmt"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type clusterRoleBindingDefinition struct {
	*valuesHelper
}

var _ ResourceDefinition[*v1.ClusterRoleBinding] = &clusterRoleBindingDefinition{}

func newClusterRoleBindingDefinition(b *valuesHelper) ResourceDefinition[*v1.ClusterRoleBinding] {
	return &clusterRoleBindingDefinition{valuesHelper: b}
}

func (d *clusterRoleBindingDefinition) String() string {
	return fmt.Sprintf("clusterrolebinding %s", d.clusterRoleName())
}

func (d *clusterRoleBindingDefinition) Empty() *v1.ClusterRoleBinding {
	return &v1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: d.clusterRoleName(),
		},
	}
}

func (d *clusterRoleBindingDefinition) Mutate(r *v1.ClusterRoleBinding) error {
	r.ObjectMeta.Labels = d.deployerLabels()
	r.RoleRef = v1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     d.clusterRoleName(),
	}
	r.Subjects = []v1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      d.deployerFullName(),
			Namespace: d.hostNamespace(),
		},
	}
	return nil
}

package helmdeployer

import (
	"fmt"
	"github.com/gardener/landscaper/installer/resources"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type clusterRoleBindingMutator struct {
	*valuesHelper
}

var _ resources.Mutator[*v1.ClusterRoleBinding] = &clusterRoleBindingMutator{}

func newClusterRoleBindingMutator(b *valuesHelper) resources.Mutator[*v1.ClusterRoleBinding] {
	return &clusterRoleBindingMutator{valuesHelper: b}
}

func (d *clusterRoleBindingMutator) String() string {
	return fmt.Sprintf("clusterrolebinding %s", d.clusterRoleName())
}

func (d *clusterRoleBindingMutator) Empty() *v1.ClusterRoleBinding {
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

func (d *clusterRoleBindingMutator) Mutate(r *v1.ClusterRoleBinding) error {
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

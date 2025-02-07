package resources

import (
	"fmt"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterRoleBindingMutator struct {
	ClusterRoleBindingName  string
	ClusterRoleName         string
	ServiceAccountName      string
	ServiceAccountNamespace string
	Labels                  map[string]string
}

var _ Mutator[*v1.ClusterRoleBinding] = &ClusterRoleBindingMutator{}

func (m *ClusterRoleBindingMutator) String() string {
	return fmt.Sprintf("clusterrolebinding %s", m.ClusterRoleBindingName)
}

func (m *ClusterRoleBindingMutator) Empty() *v1.ClusterRoleBinding {
	return &v1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: m.ClusterRoleBindingName,
		},
	}
}

func (m *ClusterRoleBindingMutator) Mutate(r *v1.ClusterRoleBinding) error {
	r.ObjectMeta.Labels = m.Labels
	r.RoleRef = v1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     m.ClusterRoleName,
	}
	r.Subjects = []v1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      m.ServiceAccountName,
			Namespace: m.ServiceAccountNamespace,
		},
	}
	return nil
}

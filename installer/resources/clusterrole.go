package resources

import (
	"fmt"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterRoleMutator struct {
	Name   string
	Labels map[string]string
	Rules  []v1.PolicyRule
}

var _ Mutator[*v1.ClusterRole] = &ClusterRoleMutator{}

func (m *ClusterRoleMutator) String() string {
	return fmt.Sprintf("clusterrole %s", m.Name)
}

func (m *ClusterRoleMutator) Empty() *v1.ClusterRole {
	return &v1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: m.Name,
		},
	}
}

func (m *ClusterRoleMutator) Mutate(r *v1.ClusterRole) error {
	r.ObjectMeta.Labels = m.Labels
	r.Rules = m.Rules
	return nil
}

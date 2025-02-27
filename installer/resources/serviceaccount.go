package resources

import (
	"fmt"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceAccountMutator struct {
	Name      string
	Namespace string
	Labels    map[string]string
}

var _ Mutator[*core.ServiceAccount] = &ServiceAccountMutator{}

func (m *ServiceAccountMutator) String() string {
	return fmt.Sprintf("service account %s/%s", m.Namespace, m.Name)
}

func (m *ServiceAccountMutator) Empty() *core.ServiceAccount {
	return &core.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
	}
}

func (m *ServiceAccountMutator) Mutate(s *core.ServiceAccount) error {
	s.ObjectMeta.Labels = m.Labels
	return nil
}

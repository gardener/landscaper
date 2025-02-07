package manifestdeployer

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type serviceAccountDefinition struct {
	*valuesHelper
}

var _ ResourceDefinition[*v1.ServiceAccount] = &serviceAccountDefinition{}

func newServiceAccountDefinition(b *valuesHelper) ResourceDefinition[*v1.ServiceAccount] {
	return &serviceAccountDefinition{valuesHelper: b}
}

func (d *serviceAccountDefinition) String() string {
	return fmt.Sprintf("service account %s/%s", d.hostNamespace(), d.deployerFullName())
}

func (d *serviceAccountDefinition) Empty() *v1.ServiceAccount {
	return &v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.deployerFullName(),
			Namespace: d.hostNamespace(),
		},
	}
}

func (d *serviceAccountDefinition) Mutate(s *v1.ServiceAccount) error {
	s.ObjectMeta.Labels = d.deployerLabels()
	return nil
}

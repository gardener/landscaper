package helmdeployer

import (
	"fmt"
	"github.com/gardener/landscaper/installer/resources"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type serviceAccountMutator struct {
	*valuesHelper
}

var _ resources.Mutator[*core.ServiceAccount] = &serviceAccountMutator{}

func newServiceAccountMutator(b *valuesHelper) resources.Mutator[*core.ServiceAccount] {
	return &serviceAccountMutator{valuesHelper: b}
}

func (d *serviceAccountMutator) String() string {
	return fmt.Sprintf("service account %s/%s", d.hostNamespace(), d.deployerFullName())
}

func (d *serviceAccountMutator) Empty() *core.ServiceAccount {
	return &core.ServiceAccount{
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

func (d *serviceAccountMutator) Mutate(s *core.ServiceAccount) error {
	s.ObjectMeta.Labels = d.deployerLabels()
	return nil
}

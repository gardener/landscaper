package landscaper

import (
	"fmt"
	"github.com/gardener/landscaper/installer/resources"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type serviceMutator struct {
	*valuesHelper
}

var _ resources.Mutator[*core.Service] = &serviceMutator{}

func newServiceMutator(b *valuesHelper) resources.Mutator[*core.Service] {
	return &serviceMutator{valuesHelper: b}
}

func (m *serviceMutator) String() string {
	return fmt.Sprintf("landscaper service %s/%s", m.hostNamespace(), m.landscaperFullName())
}

func (m *serviceMutator) Empty() *core.Service {
	return &core.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.landscaperFullName(),
			Namespace: m.hostNamespace(),
		},
	}
}

func (m *serviceMutator) Mutate(r *core.Service) error {
	r.ObjectMeta.Labels = m.controllerComponent.Labels()
	r.Spec = core.ServiceSpec{
		Ports: []core.ServicePort{
			{
				Name:       "http",
				Port:       m.values.Controller.Service.Port,
				TargetPort: intstr.FromString("http"),
				Protocol:   "TCP",
			},
		},
		Selector: m.controllerComponent.SelectorLabels(),
		Type:     core.ServiceType(m.values.Controller.Service.Type),
	}
	return nil
}

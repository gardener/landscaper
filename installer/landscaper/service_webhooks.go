package landscaper

import (
	"fmt"
	"github.com/gardener/landscaper/installer/resources"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type webhooksServiceMutator struct {
	*valuesHelper
}

var _ resources.Mutator[*core.Service] = &webhooksServiceMutator{}

func newWebhooksServiceMutator(b *valuesHelper) resources.Mutator[*core.Service] {
	return &webhooksServiceMutator{valuesHelper: b}
}

func (m *webhooksServiceMutator) String() string {
	return fmt.Sprintf("landscaper webhooks service %s/%s", m.hostNamespace(), m.landscaperWebhooksFullName())
}

func (m *webhooksServiceMutator) Empty() *core.Service {
	return &core.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.landscaperWebhooksFullName(),
			Namespace: m.hostNamespace(),
		},
	}
}

func (m *webhooksServiceMutator) Mutate(r *core.Service) error {
	r.ObjectMeta.Labels = m.webhooksComponent.Labels()
	r.Spec = core.ServiceSpec{
		Ports: []core.ServicePort{
			{
				Name:       "webhooks",
				Port:       m.values.WebhooksServer.ServicePort,
				TargetPort: intstr.FromInt32(m.values.WebhooksServer.ServicePort),
				Protocol:   "TCP",
			},
		},
		Selector: m.webhooksComponent.SelectorLabels(),
		Type:     core.ServiceType(m.values.WebhooksServer.Service.Type),
	}
	return nil
}

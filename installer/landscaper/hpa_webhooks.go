package landscaper

import (
	"fmt"
	"github.com/gardener/landscaper/installer/resources"
	v2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

type webhooksHPAMutator struct {
	*valuesHelper
}

var _ resources.Mutator[*v2.HorizontalPodAutoscaler] = &webhooksHPAMutator{}

func newWebhooksHPAMutator(b *valuesHelper) resources.Mutator[*v2.HorizontalPodAutoscaler] {
	return &webhooksHPAMutator{valuesHelper: b}
}

func (m *webhooksHPAMutator) String() string {
	return fmt.Sprintf("hpa %s/%s", m.hostNamespace(), m.landscaperWebhooksFullName())
}

func (m *webhooksHPAMutator) Empty() *v2.HorizontalPodAutoscaler {
	return &v2.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "autoscaling/v2",
			Kind:       "HorizontalPodAutoscaler",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.landscaperWebhooksFullName(),
			Namespace: m.hostNamespace(),
		},
	}
}

func (m *webhooksHPAMutator) Mutate(r *v2.HorizontalPodAutoscaler) error {
	r.ObjectMeta.Labels = m.webhooksComponent.Labels()
	r.Spec = v2.HorizontalPodAutoscalerSpec{
		ScaleTargetRef: v2.CrossVersionObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       m.landscaperWebhooksFullName(),
		},
		MinReplicas: ptr.To[int32](2),
		MaxReplicas: m.values.WebhooksServer.HPA.MaxReplicas,
		Metrics: []v2.MetricSpec{
			{
				Type: "Resource",
				Resource: &v2.ResourceMetricSource{
					Name: "cpu",
					Target: v2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: m.values.WebhooksServer.HPA.AverageCpuUtilization,
					},
				},
			},
			{
				Type: "Resource",
				Resource: &v2.ResourceMetricSource{
					Name: "memory",
					Target: v2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: m.values.WebhooksServer.HPA.AverageMemoryUtilization,
					},
				},
			},
		},
	}
	return nil
}

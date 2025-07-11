package landscaper

import (
	"fmt"
	"github.com/gardener/landscaper/installer/resources"
	v2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

type mainHPAMutator struct {
	*valuesHelper
}

var _ resources.Mutator[*v2.HorizontalPodAutoscaler] = &mainHPAMutator{}

func newMainHPAMutator(b *valuesHelper) resources.Mutator[*v2.HorizontalPodAutoscaler] {
	return &mainHPAMutator{valuesHelper: b}
}

func (m *mainHPAMutator) String() string {
	return fmt.Sprintf("hpa %s/%s", m.hostNamespace(), m.landscaperMainFullName())
}

func (m *mainHPAMutator) Empty() *v2.HorizontalPodAutoscaler {
	return &v2.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "autoscaling/v2",
			Kind:       "HorizontalPodAutoscaler",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.landscaperMainFullName(),
			Namespace: m.hostNamespace(),
		},
	}
}

func (m *mainHPAMutator) Mutate(r *v2.HorizontalPodAutoscaler) error {
	r.ObjectMeta.Labels = m.controllerMainComponent.Labels()
	r.Spec = v2.HorizontalPodAutoscalerSpec{
		ScaleTargetRef: v2.CrossVersionObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       m.landscaperMainFullName(),
		},
		MinReplicas: ptr.To[int32](1),
		MaxReplicas: m.values.Controller.HPAMain.MaxReplicas,
		Metrics: []v2.MetricSpec{
			{
				Type: "Resource",
				Resource: &v2.ResourceMetricSource{
					Name: "cpu",
					Target: v2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: m.values.Controller.HPAMain.AverageCpuUtilization,
					},
				},
			},
			{
				Type: "Resource",
				Resource: &v2.ResourceMetricSource{
					Name: "memory",
					Target: v2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: m.values.Controller.HPAMain.AverageMemoryUtilization,
					},
				},
			},
		},
	}
	return nil
}

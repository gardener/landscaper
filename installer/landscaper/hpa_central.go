package landscaper

import (
	"fmt"
	"github.com/gardener/landscaper/installer/resources"
	v2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

type centralHPAMutator struct {
	*valuesHelper
}

var _ resources.Mutator[*v2.HorizontalPodAutoscaler] = &centralHPAMutator{}

func newCentralHPAMutator(b *valuesHelper) resources.Mutator[*v2.HorizontalPodAutoscaler] {
	return &centralHPAMutator{valuesHelper: b}
}

func (m *centralHPAMutator) String() string {
	return fmt.Sprintf("hpa %s/%s", m.hostNamespace(), m.landscaperFullName())
}

func (m *centralHPAMutator) Empty() *v2.HorizontalPodAutoscaler {
	return &v2.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "autoscaling/v2",
			Kind:       "HorizontalPodAutoscaler",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.landscaperFullName(),
			Namespace: m.hostNamespace(),
		},
	}
}

func (m *centralHPAMutator) Mutate(r *v2.HorizontalPodAutoscaler) error {
	r.ObjectMeta.Labels = m.controllerComponent.Labels()
	r.Spec = v2.HorizontalPodAutoscalerSpec{
		ScaleTargetRef: v2.CrossVersionObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       m.landscaperFullName(),
		},
		MinReplicas: ptr.To[int32](1),
		MaxReplicas: 1,
		Metrics: []v2.MetricSpec{
			{
				Type: "Resource",
				Resource: &v2.ResourceMetricSource{
					Name: "cpu",
					Target: v2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: ptr.To[int32](80),
					},
				},
			},
			{
				Type: "Resource",
				Resource: &v2.ResourceMetricSource{
					Name: "memory",
					Target: v2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: ptr.To[int32](80),
					},
				},
			},
		},
	}
	return nil
}

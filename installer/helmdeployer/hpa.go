package helmdeployer

import (
	"fmt"
	"github.com/gardener/landscaper/installer/resources"
	v2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

type hpaMutator struct {
	*valuesHelper
}

var _ resources.Mutator[*v2.HorizontalPodAutoscaler] = &hpaMutator{}

func newHPAMutator(b *valuesHelper) resources.Mutator[*v2.HorizontalPodAutoscaler] {
	return &hpaMutator{valuesHelper: b}
}

func (d *hpaMutator) String() string {
	return fmt.Sprintf("hpa %s/%s", d.hostNamespace(), d.deployerFullName())
}

func (d *hpaMutator) Empty() *v2.HorizontalPodAutoscaler {
	return &v2.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "autoscaling/v2",
			Kind:       "HorizontalPodAutoscaler",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.deployerFullName(),
			Namespace: d.hostNamespace(),
		},
	}
}

func (d *hpaMutator) Mutate(r *v2.HorizontalPodAutoscaler) error {
	r.ObjectMeta.Labels = d.helmDeployerComponent.Labels()
	r.Spec = v2.HorizontalPodAutoscalerSpec{
		ScaleTargetRef: v2.CrossVersionObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       d.deployerFullName(),
		},
		MinReplicas: ptr.To[int32](1),
		MaxReplicas: d.values.HPA.MaxReplicas,
		Metrics: []v2.MetricSpec{
			{
				Type: "Resource",
				Resource: &v2.ResourceMetricSource{
					Name: "cpu",
					Target: v2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: d.values.HPA.AverageCpuUtilization,
					},
				},
			},
			{
				Type: "Resource",
				Resource: &v2.ResourceMetricSource{
					Name: "memory",
					Target: v2.MetricTarget{
						Type:               "Utilization",
						AverageUtilization: d.values.HPA.AverageMemoryUtilization,
					},
				},
			},
		},
	}
	return nil
}

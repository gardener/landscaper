package manifestdeployer

import (
	"fmt"
	v2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

type hpaDefinition struct {
	*valuesHelper
}

var _ ResourceDefinition[*v2.HorizontalPodAutoscaler] = &hpaDefinition{}

func newHPADefinition(b *valuesHelper) ResourceDefinition[*v2.HorizontalPodAutoscaler] {
	return &hpaDefinition{valuesHelper: b}
}

func (d *hpaDefinition) String() string {
	return fmt.Sprintf("hpa %s/%s", d.hostNamespace(), d.deployerFullName())
}

func (d *hpaDefinition) Empty() *v2.HorizontalPodAutoscaler {
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

func (d *hpaDefinition) Mutate(r *v2.HorizontalPodAutoscaler) error {
	r.ObjectMeta.Labels = d.deployerLabels()
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

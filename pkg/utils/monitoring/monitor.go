package monitoring

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"time"

	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
)

type Monitor struct {
	namespace  string
	hostClient client.Client
}

func NewMonitor(namespace string, hostClient client.Client) *Monitor {
	return &Monitor{
		namespace:  namespace,
		hostClient: hostClient,
	}
}

func (m *Monitor) StartMonitoring(ctx context.Context, logger logging.Logger) {
	log := logger.WithName("landscaper-monitoring")
	ctx = logging.NewContext(ctx, log)

	log.Info("monitor: starting periodical landscaper monitoring")

	wait.UntilWithContext(ctx, m.monitorHpas, time.Minute)
}

func (m *Monitor) monitorHpas(ctx context.Context) {
	log, ctx := logging.FromContextOrNew(ctx, nil)
	log.Info("monitor: starting monitoring hpas")

	hpas := &v2.HorizontalPodAutoscalerList{}
	if err := m.hostClient.List(ctx, hpas, client.InNamespace(m.namespace)); err != nil {
		log.Error(err, "monitor: failed to list hpas")
		return
	}

	for i := range hpas.Items {
		hpa := &hpas.Items[i]

		keyValueList := []interface{}{
			lc.KeyResource, client.ObjectKey{Namespace: m.namespace, Name: hpa.Spec.ScaleTargetRef.Name},
			"currentReplicas", hpa.Status.CurrentReplicas,
		}

		for j := range hpa.Status.CurrentMetrics {
			metric := &hpa.Status.CurrentMetrics[j]
			if metric.Type == v2.ResourceMetricSourceType && metric.Resource != nil {
				if metric.Resource.Name == v1.ResourceMemory {
					keyValueList = append(keyValueList, "memoryAverageUtilization", metric.Resource.Current.AverageUtilization)
					keyValueList = append(keyValueList, "memoryAverageValue", metric.Resource.Current.AverageValue)
				} else if metric.Resource.Name == v1.ResourceCPU {
					keyValueList = append(keyValueList, "cpuAverageUtilization", metric.Resource.Current.AverageUtilization)
					keyValueList = append(keyValueList, "cpuAverageValue", metric.Resource.Current.AverageValue)
				}
			}
		}

		log.Info("HPA Statistics", keyValueList...)
	}
}

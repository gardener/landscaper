package monitoring

import (
	"context"
	"time"

	v2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
)

const (
	keyCurrentReplicas          = "currentReplicas"
	keyDesiredReplicas          = "desiredReplicas"
	keyMemoryAverageUtilization = "memoryAverageUtilization"
	keyMemoryAverageValue       = "memoryAverageValue"
	keyCpuAverageUtilization    = "cpuAverageUtilization"
	keyCpuAverageValue          = "cpuAverageValue"
)

type Monitor struct {
	namespace          string
	lsUncachedClient   client.Client
	lsCachedClient     client.Client
	hostUncachedClient client.Client
	hostCachedClient   client.Client
}

func NewMonitor(lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient client.Client, namespace string) *Monitor {
	return &Monitor{
		namespace:          namespace,
		lsUncachedClient:   lsUncachedClient,
		lsCachedClient:     lsCachedClient,
		hostUncachedClient: hostUncachedClient,
		hostCachedClient:   hostCachedClient,
	}
}

// StartMonitoring is a blocking method that periodically logs data from the status of HorizontalPodAutoscaler objects.
// It takes into account all HPAs in the same namespace as the current landscaper deployment. In this way it allows a
// simple monitoring of the cpu and memory consumption of the landscaper pods.
func (m *Monitor) StartMonitoring(ctx context.Context, logger logging.Logger) {
	log := logger.WithName("landscaper-monitoring")
	ctx = logging.NewContext(ctx, log)

	log.Info("monitor: starting periodical landscaper monitoring")

	wait.UntilWithContext(ctx, m.monitorHpas, time.Minute)
}

func (m *Monitor) monitorHpas(ctx context.Context) {
	log, ctx := logging.FromContextOrNew(ctx, nil)
	log.Debug("monitor: starting monitoring hpas")

	hpas := &v2.HorizontalPodAutoscalerList{}
	if err := m.hostUncachedClient.List(ctx, hpas, client.InNamespace(m.namespace)); err != nil {
		log.Error(err, "monitor: failed to list hpas")
		return
	}

	for i := range hpas.Items {
		hpa := &hpas.Items[i]

		shouldLog := false

		keyValueList := []interface{}{
			lc.KeyResource, client.ObjectKey{Namespace: m.namespace, Name: hpa.Spec.ScaleTargetRef.Name}.String(),
			keyCurrentReplicas, hpa.Status.CurrentReplicas,
			keyDesiredReplicas, hpa.Status.DesiredReplicas,
		}

		if hpa.Status.CurrentReplicas > 2 || hpa.Status.DesiredReplicas > 2 {
			shouldLog = true
		}

		for j := range hpa.Status.CurrentMetrics {
			metric := &hpa.Status.CurrentMetrics[j]
			if metric.Type == v2.ResourceMetricSourceType && metric.Resource != nil {
				switch metric.Resource.Name {
				case v1.ResourceMemory:
					keyValueList = append(keyValueList, keyMemoryAverageUtilization, metric.Resource.Current.AverageUtilization)
					keyValueList = append(keyValueList, keyMemoryAverageValue, metric.Resource.Current.AverageValue)
					if metric.Resource.Current.AverageUtilization != nil && *metric.Resource.Current.AverageUtilization > 50 {
						shouldLog = true
					}
				case v1.ResourceCPU:
					keyValueList = append(keyValueList, keyCpuAverageUtilization, metric.Resource.Current.AverageUtilization)
					keyValueList = append(keyValueList, keyCpuAverageValue, metric.Resource.Current.AverageValue)
					if metric.Resource.Current.AverageUtilization != nil && *metric.Resource.Current.AverageUtilization > 50 {
						shouldLog = true
					}
				}
			}
		}

		if shouldLog {
			log.Info("HPA Statistics", keyValueList...)
		}
	}
}

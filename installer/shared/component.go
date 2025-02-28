package shared

import (
	"fmt"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// application: landscaper
// components:
//   - controller (controller-central)
//   - main-controller (controller-main)
//   - webhooks-server
//   - manifest-deployer
//   - helm-deployer

const (
	landscaperApplicationName = "landscaper"
)

const (
	LabelAppName        = "app.kubernetes.io/name"
	LabelAppInstance    = "app.kubernetes.io/instance"
	LabelComponent      = "app.kubernetes.io/component"
	LabelVersion        = "app.kubernetes.io/version"
	LabelManagedBy      = "app.kubernetes.io/managed-by"
	LabelValueManagedBy = "landscaper-provider"
	LabelTopology       = "landscaper.gardener.cloud/topology"
	LabelTopologyNs     = "landscaper.gardener.cloud/topology-ns"
)

type Component struct {
	Instance        // for example "test0001-abcdefgh"
	Version  string // for example "v1.0.0"
	Name     string // for example "main", "central", "webhooks"
}

func (c *Component) applicationAndInstance() string {
	return fmt.Sprintf("%s-%s", landscaperApplicationName, c.Instance)
}

func (c *Component) ComponentAndInstance() string {
	return fmt.Sprintf("%s-%s", c.Name, c.Instance)
}

func (c *Component) Labels() map[string]string {
	labels := map[string]string{}
	maps.Copy(labels, c.InfoLabels())
	maps.Copy(labels, c.SelectorLabels())
	return labels
}

func (c *Component) DeploymentTemplateLabels() map[string]string {
	labels := map[string]string{}
	maps.Copy(labels, c.TopologyLabels())
	maps.Copy(labels, c.SelectorLabels())
	return labels
}

func (c *Component) SelectorLabels() map[string]string {
	return map[string]string{
		LabelAppName:     landscaperApplicationName,
		LabelAppInstance: c.applicationAndInstance(),
		LabelComponent:   c.Name,
	}
}

func (c *Component) InfoLabels() map[string]string {
	return map[string]string{
		LabelVersion:   c.Version,
		LabelManagedBy: LabelValueManagedBy,
	}
}

func (c *Component) TopologyLabels() map[string]string {
	return map[string]string{
		LabelTopology:   c.Name,
		LabelTopologyNs: c.Namespace(),
	}
}

func (c *Component) TopologySpreadConstraints() []corev1.TopologySpreadConstraint {
	return []corev1.TopologySpreadConstraint{
		{
			MaxSkew:           1,
			TopologyKey:       "topology.kubernetes.io/zone",
			WhenUnsatisfiable: "ScheduleAnyway",
			LabelSelector:     &metav1.LabelSelector{MatchLabels: c.TopologyLabels()},
		},
		{
			MaxSkew:           1,
			TopologyKey:       "kubernetes.io/hostname",
			WhenUnsatisfiable: "ScheduleAnyway",
			LabelSelector:     &metav1.LabelSelector{MatchLabels: c.TopologyLabels()},
		},
	}
}

// ClusterScopedDefaultResourceName returns the default name for a cluster-scoped resource:
// "landscaper:<instance>:<component>", for example "landscaper:test0001-abcdefgh:manifest-deployer".
func (c *Component) ClusterScopedDefaultResourceName() string {
	return fmt.Sprintf("%s:%s:%s", landscaperApplicationName, c.Instance, c.Name)
}

// ClusterScopedResourceName returns the name for a cluster-scoped resource with a given suffix:
// "landscaper:<instance>:<component>:<suffix>", for example "landscaper:test0001-abcdefgh:landscaper-rbac:user".
func (c *Component) ClusterScopedResourceName(suffix string) string {
	return fmt.Sprintf("%s:%s:%s:%s", landscaperApplicationName, c.Instance, c.Name, suffix)
}

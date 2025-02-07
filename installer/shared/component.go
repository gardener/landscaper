package shared

import (
	"fmt"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	applicationLandscaper = "landscaper"
)

const (
	labelAppName        = "app.kubernetes.io/name"
	labelAppInstance    = "app.kubernetes.io/instance"
	labelComponent      = "app.kubernetes.io/component"
	labelVersion        = "app.kubernetes.io/version"
	labelManagedBy      = "app.kubernetes.io/managed-by"
	labelValueManagedBy = "landscaper-provider"
	labelTopology       = "landscaper.gardener.cloud/topology"
	labelTopologyNs     = "landscaper.gardener.cloud/topology-ns"
)

type Component struct {
	Instance        // for example "test0001-abcdefgh"
	Version  string // for example "v1.0.0"
	Name     string // for example "manifest-deployer"
}

func NewComponent(instance Instance, version, name string) *Component {
	return &Component{
		Instance: Instance(instance),
		Version:  version,
		Name:     name,
	}
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
		labelAppName:     applicationLandscaper,
		labelAppInstance: fmt.Sprintf("%s-%s", applicationLandscaper, c.Instance),
		labelComponent:   c.Name,
	}
}

func (c *Component) InfoLabels() map[string]string {
	return map[string]string{
		labelVersion:   c.Version,
		labelManagedBy: labelValueManagedBy,
	}
}

func (c *Component) TopologyLabels() map[string]string {
	return map[string]string{
		labelTopology:   c.Name,
		labelTopologyNs: c.Namespace(),
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
	return fmt.Sprintf("%s:%s:%s", applicationLandscaper, c.Instance, c.Name)
}

// ClusterScopedResourceName returns the name for a cluster-scoped resource with a given suffix:
// "landscaper:<instance>:<component>:<suffix>", for example "landscaper:test0001-abcdefgh:landscaper-rbac:user".
func (c *Component) ClusterScopedResourceName(suffix string) string {
	return fmt.Sprintf("%s:%s:%s:%s", applicationLandscaper, c.Instance, c.Name, suffix)
}

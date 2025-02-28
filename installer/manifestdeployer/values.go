package manifestdeployer

import (
	"fmt"
	"github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	"github.com/gardener/landscaper/installer/shared"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

type Values struct {
	Instance                    shared.Instance           `json:"instance,omitempty"`
	Version                     string                    `json:"version,omitempty"`
	VerbosityLevel              string                    `json:"verbosityLevel,omitempty"`
	LandscaperClusterKubeconfig *KubeconfigValues         `json:"landscaperClusterKubeconfig,omitempty"`
	Image                       ImageValues               `json:"image,omitempty"`
	ImagePullSecrets            []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	ReplicaCount                *int32                    `json:"replicaCount,omitempty"`
	Resources                   v1.ResourceRequirements   `json:"resources,omitempty"`
	PodSecurityContext          *v1.PodSecurityContext    `json:"podSecurityContext,omitempty"`
	SecurityContext             *v1.SecurityContext       `json:"securityContext,omitempty"`
	ServiceAccount              *ServiceAccountValues     `json:"serviceAccount,omitempty"`
	Configuration               v1alpha2.Configuration    `json:"configuration,omitempty"`
	HostClientSettings          *ClientSettings           `json:"hostClientSettings,omitempty"`
	ResourceClientSettings      *ClientSettings           `json:"resourceClientSettings,omitempty"`
	HPA                         HPAValues                 `json:"hpa,omitempty"`
	NodeSelector                map[string]string         `json:"nodeSelector,omitempty"`
	Affinity                    *v1.Affinity              `json:"affinity,omitempty"`
	Tolerations                 []v1.Toleration           `json:"tolerations,omitempty"`
}

type ReleaseValues struct {
	Instance string `json:"instance,omitempty"`
}

type KubeconfigValues struct {
	Kubeconfig string `json:"kubeconfig,omitempty"`
	SecretRef  string `json:"secretRef,omitempty"`
}

type ClientSettings struct {
	Burst int32 `json:"burst,omitempty"`
	QPS   int32 `json:"qps,omitempty"`
}

type ImageValues struct {
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
	PullPolicy string `json:"pullPolicy,omitempty"`
}

type ServiceAccountValues struct {
	Create bool `json:"create,omitempty"`
}

type HPAValues struct {
	MaxReplicas              int32  `json:"maxReplicas,omitempty"`
	AverageCpuUtilization    *int32 `json:"averageCpuUtilization,omitempty"`
	AverageMemoryUtilization *int32 `json:"averageMemoryUtilization,omitempty"`
}

func (v *Values) Default() {
	if v.VerbosityLevel == "" {
		v.VerbosityLevel = "info"
	}
	if v.Image.PullPolicy == "" {
		v.Image.PullPolicy = "IfNotPresent"
	}
	if v.ReplicaCount == nil {
		v.ReplicaCount = ptr.To(int32(1))
	}
	if v.Configuration.APIVersion == "" {
		v.Configuration.APIVersion = "manifest.deployer.landscaper.gardener.cloud/v1alpha2"
	}
	if v.Configuration.Kind == "" {
		v.Configuration.Kind = "Configuration"
	}
	if v.Configuration.Identity == "" {
		//TODO
		v.Configuration.Identity = fmt.Sprintf("manifest-deployer-%s", v.Instance)
	}
	if v.HostClientSettings == nil {
		v.HostClientSettings = &ClientSettings{}
	}
	if v.HostClientSettings.Burst == 0 {
		v.HostClientSettings.Burst = 30
	}
	if v.HostClientSettings.QPS == 0 {
		v.HostClientSettings.QPS = 20
	}
	if v.ResourceClientSettings == nil {
		v.ResourceClientSettings = &ClientSettings{}
	}
	if v.ResourceClientSettings.Burst == 0 {
		v.ResourceClientSettings.Burst = 60
	}
	if v.ResourceClientSettings.QPS == 0 {
		v.ResourceClientSettings.QPS = 40
	}
	if v.HPA.MaxReplicas == 0 {
		v.HPA.MaxReplicas = 1
	}
	if v.HPA.AverageCpuUtilization == nil {
		v.HPA.AverageCpuUtilization = ptr.To(int32(80))
	}
	if v.HPA.AverageMemoryUtilization == nil {
		v.HPA.AverageMemoryUtilization = ptr.To(int32(80))
	}
}

func (v *Values) Validate() error {
	return nil
}

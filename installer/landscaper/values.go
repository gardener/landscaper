package landscaper

import (
	"github.com/gardener/landscaper/apis/config/v1alpha1"
	"github.com/gardener/landscaper/installer/shared"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

type Values struct {
	Instance           shared.Instance                  `json:"instance,omitempty"`
	Version            string                           `json:"version,omitempty"`
	VerbosityLevel     string                           `json:"verbosityLevel,omitempty"`
	Configuration      v1alpha1.LandscaperConfiguration `json:"configuration,omitempty"`
	ServiceAccount     *ServiceAccountValues            `json:"serviceAccount,omitempty"`
	Controller         ControllerValues                 `json:"controller,omitempty"`
	WebhooksServer     WebhooksServerValues             `json:"webhooksServer,omitempty"`
	ImagePullSecrets   []core.LocalObjectReference      `json:"imagePullSecrets,omitempty"`
	PodSecurityContext *core.PodSecurityContext         `json:"podSecurityContext,omitempty"`
	SecurityContext    *core.SecurityContext            `json:"securityContext,omitempty"`
	NodeSelector       map[string]string                `json:"nodeSelector,omitempty"`
	Affinity           *core.Affinity                   `json:"affinity,omitempty"`
	Tolerations        []core.Toleration                `json:"tolerations,omitempty"`
}

type KubeconfigValues struct {
	Kubeconfig string `json:"kubeconfig,omitempty"`
	SecretRef  string `json:"secretRef,omitempty"`
}

type ServiceAccountValues struct {
	Create bool `json:"create,omitempty"`
}

type ControllerValues struct {
	// LandscaperKubeconfig contains the kubeconfig for the resource cluster (= landscaper cluster).
	LandscaperKubeconfig   *KubeconfigValues         `json:"landscaperKubeconfig,omitempty"`
	Service                *ServiceValues            `json:"service,omitempty"` // optional, has default values
	Image                  ImageValues               `json:"image,omitempty"`
	ReplicaCount           *int32                    `json:"replicaCount,omitempty"`
	Resources              core.ResourceRequirements `json:"resources,omitempty"`
	ResourcesMain          core.ResourceRequirements `json:"resourcesMain,omitempty"`
	Metrics                *MetricsValues            `json:"metrics,omitempty"`
	HostClientSettings     ClientSettings            `json:"hostClientSettings,omitempty"`     // optional, has default value
	ResourceClientSettings ClientSettings            `json:"resourceClientSettings,omitempty"` // optional, has default value
	// HPAMain contains the values for the HPA of the main deployment.
	// (There is no configuration for HPACentral, because its values are fix.)
	HPAMain HPAValues `json:"hpaMain,omitempty"` // optional, has default value
}

const (
	allWebhooks         = "all"
	installationWebhook = "installation"
	executionWebhook    = "execution"
	deployitemWebhook   = "deployitem"
)

type WebhooksServerValues struct {
	DisableWebhooks []string `json:"disableWebhooks,omitempty"`
	// LandscaperKubeconfig contains the kubeconfig for the resource cluster (= landscaper cluster).
	LandscaperKubeconfig  *KubeconfigValues         `json:"landscaperKubeconfig,omitempty"`
	Service               *ServiceValues            `json:"service,omitempty"` // optional, has default value
	Image                 ImageValues               `json:"image,omitempty"`
	ServicePort           int32                     `json:"servicePort,omitempty"` // required unless DisableWebhooks contains "all"
	CertificatesNamespace string                    `json:"certificatesNamespace,omitempty"`
	ReplicaCount          *int32                    `json:"replicaCount,omitempty"` // optional - has default value
	Ingress               *IngressValues            `json:"ingress,omitempty"`      // optional - if nil, no ingress will be created.
	Resources             core.ResourceRequirements `json:"resources,omitempty"`    // optional - has default value
	HPA                   HPAValues                 `json:"hpa,omitempty"`          // optional - has default value
}

type ImageValues struct {
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
	PullPolicy string `json:"pullPolicy,omitempty"`
}

type ServiceValues struct {
	Type string `json:"type,omitempty"`
	Port int32  `json:"port,omitempty"`
}

type IngressValues struct {
	Host      string  `json:"host,omitempty"`
	ClassName *string `json:"className,omitempty"` // optional - if not set, some annotations are omitted.
	DNSClass  string  `json:"dnsClass,omitempty"`
}

type MetricsValues struct {
	Port int32 `json:"port,omitempty"`
}

type ClientSettings struct {
	Burst int32 `json:"burst,omitempty"`
	QPS   int32 `json:"qps,omitempty"`
}

type HPAValues struct {
	MaxReplicas              int32  `json:"maxReplicas,omitempty"`
	AverageCpuUtilization    *int32 `json:"averageCpuUtilization,omitempty"`
	AverageMemoryUtilization *int32 `json:"averageMemoryUtilization,omitempty"`
}

func (v *Values) Default() error {
	if v.Controller.Service == nil {
		v.Controller.Service = &ServiceValues{}
	}
	if v.Controller.Service.Type == "" {
		v.Controller.Service.Type = "ClusterIP"
	}
	if v.Controller.Service.Port == 0 {
		v.Controller.Service.Port = 80
	}

	if v.Controller.HostClientSettings.Burst == 0 {
		v.Controller.HostClientSettings.Burst = 30
	}
	if v.Controller.HostClientSettings.QPS == 0 {
		v.Controller.HostClientSettings.QPS = 20
	}
	if v.Controller.ResourceClientSettings.Burst == 0 {
		v.Controller.ResourceClientSettings.Burst = 60
	}
	if v.Controller.ResourceClientSettings.QPS == 0 {
		v.Controller.ResourceClientSettings.QPS = 40
	}
	if v.Controller.HPAMain.MaxReplicas == 0 {
		v.Controller.HPAMain.MaxReplicas = 1
	}
	if v.Controller.HPAMain.AverageCpuUtilization == nil {
		v.Controller.HPAMain.AverageCpuUtilization = ptr.To(int32(80))
	}
	if v.Controller.HPAMain.AverageMemoryUtilization == nil {
		v.Controller.HPAMain.AverageMemoryUtilization = ptr.To(int32(80))
	}

	if v.WebhooksServer.Service == nil {
		v.WebhooksServer.Service = &ServiceValues{}
	}
	if v.WebhooksServer.Service.Type == "" {
		v.WebhooksServer.Service.Type = "ClusterIP"
	}
	if v.WebhooksServer.Service.Port == 0 {
		v.WebhooksServer.Service.Port = 80
	}
	if v.WebhooksServer.ReplicaCount == nil {
		v.WebhooksServer.ReplicaCount = ptr.To[int32](2)
	}
	if v.WebhooksServer.Resources.Requests == nil {
		cpu, err := resource.ParseQuantity("100m")
		if err != nil {
			return err
		}
		memory, err := resource.ParseQuantity("100Mi")
		if err != nil {
			return err
		}
		v.WebhooksServer.Resources.Requests = core.ResourceList{
			core.ResourceCPU:    cpu,
			core.ResourceMemory: memory,
		}
	}
	if v.WebhooksServer.HPA.MaxReplicas == 0 {
		v.WebhooksServer.HPA.MaxReplicas = 2
	}
	if v.WebhooksServer.HPA.AverageCpuUtilization == nil {
		v.WebhooksServer.HPA.AverageCpuUtilization = ptr.To(int32(80))
	}
	if v.WebhooksServer.HPA.AverageMemoryUtilization == nil {
		v.WebhooksServer.HPA.AverageMemoryUtilization = ptr.To(int32(80))
	}

	return nil
}

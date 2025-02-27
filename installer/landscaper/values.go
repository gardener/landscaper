package landscaper

import (
	"fmt"
	"github.com/gardener/landscaper/apis/config/v1alpha1"
	core "k8s.io/api/core/v1"
)

type Values struct {
	Key                *KeyValues                       `json:"key,omitempty"`
	Version            string                           `json:"version,omitempty"`
	VerbosityLevel     string                           `json:"verbosityLevel,omitempty"`
	Configuration      v1alpha1.LandscaperConfiguration `json:"configuration,omitempty"`
	ServiceAccount     *ServiceAccountValues            `json:"serviceAccount,omitempty"`
	Controller         ControllerValues                 `json:"controller,omitempty"`
	WebhooksServer     *WebhooksServerValues            `json:"webhooksServer,omitempty"`
	ImagePullSecrets   []core.LocalObjectReference      `json:"imagePullSecrets,omitempty"`
	PodSecurityContext *core.PodSecurityContext         `json:"podSecurityContext,omitempty"`
	SecurityContext    *core.SecurityContext            `json:"securityContext,omitempty"`
	NodeSelector       map[string]string                `json:"nodeSelector,omitempty"`
	Affinity           *core.Affinity                   `json:"affinity,omitempty"`
	Tolerations        []core.Toleration                `json:"tolerations,omitempty"`
}

// KeyValues is the key to identify the rbac installation for an update or delete operation.
type KeyValues struct {
	// Name is the name of the landscaper installation, e.g. "landscaper-test0001-abcdefgh".
	Name string `json:"name,omitempty"`

	// HostNamespace is the namespace on the host cluster where the landscaper will be installed.
	HostNamespace string `json:"hostNamespace,omitempty"`
}

func NewKey(name, hostNamespace string) *KeyValues {
	return &KeyValues{
		Name:          name,
		HostNamespace: hostNamespace,
	}
}

func NewDefaultKey() *KeyValues {
	return &KeyValues{
		Name:          "landscaper",
		HostNamespace: "ls-system",
	}
}

func NewKeyFromID(id string) *KeyValues {
	return &KeyValues{
		Name:          fmt.Sprintf("landscaper-%s", id),
		HostNamespace: fmt.Sprintf("ls-system-%s", id),
	}
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
	Service                *ServiceValues            `json:"service,omitempty"`
	Image                  ImageValues               `json:"image,omitempty"`
	ReplicaCount           *int32                    `json:"replicaCount,omitempty"`
	Resources              core.ResourceRequirements `json:"resources,omitempty"`
	ResourcesMain          core.ResourceRequirements `json:"resourcesMain,omitempty"`
	Metrics                *MetricsValues            `json:"metrics,omitempty"`
	HostClientSettings     *ClientSettings           `json:"hostClientSettings,omitempty"`
	ResourceClientSettings *ClientSettings           `json:"resourceClientSettings,omitempty"`
	// HPAMain contains the values for the HPA of the main deployment.
	// (There is no configuration for HPACentral, because its values are fix.)
	HPAMain HPAValues `json:"hpaMain,omitempty"`
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
	Service               ServiceValues             `json:"service,omitempty"`
	Image                 ImageValues               `json:"image,omitempty"`
	ServicePort           int32                     `json:"servicePort,omitempty"` // required unless DisableWebhooks contains "all"
	CertificatesNamespace string                    `json:"certificatesNamespace,omitempty"`
	ReplicaCount          *int32                    `json:"replicaCount,omitempty"`
	Ingress               *IngressValues            `json:"ingress,omitempty"` // optional - if not set, no ingress will be created.
	Resources             core.ResourceRequirements `json:"resources,omitempty"`
	HPA                   HPAValues                 `json:"hpa,omitempty"`
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

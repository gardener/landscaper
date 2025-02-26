package landscaper

import "fmt"

type Values struct {
	Key            *KeyValues            `json:"key,omitempty"`
	Version        string                `json:"version,omitempty"`
	ServiceAccount *ServiceAccountValues `json:"serviceAccount,omitempty"`
	Service        *ServiceValues        `json:"service,omitempty"`
	WebhooksServer *WebhooksServerValues `json:"webhooksServer,omitempty"`

	HPAMain HPAValues `json:"hpaMain,omitempty"`
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

type ServiceAccountValues struct {
	Create bool `json:"create,omitempty"`
}

type DisabledWebhook string

const (
	allWebhooks         DisabledWebhook = "all"
	installationWebhook DisabledWebhook = "installation"
	executionWebhook    DisabledWebhook = "execution"
	deployitemWebhook   DisabledWebhook = "deployitem"
)

type WebhooksServerValues struct {
	DisableWebhooks []DisabledWebhook `json:"disableWebhooks,omitempty"`
	Service         ServiceValues     `json:"service,omitempty"`
	Ingress         *IngressValues    `json:"ingress,omitempty"` // optional - if not set, no ingress will be created.
	HPA             HPAValues         `json:"hpa,omitempty"`
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

type HPAValues struct {
	MaxReplicas              int32  `json:"maxReplicas,omitempty"`
	AverageCpuUtilization    *int32 `json:"averageCpuUtilization,omitempty"`
	AverageMemoryUtilization *int32 `json:"averageMemoryUtilization,omitempty"`
}

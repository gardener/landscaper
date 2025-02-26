package landscaper

import "fmt"

type Values struct {
	Key            *KeyValues            `json:"key,omitempty"`
	Version        string                `json:"version,omitempty"`
	ServiceAccount *ServiceAccountValues `json:"serviceAccount,omitempty"`
	HPAMain        HPAValues             `json:"hpaMain,omitempty"`
	HPAWebhooks    HPAValues             `json:"hpaWebhooks,omitempty"`
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

type HPAValues struct {
	MaxReplicas              int32  `json:"maxReplicas,omitempty"`
	AverageCpuUtilization    *int32 `json:"averageCpuUtilization,omitempty"`
	AverageMemoryUtilization *int32 `json:"averageMemoryUtilization,omitempty"`
}

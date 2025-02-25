package landscaper

type Values struct {
	Key            *KeyValues            `json:"key,omitempty"`
	Version        string                `json:"version,omitempty"`
	ServiceAccount *ServiceAccountValues `json:"serviceAccount,omitempty"`
	HPAMain        HPAValues             `json:"hpaMain,omitempty"`
	HPAWebhooks    HPAValues             `json:"hpaWebhooks,omitempty"`
}

// KeyValues is the key to identify the rbac installation for an update or delete operation.
type KeyValues struct {
	// Name is the name of the rbac installation.
	Name string `json:"name,omitempty"`

	// HostNamespace is the namespace on the host cluster where the landscaper will be installed.
	HostNamespace string `json:"hostNamespace,omitempty"`
}

type ServiceAccountValues struct {
	Create bool `json:"create,omitempty"`
}

type HPAValues struct {
	MaxReplicas              int32  `json:"maxReplicas,omitempty"`
	AverageCpuUtilization    *int32 `json:"averageCpuUtilization,omitempty"`
	AverageMemoryUtilization *int32 `json:"averageMemoryUtilization,omitempty"`
}

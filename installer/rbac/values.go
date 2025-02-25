package rbac

type Values struct {
	Key     *KeyValues `json:"key,omitempty"`
	Version string     `json:"version,omitempty"`

	ServiceAccount *ServiceAccountValues `json:"serviceAccount,omitempty"`
}

// KeyValues is the key to identify the rbac installation for an update or delete operation.
type KeyValues struct {
	// Name is the name of the rbac installation.
	Name string `json:"name,omitempty"`

	// ResourceNamespace is the namespace on the resource cluster where the rbac objects will be installed.
	ResourceNamespace string `json:"resourceNamespace,omitempty"`
}

type ServiceAccountValues struct {
	Create bool `json:"create,omitempty"`
}

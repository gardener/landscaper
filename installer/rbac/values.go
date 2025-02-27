package rbac

import "fmt"

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

func NewKey(instance, resourceNamespace string) *KeyValues {
	return &KeyValues{
		Name:              instance,
		ResourceNamespace: resourceNamespace,
	}
}

func NewDefaultKey() *KeyValues {
	return &KeyValues{
		Name:              "landscaper-rbac",
		ResourceNamespace: "ls-system",
	}
}

func NewKeyFromID(id string) *KeyValues {
	return &KeyValues{
		Name:              fmt.Sprintf("rbac-%s", id),
		ResourceNamespace: fmt.Sprintf("ls-system-%s", id),
	}
}

type ServiceAccountValues struct {
	Create bool `json:"create,omitempty"`
}

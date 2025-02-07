package rbac

import (
	"github.com/gardener/landscaper/installer/shared"
)

type Values struct {
	Instance       shared.Instance       `json:"instance,omitempty"`
	Version        string                `json:"version,omitempty"`
	ServiceAccount *ServiceAccountValues `json:"serviceAccount,omitempty"`
}

type ServiceAccountValues struct {
	Create bool `json:"create,omitempty"`
}

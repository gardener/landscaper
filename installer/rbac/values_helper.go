package rbac

import (
	"fmt"
	"github.com/gardener/landscaper/installer/shared"
)

type valuesHelper struct {
	values        *Values
	rbacComponent *shared.Component
}

func newValuesHelper(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}

	return &valuesHelper{
		values: values,
		rbacComponent: &shared.Component{
			Instance: values.Instance,
			Version:  values.Version,
			Name:     "landscaper-rbac",
		},
	}, nil
}

func (h *valuesHelper) resourceNamespace() string {
	return h.values.Instance.Namespace()
}

func (h *valuesHelper) isCreateServiceAccount() bool {
	return h.values.ServiceAccount != nil && h.values.ServiceAccount.Create
}

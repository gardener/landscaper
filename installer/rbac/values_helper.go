package rbac

import (
	"fmt"
	"maps"
)

type valuesHelper struct {
	values *Values
}

func newValuesHelper(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}

	return &valuesHelper{
		values: values,
	}, nil
}

func (h *valuesHelper) resourceNamespace() string {
	return h.values.Key.ResourceNamespace
}

func (h *valuesHelper) landscaperLabels() map[string]string {
	labels := map[string]string{
		"app.kubernetes.io/version":    h.values.Version,
		"app.kubernetes.io/managed-by": "landscaper-installer",
	}
	maps.Copy(labels, h.selectorLabels())
	return labels
}

func (h *valuesHelper) selectorLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     "landscaper-rbac",
		"app.kubernetes.io/instance": h.values.Key.Name,
	}
}

func (h *valuesHelper) isCreateServiceAccount() bool {
	return h.values.ServiceAccount != nil && h.values.ServiceAccount.Create
}

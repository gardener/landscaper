package landscaper

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

func (h *valuesHelper) hostNamespace() string {
	return h.values.Key.HostNamespace
}

func (h *valuesHelper) landscaperFullName() string {
	//TODO also called landscaperCentralFullName()
	return fmt.Sprintf("%s-%s", h.values.Key.Name, h.values.Key.HostNamespace)
}

func (h *valuesHelper) landscaperMainFullName() string {
	//TODO
	return fmt.Sprintf("%s-%s", h.values.Key.Name, h.values.Key.HostNamespace)
}

func (h *valuesHelper) landscaperWebhooksFullName() string {
	//TODO
	return fmt.Sprintf("%s-%s", h.values.Key.Name, h.values.Key.HostNamespace)
}

func (h *valuesHelper) landscaperLabels() map[string]string {
	//TODO
	labels := map[string]string{
		"app.kubernetes.io/version":    h.values.Version,
		"app.kubernetes.io/managed-by": "landscaper-installer",
	}
	maps.Copy(labels, h.selectorLabels())
	return labels
}

func (h *valuesHelper) selectorLabels() map[string]string {
	//TODO
	return map[string]string{
		"app.kubernetes.io/name":     "landscaper-rbac",
		"app.kubernetes.io/instance": h.values.Key.Name,
	}
}

func (h *valuesHelper) isCreateServiceAccount() bool {
	//TODO
	return h.values.ServiceAccount != nil && h.values.ServiceAccount.Create
}

func (h *valuesHelper) landscaperControllerServiceAccountName() string {
	//TODO
	return "landscaper"
}

func (h *valuesHelper) landscaperAgentFullName() string {
	//TODO
	return "landscaper"
}

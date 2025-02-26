package landscaper

import (
	"fmt"
	"maps"
	"slices"
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
	return h.values.Key.Name
}

func (h *valuesHelper) landscaperMainFullName() string {
	return fmt.Sprintf("%s-main", h.values.Key.Name)
}

func (h *valuesHelper) landscaperWebhooksFullName() string {
	return fmt.Sprintf("%s-webhooks", h.values.Key.Name)
}

func (h *valuesHelper) clusterRoleName() string {
	return fmt.Sprintf("landscaper:%s-agent", h.values.Key.Name)
}

func (h *valuesHelper) landscaperLabels() map[string]string {
	labels := map[string]string{
		"app.kubernetes.io/version":    h.values.Version,
		"app.kubernetes.io/managed-by": "landscaper-installer",
	}
	maps.Copy(labels, h.selectorLabels())
	return labels
}

// TODO labels for component and topology
func (h *valuesHelper) selectorLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     "landscaper",
		"app.kubernetes.io/instance": h.values.Key.Name,
	}
}

// TODO labels for component and topology
func (h *valuesHelper) mainSelectorLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     "landscaper",
		"app.kubernetes.io/instance": h.values.Key.Name,
	}
}

// TODO labels for component and topology
func (h *valuesHelper) webhooksSelectorLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     "landscaper",
		"app.kubernetes.io/instance": h.values.Key.Name,
	}
}

func (h *valuesHelper) configSecretName() string {
	return fmt.Sprintf("%s-config", h.landscaperFullName())
}

func (h *valuesHelper) controllerKubeconfigSecretName() string {
	return fmt.Sprintf("%s-controller-cluster-kubeconfig", h.landscaperFullName())
}

func (h *valuesHelper) isCreateServiceAccount() bool {
	return h.values.ServiceAccount != nil && h.values.ServiceAccount.Create
}

func (h *valuesHelper) areAllWebhooksDisabled() bool {
	return slices.Contains(h.values.WebhooksServer.DisableWebhooks, allWebhooks)
}

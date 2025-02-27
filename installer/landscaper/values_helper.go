package landscaper

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/gardener/landscaper/apis/config/v1alpha1"
	"maps"
	"sigs.k8s.io/yaml"
	"slices"
)

type valuesHelper struct {
	values *Values

	config     v1alpha1.LandscaperConfiguration
	configYaml []byte
	configHash string
}

func newValuesHelper(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}

	// compute values
	config := values.Configuration
	configYaml, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal landscaper config: %w", err)
	}
	hash := sha256.Sum256(configYaml)
	configHash := hex.EncodeToString(hash[:])

	return &valuesHelper{
		values:     values,
		config:     config,
		configYaml: configYaml,
		configHash: configHash,
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

func (h *valuesHelper) podAnnotations() map[string]string {
	return make(map[string]string)
}

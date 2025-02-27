package helmdeployer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/exp/maps"
	"sigs.k8s.io/yaml"
)

type valuesHelper struct {
	values *Values

	configYaml          []byte
	configHash          string
	registrySecretsYaml []byte
	registrySecretsHash string
	registrySecretsData map[string][]byte
}

func newValuesHelper(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}
	values.Default()
	if err := values.Validate(); err != nil {
		return nil, fmt.Errorf("invalid helm deployer values: %w", err)
	}

	// compute values
	configYaml, err := yaml.Marshal(values.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal helm deployer config: %w", err)
	}
	configHash := sha256.Sum256(configYaml)

	registrySecretsYaml, err := yaml.Marshal(values.OCI)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal helm deployer config: %w", err)
	}
	registrySecretsHash := sha256.Sum256(registrySecretsYaml)

	registrySecretsData := make(map[string][]byte)
	if values.OCI != nil {
		for key, valueObj := range values.OCI.Secrets {
			valueBytes, err := json.Marshal(valueObj)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal value for registries secret: %w", err)
			}
			registrySecretsData[key] = valueBytes
		}
	}

	return &valuesHelper{
		values:              values,
		configYaml:          configYaml,
		configHash:          hex.EncodeToString(configHash[:]),
		registrySecretsYaml: registrySecretsYaml,
		registrySecretsHash: hex.EncodeToString(registrySecretsHash[:]),
		registrySecretsData: registrySecretsData,
	}, nil
}

func newValuesHelperForDelete(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}
	values.Default()
	if err := values.Validate(); err != nil {
		return nil, fmt.Errorf("invalid helm deployer values: %w", err)
	}

	return &valuesHelper{
		values: values,
	}, nil
}

func (h *valuesHelper) hostNamespace() string {
	return h.values.Key.HostNamespace
}

func (h *valuesHelper) deployerFullName() string {
	return h.values.Key.Name
}

func (h *valuesHelper) clusterRoleName() string {
	return fmt.Sprintf("landscaper:%s", h.values.Key.Name)
}

func (h *valuesHelper) deployerLabels() map[string]string {
	labels := map[string]string{
		"app.kubernetes.io/version":    h.values.Version,
		"app.kubernetes.io/managed-by": "landscaper-installer",
	}
	maps.Copy(labels, h.selectorLabels())
	return labels
}

func (h *valuesHelper) selectorLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     "helm-deployer",
		"app.kubernetes.io/instance": h.values.Key.Name,
	}
}

func (h *valuesHelper) identity() string {
	return fmt.Sprintf("helm-%s", h.hostNamespace())
}

func (h *valuesHelper) deployerConfig() ([]byte, error) {
	configYaml, err := yaml.Marshal(h.values.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal helm deployer config: %w", err)
	}
	return configYaml, nil
}

func (h *valuesHelper) landscaperClusterKubeconfig() []byte {
	return []byte(h.values.LandscaperClusterKubeconfig.Kubeconfig)
}

func (h *valuesHelper) isCreateServiceAccount() bool {
	return h.values.ServiceAccount != nil && h.values.ServiceAccount.Create
}

func (h *valuesHelper) podAnnotations() map[string]string {
	return make(map[string]string)
}

func (h *valuesHelper) ociSecrets() map[string]any {
	return nil // TODO
}

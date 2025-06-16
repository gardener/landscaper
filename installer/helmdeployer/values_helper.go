package helmdeployer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gardener/landscaper/installer/shared"
	"sigs.k8s.io/yaml"
)

const (
	componentHelmDeployer = "helm-deployer"
)

type valuesHelper struct {
	values *Values

	helmDeployerComponent *shared.Component

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
	if err := values.Default(); err != nil {
		return nil, fmt.Errorf("failed to apply default helm deployer values: %w", err)
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
		values:                values,
		helmDeployerComponent: shared.NewComponent(values.Instance, values.Version, componentHelmDeployer),
		configYaml:            configYaml,
		configHash:            hex.EncodeToString(configHash[:]),
		registrySecretsYaml:   registrySecretsYaml,
		registrySecretsHash:   hex.EncodeToString(registrySecretsHash[:]),
		registrySecretsData:   registrySecretsData,
	}, nil
}

func newValuesHelperForDelete(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}
	if err := values.Default(); err != nil {
		return nil, fmt.Errorf("failed to apply default helm deployer values during delete operation: %w", err)
	}

	return &valuesHelper{
		values:                values,
		helmDeployerComponent: shared.NewComponent(values.Instance, values.Version, componentHelmDeployer),
	}, nil
}

func (h *valuesHelper) hostNamespace() string {
	return h.values.Instance.Namespace()
}

func (h *valuesHelper) deployerFullName() string {
	return h.helmDeployerComponent.ComponentAndInstance()
}

func (h *valuesHelper) clusterRoleName() string {
	return h.helmDeployerComponent.ClusterScopedDefaultResourceName()
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

func (h *valuesHelper) ociSecrets() map[string]any {
	return nil // TODO
}

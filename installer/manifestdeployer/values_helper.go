package manifestdeployer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/gardener/landscaper/installer/shared"
	"sigs.k8s.io/yaml"
)

const (
	componentManifestDeployer = "manifest-deployer"
)

type valuesHelper struct {
	values *Values

	manifestDeployerComponent *shared.Component

	configYaml []byte
	configHash string
}

func newValuesHelper(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}
	if err := values.Default(); err != nil {
		return nil, fmt.Errorf("failed to apply default manifest deployer values: %w", err)
	}

	// compute values
	configYaml, err := yaml.Marshal(values.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest deployer config: %w", err)
	}
	hash := sha256.Sum256(configYaml)
	configHash := hex.EncodeToString(hash[:])

	return &valuesHelper{
		values:                    values,
		manifestDeployerComponent: shared.NewComponent(values.Instance, values.Version, componentManifestDeployer),
		configYaml:                configYaml,
		configHash:                configHash,
	}, nil
}

func newValuesHelperForDelete(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}
	if err := values.Default(); err != nil {
		return nil, fmt.Errorf("failed to apply default manifest deployer values during delete operation: %w", err)
	}

	return &valuesHelper{
		values:                    values,
		manifestDeployerComponent: shared.NewComponent(values.Instance, values.Version, componentManifestDeployer),
	}, nil
}

func (h *valuesHelper) hostNamespace() string {
	return h.values.Instance.Namespace()
}

func (h *valuesHelper) deployerFullName() string {
	return h.manifestDeployerComponent.ComponentAndInstance()
}

func (h *valuesHelper) clusterRoleName() string {
	return h.manifestDeployerComponent.ClusterScopedDefaultResourceName()
}

func (h *valuesHelper) deployerConfig() ([]byte, error) {
	configYaml, err := yaml.Marshal(h.values.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest deployer config: %w", err)
	}
	return configYaml, nil
}

func (h *valuesHelper) landscaperClusterKubeconfig() []byte {
	return []byte(h.values.LandscaperClusterKubeconfig.Kubeconfig)
}

func (h *valuesHelper) isCreateServiceAccount() bool {
	return h.values.ServiceAccount != nil && h.values.ServiceAccount.Create
}

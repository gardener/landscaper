package manifestdeployer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/gardener/landscaper/installer/shared"
	"golang.org/x/exp/maps"
	"sigs.k8s.io/yaml"
)

const (
	appNameManifestDeployer = "manifest-deployer"
)

type valuesHelper struct {
	values *Values

	configYaml []byte
	configHash string
}

func newValuesHelper(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}
	values.Default()
	if err := values.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest deployer values: %w", err)
	}

	// compute values
	configYaml, err := yaml.Marshal(values.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest deployer config: %w", err)
	}
	hash := sha256.Sum256(configYaml)
	configHash := hex.EncodeToString(hash[:])

	return &valuesHelper{
		values:     values,
		configYaml: configYaml,
		configHash: configHash,
	}, nil
}

func newValuesHelperForDelete(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}
	values.Default()
	if err := values.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest deployer values: %w", err)
	}

	return &valuesHelper{
		values: values,
	}, nil
}

func (h *valuesHelper) appAndInstance() string {
	return fmt.Sprintf("%s-%s", appNameManifestDeployer, h.values.Instance)
}

func (h *valuesHelper) hostNamespace() string {
	return h.values.Instance.Namespace()
}

func (h *valuesHelper) deployerFullName() string {
	return h.appAndInstance()
}

func (h *valuesHelper) clusterRoleName() string {
	return h.values.Instance.ClusterScopedResourceName(appNameManifestDeployer)
}

func (h *valuesHelper) deployerLabels() map[string]string {
	labels := map[string]string{
		shared.LabelVersion:   h.values.Version,
		shared.LabelManagedBy: shared.LabelValueManagedBy,
	}
	maps.Copy(labels, h.selectorLabels())
	return labels
}

func (h *valuesHelper) selectorLabels() map[string]string {
	return map[string]string{
		shared.LabelAppName:     appNameManifestDeployer,
		shared.LabelAppInstance: h.appAndInstance(),
	}
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

func (h *valuesHelper) podAnnotations() map[string]string {
	return make(map[string]string)
}

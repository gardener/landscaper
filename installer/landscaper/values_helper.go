package landscaper

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/gardener/landscaper/apis/config/v1alpha1"
	"github.com/gardener/landscaper/installer/shared"
	"sigs.k8s.io/yaml"
	"slices"
)

const (
	componentControllerMain = "landscaper-controller-main"
	componentController     = "landscaper-controller"
	componentWebhooks       = "landscaper-webhooks-server"
)

type valuesHelper struct {
	values *Values

	controllerMainComponent *shared.Component
	controllerComponent     *shared.Component
	webhooksComponent       *shared.Component

	config     v1alpha1.LandscaperConfiguration
	configYaml []byte
	configHash string
}

func newValuesHelper(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}
	values.Default()

	// compute values
	config := values.Configuration
	configYaml, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal landscaper config: %w", err)
	}
	hash := sha256.Sum256(configYaml)
	configHash := hex.EncodeToString(hash[:])

	return &valuesHelper{
		values:                  values,
		controllerMainComponent: shared.NewComponent(values.Instance, values.Version, componentControllerMain),
		controllerComponent:     shared.NewComponent(values.Instance, values.Version, componentController),
		webhooksComponent:       shared.NewComponent(values.Instance, values.Version, componentWebhooks),
		config:                  config,
		configYaml:              configYaml,
		configHash:              configHash,
	}, nil
}

func newValuesHelperForDelete(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}

	return &valuesHelper{
		values:                  values,
		controllerMainComponent: shared.NewComponent(values.Instance, values.Version, componentControllerMain),
		controllerComponent:     shared.NewComponent(values.Instance, values.Version, componentController),
		webhooksComponent:       shared.NewComponent(values.Instance, values.Version, componentWebhooks),
	}, nil
}

func (h *valuesHelper) hostNamespace() string {
	return h.values.Instance.Namespace()
}

func (h *valuesHelper) resourceNamespace() string {
	return h.values.Instance.Namespace()
}

func (h *valuesHelper) landscaperFullName() string {
	return h.controllerComponent.ComponentAndInstance()
}

func (h *valuesHelper) landscaperMainFullName() string {
	return h.controllerMainComponent.ComponentAndInstance()
}

func (h *valuesHelper) landscaperWebhooksFullName() string {
	return h.webhooksComponent.ComponentAndInstance()
}

func (h *valuesHelper) configSecretName() string {
	return fmt.Sprintf("%s-config", h.landscaperFullName())
}

func (h *valuesHelper) controllerKubeconfigSecretName() string {
	return fmt.Sprintf("%s-controller-cluster-kubeconfig", h.landscaperFullName())
}

func (h *valuesHelper) webhooksKubeconfigSecretName() string {
	return fmt.Sprintf("%s-webhooks-cluster-kubeconfig", h.landscaperFullName())
}

func (h *valuesHelper) isCreateServiceAccount() bool {
	return h.values.ServiceAccount != nil && h.values.ServiceAccount.Create
}

func (h *valuesHelper) areAllWebhooksDisabled() bool {
	return slices.Contains(h.values.WebhooksServer.DisableWebhooks, allWebhooks)
}

package landscaper

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/gardener/landscaper/apis/config/v1alpha1"
	"github.com/gardener/landscaper/installer/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
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

	config     *v1alpha1.LandscaperConfiguration
	configYaml []byte
	configHash string
}

func newValuesHelper(values *Values) (*valuesHelper, error) {
	if values == nil {
		return nil, fmt.Errorf("values must not be nil")
	}
	values.Default()

	h := &valuesHelper{
		values:                  values,
		controllerMainComponent: shared.NewComponent(values.Instance, values.Version, componentControllerMain),
		controllerComponent:     shared.NewComponent(values.Instance, values.Version, componentController),
		webhooksComponent:       shared.NewComponent(values.Instance, values.Version, componentWebhooks),
	}

	if err := h.computeConfiguration(); err != nil {
		return nil, err
	}

	return h, nil
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

func (h *valuesHelper) computeConfiguration() (err error) {
	h.config = &v1alpha1.LandscaperConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "config.landscaper.gardener.cloud/v1alpha1",
			Kind:       "LandscaperConfiguration",
		},
		Controllers: v1alpha1.Controllers{
			Installations: h.values.Controller.Installations,
			Executions:    h.values.Controller.Executions,
			DeployItems:   h.values.Controller.DeployItems,
			Contexts:      h.values.Controller.Contexts,
		},
		Registry: v1alpha1.RegistryConfiguration{
			OCI: &v1alpha1.OCIConfiguration{
				Cache: &v1alpha1.OCICacheConfiguration{
					UseInMemoryOverlay: false,
					Path:               "/app/ls/oci-cache/",
				},
				AllowPlainHttp:     false,
				InsecureSkipVerify: false,
			},
		},
		CrdManagement: v1alpha1.CrdManagementConfiguration{
			DeployCustomResourceDefinitions: ptr.To(true),
			ForceUpdate:                     ptr.To(true),
		},
		DeployItemTimeouts: h.values.Controller.DeployItemTimeouts,
		LsDeployments: &v1alpha1.LsDeployments{
			LsController:          h.landscaperFullName(),
			LsMainController:      h.landscaperMainFullName(),
			WebHook:               h.landscaperWebhooksFullName(),
			DeploymentsNamespace:  h.hostNamespace(),
			LsHealthCheckName:     h.landscaperFullName(),
			AdditionalDeployments: h.values.Controller.HealthChecks,
		},
		HPAMainConfiguration: &v1alpha1.HPAMainConfiguration{
			MaxReplicas: h.values.Controller.HPAMain.MaxReplicas,
		},
	}

	h.configYaml, err = yaml.Marshal(h.config)
	if err != nil {
		return fmt.Errorf("failed to marshal landscaper configuration: %w", err)
	}

	hash := sha256.Sum256(h.configYaml)
	h.configHash = hex.EncodeToString(hash[:])

	return nil
}

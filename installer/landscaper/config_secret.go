package landscaper

import (
	"fmt"
	"github.com/gardener/landscaper/installer/resources"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type configSecretMutator struct {
	*valuesHelper
}

var _ resources.Mutator[*v1.Secret] = &configSecretMutator{}

func newConfigSecretMutator(b *valuesHelper) resources.Mutator[*v1.Secret] {
	return &configSecretMutator{valuesHelper: b}
}

func (m *configSecretMutator) String() string {
	return fmt.Sprintf("secret %s/%s", m.hostNamespace(), m.configSecretName())
}

func (m *configSecretMutator) Empty() *v1.Secret {
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.configSecretName(),
			Namespace: m.hostNamespace(),
		},
	}
}

func (m *configSecretMutator) Mutate(r *v1.Secret) error {
	r.ObjectMeta.Labels = m.controllerComponent.Labels()
	r.Data = map[string][]byte{
		"config.yaml": m.valuesHelper.configYaml,
	}
	return nil
}

//func (m *configSecretMutator) config() ([]byte, error) {
//	conf := v1alpha1.LandscaperConfiguration{
//		TypeMeta: metav1.TypeMeta{
//			APIVersion: "config.landscaper.gardener.cloud/v1alpha1",
//			Kind:       "LandscaperConfiguration",
//		},
//		Controllers:                            v1alpha1.Controllers{},
//		RepositoryContext:                      nil,
//		Registry:                               v1alpha1.RegistryConfiguration{},
//		BlueprintStore:                         v1alpha1.BlueprintStore{},
//		Metrics:                                nil,
//		CrdManagement:                          v1alpha1.CrdManagementConfiguration{},
//		DeployItemTimeouts:                     nil,
//		LsDeployments:                          nil,
//		HPAMainConfiguration:                   nil,
//		SignatureVerificationEnforcementPolicy: "",
//	}
//
//	confBytes, err := yaml.Marshal(conf)
//	if err != nil {
//		return nil, fmt.Errorf("failed to marshal landscaper configuration: %w", err)
//	}
//
//	return confBytes, nil
//}

package manifestdeployer

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type configSecretDefinition struct {
	*valuesHelper
}

var _ ResourceDefinition[*v1.Secret] = &configSecretDefinition{}

func newConfigSecretDefinition(b *valuesHelper) ResourceDefinition[*v1.Secret] {
	return &configSecretDefinition{valuesHelper: b}
}

func (d *configSecretDefinition) String() string {
	return fmt.Sprintf("config secret %s/%s", d.hostNamespace(), d.name())
}

func (d *configSecretDefinition) name() string {
	return fmt.Sprintf("%s-config", d.deployerFullName())
}

func (d *configSecretDefinition) Empty() *v1.Secret {
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.name(),
			Namespace: d.hostNamespace(),
		},
	}
}

func (d *configSecretDefinition) Mutate(r *v1.Secret) error {
	r.ObjectMeta.Labels = d.deployerLabels()
	r.Data = map[string][]byte{
		"config.yaml": d.configYaml,
	}
	return nil
}

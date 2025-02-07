package helmdeployer

import (
	"fmt"
	"github.com/gardener/landscaper/installer/resources"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type registrySecretMutator struct {
	*valuesHelper
}

var _ resources.Mutator[*v1.Secret] = &registrySecretMutator{}

func newRegistrySecretMutator(b *valuesHelper) resources.Mutator[*v1.Secret] {
	return &registrySecretMutator{valuesHelper: b}
}

func (d *registrySecretMutator) String() string {
	return fmt.Sprintf("registry secret %s/%s", d.hostNamespace(), d.name())
}

func (d *registrySecretMutator) name() string {
	return fmt.Sprintf("%s-registries", d.deployerFullName())
}

func (d *registrySecretMutator) Empty() *v1.Secret {
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

func (d *registrySecretMutator) Mutate(r *v1.Secret) error {
	r.ObjectMeta.Labels = d.helmDeployerComponent.Labels()
	r.Data = d.registrySecretsData
	return nil
}

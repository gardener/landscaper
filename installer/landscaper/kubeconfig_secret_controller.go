package landscaper

import (
	"fmt"
	"github.com/gardener/landscaper/installer/resources"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type controllerKubeconfigSecretMutator struct {
	*valuesHelper
}

var _ resources.Mutator[*v1.Secret] = &controllerKubeconfigSecretMutator{}

func newControllerKubeconfigSecretMutator(b *valuesHelper) resources.Mutator[*v1.Secret] {
	return &controllerKubeconfigSecretMutator{valuesHelper: b}
}

func (d *controllerKubeconfigSecretMutator) String() string {
	return fmt.Sprintf("secret %s/%s", d.hostNamespace(), d.controllerKubeconfigSecretName())
}

func (d *controllerKubeconfigSecretMutator) Empty() *v1.Secret {
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.controllerKubeconfigSecretName(),
			Namespace: d.hostNamespace(),
		},
	}
}

func (d *controllerKubeconfigSecretMutator) Mutate(r *v1.Secret) error {
	r.ObjectMeta.Labels = d.controllerComponent.Labels()
	r.Data = map[string][]byte{
		"kubeconfig": []byte(d.values.Controller.LandscaperKubeconfig.Kubeconfig),
	}
	return nil
}

package manifestdeployer

import (
	"fmt"
	"github.com/gardener/landscaper/installer/resources"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type kubeconfigSecretMutator struct {
	*valuesHelper
}

var _ resources.Mutator[*v1.Secret] = &kubeconfigSecretMutator{}

func newKubeconfigSecretMutator(b *valuesHelper) resources.Mutator[*v1.Secret] {
	return &kubeconfigSecretMutator{valuesHelper: b}
}

func (d *kubeconfigSecretMutator) String() string {
	return fmt.Sprintf("kubeconfig secret %s/%s", d.hostNamespace(), d.name())
}

func (d *kubeconfigSecretMutator) name() string {
	return fmt.Sprintf("%s-landscaper-cluster-kubeconfig", d.deployerFullName())
}

func (d *kubeconfigSecretMutator) Empty() *v1.Secret {
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

func (d *kubeconfigSecretMutator) Mutate(r *v1.Secret) error {
	r.ObjectMeta.Labels = d.manifestDeployerComponent.Labels()
	r.Data = map[string][]byte{
		"kubeconfig": d.landscaperClusterKubeconfig(),
	}
	return nil
}

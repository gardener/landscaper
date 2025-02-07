package manifestdeployer

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type landscaperClusterKubeconfigSecretDefinition struct {
	*valuesHelper
}

var _ ResourceDefinition[*v1.Secret] = &landscaperClusterKubeconfigSecretDefinition{}

func newLandscaperClusterKubeconfigSecretDefinition(b *valuesHelper) ResourceDefinition[*v1.Secret] {
	return &landscaperClusterKubeconfigSecretDefinition{valuesHelper: b}
}

func (d *landscaperClusterKubeconfigSecretDefinition) String() string {
	return fmt.Sprintf("landscaper cluster kubeconfig secret %s/%s", d.hostNamespace(), d.name())
}

func (d *landscaperClusterKubeconfigSecretDefinition) name() string {
	return fmt.Sprintf("%s-landscaper-cluster-kubeconfig", d.deployerFullName())
}

func (d *landscaperClusterKubeconfigSecretDefinition) Empty() *v1.Secret {
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

func (d *landscaperClusterKubeconfigSecretDefinition) Mutate(r *v1.Secret) error {
	r.ObjectMeta.Labels = d.deployerLabels()
	r.Data = map[string][]byte{
		"kubeconfig": d.landscaperClusterKubeconfig(),
	}
	return nil
}

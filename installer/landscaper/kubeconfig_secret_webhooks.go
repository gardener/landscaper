package landscaper

import (
	"fmt"
	"github.com/gardener/landscaper/installer/resources"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type webhooksKubeconfigSecretMutator struct {
	*valuesHelper
}

var _ resources.Mutator[*v1.Secret] = &webhooksKubeconfigSecretMutator{}

func newWebhooksKubeconfigSecretMutator(b *valuesHelper) resources.Mutator[*v1.Secret] {
	return &webhooksKubeconfigSecretMutator{valuesHelper: b}
}

func (d *webhooksKubeconfigSecretMutator) String() string {
	return fmt.Sprintf("secret %s/%s", d.hostNamespace(), d.webhooksKubeconfigSecretName())
}

func (d *webhooksKubeconfigSecretMutator) Empty() *v1.Secret {
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.webhooksKubeconfigSecretName(),
			Namespace: d.hostNamespace(),
		},
	}
}

func (d *webhooksKubeconfigSecretMutator) Mutate(r *v1.Secret) error {
	r.ObjectMeta.Labels = d.webhooksComponent.Labels()
	r.Data = map[string][]byte{
		"kubeconfig": []byte(d.values.WebhooksServer.LandscaperKubeconfig.Kubeconfig),
	}
	return nil
}

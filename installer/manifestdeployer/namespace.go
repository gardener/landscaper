package manifestdeployer

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type namespaceDefinition struct {
	*valuesHelper
}

var _ ResourceDefinition[*v1.Namespace] = &namespaceDefinition{}

func newNamespaceDefinition(b *valuesHelper) ResourceDefinition[*v1.Namespace] {
	return &namespaceDefinition{valuesHelper: b}
}

func (d *namespaceDefinition) String() string {
	return fmt.Sprintf("namespace %s", d.hostNamespace())
}

func (d *namespaceDefinition) Empty() *v1.Namespace {
	return &v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: d.valuesHelper.hostNamespace(),
		},
	}
}

func (d *namespaceDefinition) Mutate(r *v1.Namespace) error {
	return nil
}

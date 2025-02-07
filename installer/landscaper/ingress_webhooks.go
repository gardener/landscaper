package landscaper

import (
	"fmt"
	"github.com/gardener/landscaper/installer/resources"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

type ingressMutator struct {
	*valuesHelper
}

var _ resources.Mutator[*networking.Ingress] = &ingressMutator{}

func newIngressMutator(b *valuesHelper) resources.Mutator[*networking.Ingress] {
	return &ingressMutator{valuesHelper: b}
}

func (m *ingressMutator) String() string {
	return fmt.Sprintf("landscaper webhooks ingress %s/%s", m.hostNamespace(), m.landscaperWebhooksFullName())
}

func (m *ingressMutator) Empty() *networking.Ingress {
	return &networking.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.landscaperWebhooksFullName(),
			Namespace: m.hostNamespace(),
		},
	}
}

func (m *ingressMutator) Mutate(r *networking.Ingress) error {
	r.ObjectMeta.Labels = m.webhooksComponent.Labels()
	r.ObjectMeta.Annotations = map[string]string{
		"nginx.ingress.kubernetes.io/ssl-passthrough": "true",
	}
	if m.values.WebhooksServer.Ingress.DNSClass != "" {
		r.ObjectMeta.Annotations["dns.gardener.cloud/class"] = m.values.WebhooksServer.Ingress.DNSClass
		r.ObjectMeta.Annotations["dns.gardener.cloud/dnsnames"] = m.values.WebhooksServer.Ingress.Host
	}
	r.Spec = networking.IngressSpec{
		IngressClassName: m.values.WebhooksServer.Ingress.ClassName,
		Rules: []networking.IngressRule{
			{
				Host: m.values.WebhooksServer.Ingress.Host,
				IngressRuleValue: networking.IngressRuleValue{
					HTTP: &networking.HTTPIngressRuleValue{
						Paths: []networking.HTTPIngressPath{
							{
								Path:     "/",
								PathType: ptr.To(networking.PathTypePrefix),
								Backend: networking.IngressBackend{
									Service: &networking.IngressServiceBackend{
										Name: m.landscaperWebhooksFullName(),
										Port: networking.ServiceBackendPort{
											Number: m.values.WebhooksServer.Service.Port,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return nil
}

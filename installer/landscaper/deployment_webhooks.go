package landscaper

import (
	"fmt"
	"github.com/gardener/landscaper/installer/rbac"
	"github.com/gardener/landscaper/installer/resources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"strings"
)

type webhooksDeploymentMutator struct {
	*valuesHelper
}

var _ resources.Mutator[*appsv1.Deployment] = &webhooksDeploymentMutator{}

func newWebhooksDeploymentMutator(h *valuesHelper) resources.Mutator[*appsv1.Deployment] {
	return &webhooksDeploymentMutator{valuesHelper: h}
}

func (m *webhooksDeploymentMutator) String() string {
	return fmt.Sprintf("deployment %s/%s", m.hostNamespace(), m.landscaperWebhooksFullName())
}

func (m *webhooksDeploymentMutator) Empty() *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.landscaperWebhooksFullName(),
			Namespace: m.hostNamespace(),
		},
	}
}

func (m *webhooksDeploymentMutator) Mutate(r *appsv1.Deployment) error {
	r.ObjectMeta.Labels = m.webhooksComponent.Labels()
	r.Spec = appsv1.DeploymentSpec{
		Replicas: m.values.WebhooksServer.ReplicaCount,
		Selector: &metav1.LabelSelector{MatchLabels: m.webhooksComponent.SelectorLabels()},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: m.webhooksComponent.DeploymentTemplateLabels(),
			},
			Spec: corev1.PodSpec{
				Volumes:                   m.volumes(),
				Containers:                m.containers(),
				NodeSelector:              m.values.NodeSelector,
				SecurityContext:           m.values.PodSecurityContext,
				ImagePullSecrets:          m.values.ImagePullSecrets,
				Affinity:                  m.values.Affinity,
				Tolerations:               m.values.Tolerations,
				TopologySpreadConstraints: m.webhooksComponent.TopologySpreadConstraints(),
			},
		},
	}
	m.setServiceAccount(r.Spec.Template.Spec)
	return nil
}

func (m *webhooksDeploymentMutator) setServiceAccount(podSpec corev1.PodSpec) {
	if m.values.WebhooksServer.LandscaperKubeconfig != nil {
		podSpec.AutomountServiceAccountToken = ptr.To(false)
	} else {
		podSpec.ServiceAccountName = rbac.WebhooksServiceAccountName
	}
}

func (m *webhooksDeploymentMutator) containers() []corev1.Container {
	c := corev1.Container{}
	c.Name = "landscaper-webhooks"
	c.Image = m.webhooksServerImage()
	c.ImagePullPolicy = corev1.PullPolicy(m.values.WebhooksServer.Image.PullPolicy)
	c.Args = m.args()
	c.Resources = m.values.WebhooksServer.Resources
	c.VolumeMounts = m.volumeMounts()
	c.SecurityContext = m.values.SecurityContext
	return []corev1.Container{c}
}

func (m *webhooksDeploymentMutator) volumes() []corev1.Volume {
	volumes := []corev1.Volume{}

	if k := m.values.WebhooksServer.LandscaperKubeconfig; k != nil {
		secretName := ""
		if k.Kubeconfig != "" {
			secretName = m.webhooksKubeconfigSecretName()
		} else {
			secretName = k.SecretRef
		}

		kubeconfigVolume := corev1.Volume{
			Name: "landscaper-cluster-kubeconfig",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		}

		volumes = append(volumes, kubeconfigVolume)
	}

	return volumes
}

func (m *webhooksDeploymentMutator) volumeMounts() []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{}
	if m.values.WebhooksServer.LandscaperKubeconfig != nil {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "landscaper-cluster-kubeconfig",
			MountPath: "/app/ls/landscaper-cluster-kubeconfig",
		})
	}
	return volumeMounts
}

func (m *webhooksDeploymentMutator) webhooksServerImage() string {
	if strings.HasPrefix(m.values.WebhooksServer.Image.Tag, "sha256:") {
		return fmt.Sprintf("%s@%s", m.values.WebhooksServer.Image.Repository, m.values.WebhooksServer.Image.Tag)
	} else {
		return fmt.Sprintf("%s:%s", m.values.WebhooksServer.Image.Repository, m.values.WebhooksServer.Image.Tag)
	}
}

func (m *webhooksDeploymentMutator) args() []string {
	a := []string{}

	if k := m.values.WebhooksServer.LandscaperKubeconfig; k != nil {
		a = append(a, "--kubeconfig=/app/ls/landscaper-cluster-kubeconfig/kubeconfig")

		if m.values.WebhooksServer.Ingress != nil {
			a = append(a, fmt.Sprintf("--webhook-url=%s", m.values.WebhooksServer.Ingress.Host))
		} else {
			a = append(a, fmt.Sprintf("--webhook-url=https://%s.%s:%d", m.landscaperWebhooksFullName(), m.hostNamespace(), m.values.WebhooksServer.ServicePort))
		}

		a = append(a, fmt.Sprintf("--cert-ns=%s", m.values.WebhooksServer.CertificatesNamespace))
	} else {
		a = append(a, fmt.Sprintf("--webhook-service=%s/%s", m.hostNamespace(), m.landscaperWebhooksFullName()))
		a = append(a, fmt.Sprintf("--webhook-service-port=%d", m.values.WebhooksServer.ServicePort))
	}

	if m.values.VerbosityLevel != "" {
		a = append(a, fmt.Sprintf("-v=%s", m.values.VerbosityLevel))
	}

	a = append(a, fmt.Sprintf("--port=%d", m.values.WebhooksServer.ServicePort))

	if len(m.values.WebhooksServer.DisableWebhooks) > 0 {
		a = append(a, fmt.Sprintf("--disable-webhooks=%s", strings.Join(m.values.WebhooksServer.DisableWebhooks, ",")))
	}

	return a
}

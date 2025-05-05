package manifestdeployer

import (
	"fmt"
	"github.com/gardener/landscaper/installer/resources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"
)

type deploymentMutator struct {
	*valuesHelper
}

var _ resources.Mutator[*appsv1.Deployment] = &deploymentMutator{}

func newDeploymentMutator(b *valuesHelper) resources.Mutator[*appsv1.Deployment] {
	return &deploymentMutator{valuesHelper: b}
}

func (d *deploymentMutator) String() string {
	return fmt.Sprintf("deployment %s/%s", d.hostNamespace(), d.deployerFullName())
}

func (d *deploymentMutator) Empty() *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.deployerFullName(),
			Namespace: d.hostNamespace(),
		},
	}
}

func (d *deploymentMutator) Mutate(r *appsv1.Deployment) error {
	r.ObjectMeta.Labels = d.manifestDeployerComponent.Labels()
	r.Spec = appsv1.DeploymentSpec{
		Replicas: d.values.ReplicaCount,
		Selector: &metav1.LabelSelector{MatchLabels: d.manifestDeployerComponent.SelectorLabels()},
		Strategy: d.strategy(),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      d.manifestDeployerComponent.DeploymentTemplateLabels(),
				Annotations: d.templateAnnotations(),
			},
			Spec: corev1.PodSpec{
				Volumes:                   d.volumes(),
				Containers:                d.containers(),
				NodeSelector:              d.values.NodeSelector,
				ServiceAccountName:        d.deployerFullName(),
				SecurityContext:           d.values.PodSecurityContext,
				ImagePullSecrets:          d.values.ImagePullSecrets,
				Affinity:                  d.values.Affinity,
				Tolerations:               d.values.Tolerations,
				TopologySpreadConstraints: d.manifestDeployerComponent.TopologySpreadConstraints(),
			},
		},
	}
	return nil
}

func (d *deploymentMutator) strategy() appsv1.DeploymentStrategy {
	strategy := appsv1.DeploymentStrategy{}
	if d.values.HPA.MaxReplicas == 1 {
		strategy.Type = appsv1.RecreateDeploymentStrategyType
	}
	return strategy
}

func (d *deploymentMutator) templateAnnotations() map[string]string {
	annotations := map[string]string{
		"checksum/config": d.configHash,
	}
	return annotations
}

func (d *deploymentMutator) containers() []corev1.Container {
	c := corev1.Container{}
	c.Name = "manifest-deployer"
	c.Image = d.deployerImage()
	c.Args = d.args()
	c.Env = d.env()
	c.Resources = d.values.Resources
	c.VolumeMounts = d.volumeMounts()
	c.ImagePullPolicy = corev1.PullPolicy(d.values.Image.PullPolicy)
	c.SecurityContext = d.values.SecurityContext
	return []corev1.Container{c}
}

func (d *deploymentMutator) volumes() []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: fmt.Sprintf("%s-config", d.deployerFullName()),
				},
			},
		},
	}

	if k := d.values.LandscaperClusterKubeconfig; k != nil {
		secretName := ""
		if k.Kubeconfig != "" {
			secretName = fmt.Sprintf("%s-landscaper-cluster-kubeconfig", d.deployerFullName())
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

func (d *deploymentMutator) volumeMounts() []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "config",
			MountPath: "/app/ls/config",
		},
	}
	if d.values.LandscaperClusterKubeconfig != nil {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "landscaper-cluster-kubeconfig",
			MountPath: "/app/ls/landscaper-cluster-kubeconfig",
		})
	}
	return volumeMounts
}

func (d *deploymentMutator) deployerImage() string {
	if strings.HasPrefix(d.values.Image.Tag, "sha256:") {
		return fmt.Sprintf("%s@%s", d.values.Image.Repository, d.values.Image.Tag)
	} else {
		return fmt.Sprintf("%s:%s", d.values.Image.Repository, d.values.Image.Tag)
	}
}

func (d *deploymentMutator) args() []string {
	a := []string{
		"--config=/app/ls/config/config.yaml",
	}
	if d.values.LandscaperClusterKubeconfig != nil {
		a = append(a, "--landscaper-kubeconfig=/app/ls/landscaper-cluster-kubeconfig/kubeconfig")
	}
	if d.values.VerbosityLevel != "" {
		a = append(a, fmt.Sprintf("-v=%s", d.values.VerbosityLevel))
	}
	return a
}

func (d *deploymentMutator) env() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: "MY_POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: "MY_POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  "LS_HOST_CLIENT_BURST",
			Value: strconv.FormatInt(int64(d.values.HostClientSettings.Burst), 10),
		},
		{
			Name:  "LS_HOST_CLIENT_QPS",
			Value: strconv.FormatInt(int64(d.values.HostClientSettings.QPS), 10),
		},
		{
			Name:  "LS_RESOURCE_CLIENT_BURST",
			Value: strconv.FormatInt(int64(d.values.ResourceClientSettings.Burst), 10),
		},
		{
			Name:  "LS_RESOURCE_CLIENT_QPS",
			Value: strconv.FormatInt(int64(d.values.ResourceClientSettings.QPS), 10),
		},
	}
}

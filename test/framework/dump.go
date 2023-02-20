// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gardener/landscaper/hack/testcluster/pkg/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

// Dumper is a struct to dump logs and useful information about known object for a state
type Dumper struct {
	kubeClient    client.Client
	kubeClientSet kubernetes.Interface
	namespaces    sets.String //nolint:staticcheck // Ignore SA1019 // TODO: change to generic set
	lsNamespace   string
	logger        utils.Logger
	startTime     time.Time
	endTime       time.Time
}

// NewDumper create a new dumper
func NewDumper(logger utils.Logger, kubeClient client.Client, kubeClientSet kubernetes.Interface, lsNamespace string, namespaces ...string) *Dumper {
	return &Dumper{
		logger:        logger,
		kubeClient:    kubeClient,
		kubeClientSet: kubeClientSet,
		namespaces:    sets.NewString(namespaces...),
		lsNamespace:   lsNamespace,
	}
}

// AddNamespaces adds additional namespaces that should be dumped.
func (d *Dumper) AddNamespaces(namespaces ...string) {
	d.namespaces.Insert(namespaces...)
}

// ClearNamespaces removes all current namespaces
func (d *Dumper) ClearNamespaces() {
	d.namespaces = sets.NewString()
}

// Dump searches for known objects in the given namespaces and dumps useful information about their state.
// Currently information about the main landscaper resources in dumped:
// - Installations
// - DeployItems
// todo: dump additional resources
func (d *Dumper) Dump(ctx context.Context) error {
	d.logger.Logln("Dump")
	if err := d.dumpNamespaces(ctx); err != nil {
		return err
	}
	if len(d.lsNamespace) != 0 {
		// dump ls logs and deployment status
		d.logger.Logfln("--- Landscaper Controller in %s\n", d.lsNamespace)
		if err := d.DumpLandscaperResources(ctx); err != nil {
			return err
		}
	}
	return nil
}

// DumpNamespaces dumps information about all configured namespaces.
func (d *Dumper) dumpNamespaces(ctx context.Context) error {
	for ns := range d.namespaces {
		d.logger.Logfln("Dump %s", ns)
		// check if namespace exists
		if err := d.kubeClient.Get(ctx, kutil.ObjectKey(ns, ""), &corev1.Namespace{}); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		if err := d.dumpInstallationsInNamespace(ctx, ns); err != nil {
			return err
		}
		if err := d.DumpExecutionInNamespace(ctx, ns); err != nil {
			return err
		}
		if err := d.DumpDeployItemsInNamespace(ctx, ns); err != nil {
			return err
		}
		if err := d.DumpConfigMapsInNamespace(ctx, ns); err != nil {
			return err
		}
		if err := d.DumpDeploymentsInNamespace(ctx, ns); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dumper) dumpInstallationsInNamespace(ctx context.Context, namespace string) error {
	installationList := &lsv1alpha1.InstallationList{}
	if err := d.kubeClient.List(ctx, installationList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("unable to list installations for namespace %q: %w", namespace, err)
	}
	for _, inst := range installationList.Items {
		if err := DumpInstallation(d.logger, &inst); err != nil {
			return err
		}
	}
	return nil
}

// DumpInstallation dumps information about the installation
func DumpInstallation(logger utils.Logger, inst *lsv1alpha1.Installation) error {
	// do not dummp the full spec if the installation was successfull
	if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhaseSucceeded {
		logger.Logf("--- Installation %s succeeded\n", inst.Name)
		return nil
	}
	logger.Logf("--- Installation %s\n", inst.Name)
	logger.Logf("%s\n", FormatAsYAML(inst.Spec, 0))
	logger.Logf("%s\n", FormatAsYAML(inst.Status, 0))
	return nil
}

// DumpDeployItemsInNamespace dumps information about all deploy items int he given namespace
func (d *Dumper) DumpDeployItemsInNamespace(ctx context.Context, namespace string) error {
	list := &lsv1alpha1.DeployItemList{}
	if err := d.kubeClient.List(ctx, list, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("unable to list deploy items for namespace %q: %w", namespace, err)
	}
	for _, item := range list.Items {
		if err := DumpDeployItems(d.logger, &item); err != nil {
			return err
		}
	}
	return nil
}

// DumpDeployItems dumps information about the deploy items
func DumpDeployItems(logger utils.Logger, deployItem *lsv1alpha1.DeployItem) error {
	if deployItem.Status.Phase == lsv1alpha1.ExecutionPhaseSucceeded {
		logger.Logf("--- DeployItem %s succeeded\n", deployItem.Name)
		return nil
	}
	fmtMsg := `
--- DeployItem %s
Annotations: %s
Type: %s
Config: %s
`

	configData := []byte("no config")
	if deployItem.Spec.Configuration != nil {
		var configData interface{}
		if err := yaml.Unmarshal(deployItem.Spec.Configuration.Raw, &configData); err != nil {
			configData = map[string]string{
				"error": err.Error(),
			}
		}
	}

	logger.Logf(fmtMsg,
		deployItem.Name,
		FormatAsYAML(deployItem.Annotations, 2),
		deployItem.Spec.Type,
		FormatAsYAML(configData, 2))
	fmtMsg = `
Status:
  Phase: %s
  Error: %s
  ProviderConfig: %s
`

	logger.Logf(fmtMsg,
		deployItem.Status.Phase,
		FormatLastError(deployItem.Status.GetLastError(), "    "),
		FormatAsYAML(deployItem.Status.ProviderStatus, 4))
	return nil
}

// DumpExecutionInNamespace dumps all executions in a namespace
func (d *Dumper) DumpExecutionInNamespace(ctx context.Context, namespace string) error {
	executionList := &lsv1alpha1.ExecutionList{}
	if err := d.kubeClient.List(ctx, executionList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("unable to list installations for namespace %q: %w", namespace, err)
	}
	for _, exec := range executionList.Items {
		if err := DumpExecution(d.logger, &exec); err != nil {
			return err
		}
	}
	return nil
}

// DumpExecution dumps information about the execution
func DumpExecution(logger utils.Logger, exec *lsv1alpha1.Execution) error {
	if exec.Status.ExecutionPhase == lsv1alpha1.ExecPhaseSucceeded {
		logger.Logf("--- Execution %s succeeded\n", exec.Name)
		return nil
	}
	logger.Logf("--- Execution %s\n", exec.Name)
	logger.Logf("%s\n", FormatAsYAML(exec.Spec, 0))
	logger.Logf("%s\n", FormatAsYAML(exec.Status, 0))
	return nil
}

// DumpConfigMapsInNamespace dumps all configmaps in a namespace
func (d *Dumper) DumpConfigMapsInNamespace(ctx context.Context, namespace string) error {
	cmList := &corev1.ConfigMapList{}
	if err := d.kubeClient.List(ctx, cmList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("unable to list installations for namespace %q: %w", namespace, err)
	}
	for _, cm := range cmList.Items {
		// ignore default configmap
		if cm.Name == "kube-root-ca.crt" {
			continue
		}
		if err := DumpConfigMap(d.logger, &cm); err != nil {
			return err
		}
	}
	return nil
}

// DumpConfigMap dumps information about the configmap
func DumpConfigMap(logger utils.Logger, cm *corev1.ConfigMap) error {
	logger.Logf("--- ConfigMap %s\n", cm.Name)
	logger.Logf("%s\n", FormatAsYAML(cm.Data, 0))
	return nil
}

// DumpLandscaperResources dumps all landscaper resources in the ls namespace.
func (d *Dumper) DumpLandscaperResources(ctx context.Context) error {
	deployments := &appsv1.DeploymentList{}
	if err := d.kubeClient.List(ctx, deployments,
		client.InNamespace(d.lsNamespace),
		client.HasLabels{lsv1alpha1.LandscaperComponentLabelName}); err != nil {
		return fmt.Errorf("unable to list deployments for namespace %q: %w", d.lsNamespace, err)
	}
	for _, deploy := range deployments.Items {
		// skip landscaper webhook if ready and no restarts
		if deploy.Name == "ls-landscaper-webhooks" {
			if deploy.Spec.Replicas == nil || deploy.Status.ReadyReplicas == *deploy.Spec.Replicas {
				continue
			}
		}
		if err := DumpDeployment(d.logger, &deploy); err != nil {
			return err
		}
		d.logger.Logf("Pods: %s",
			d.FormatPodsWithSelector(ctx, 2, client.InNamespace(d.lsNamespace), client.MatchingLabels(deploy.Spec.Template.Labels)))
	}

	// dump deployer lcm resources
	if err := d.DumpEnvironments(ctx); err != nil {
		return err
	}
	if err := d.DumpDeployerRegistrations(ctx); err != nil {
		return err
	}
	if err := d.dumpInstallationsInNamespace(ctx, d.lsNamespace); err != nil {
		return err
	}
	if err := d.DumpExecutionInNamespace(ctx, d.lsNamespace); err != nil {
		return err
	}
	if err := d.DumpDeployItemsInNamespace(ctx, d.lsNamespace); err != nil {
		return err
	}

	deployments = &appsv1.DeploymentList{}
	req, err := labels.NewRequirement(lsv1alpha1.LandscaperComponentLabelName, selection.DoesNotExist, nil)
	if err != nil {
		return err
	}
	sel := labels.NewSelector().Add(*req)
	if err := d.kubeClient.List(ctx, deployments,
		client.InNamespace(d.lsNamespace),
		client.MatchingLabelsSelector{Selector: sel}); err != nil {
		return fmt.Errorf("unable to list deployments for namespace %q: %w", d.lsNamespace, err)
	}
	for _, deploy := range deployments.Items {
		if err := DumpDeployment(d.logger, &deploy); err != nil {
			return err
		}
		d.logger.Logf("Pods: %s",
			d.FormatPodsWithSelector(ctx, 2, client.InNamespace(d.lsNamespace), client.MatchingLabels(deploy.Spec.Template.Labels)))
	}

	return nil
}

// DumpEnvironments dumps all environments that are in the current system
func (d *Dumper) DumpEnvironments(ctx context.Context) error {
	envList := &lsv1alpha1.EnvironmentList{}
	if err := d.kubeClient.List(ctx, envList); err != nil {
		return fmt.Errorf("unable to list environments: %w", err)
	}
	for _, env := range envList.Items {
		if err := DumpEnvironment(d.logger, &env); err != nil {
			return err
		}
	}
	return nil
}

// DumpEnvironment dumps information about the environment
func DumpEnvironment(logger utils.Logger, env *lsv1alpha1.Environment) error {
	logger.Logf("--- Environment %s\n", env.Name)
	logger.Logf("%s\n", FormatAsYAML(env.Spec, 0))
	return nil
}

// DumpDeployerRegistrations dumps all deployer registrations that are in the current system
func (d *Dumper) DumpDeployerRegistrations(ctx context.Context) error {
	drList := &lsv1alpha1.DeployerRegistrationList{}
	if err := d.kubeClient.List(ctx, drList); err != nil {
		return fmt.Errorf("unable to list environments: %w", err)
	}
	for _, dr := range drList.Items {
		if err := DumpDeployerRegistration(d.logger, &dr); err != nil {
			return err
		}
	}
	return nil
}

// DumpDeployerRegistration dumps information about the deployer registration
func DumpDeployerRegistration(logger utils.Logger, dr *lsv1alpha1.DeployerRegistration) error {
	logger.Logf("--- Deployer Registration %s\n", dr.Name)
	logger.Logf("%s\n", FormatAsYAML(dr.Spec, 0))
	return nil
}

// FormatPodsWithSelector returns formatted pods that match a selector.
func (d *Dumper) FormatPodsWithSelector(ctx context.Context, indent int, opts ...client.ListOption) string {
	pods := &corev1.PodList{}
	if err := d.kubeClient.List(ctx, pods, opts...); err != nil {
		return fmt.Sprintf("error: unable to list pods for namespace %q: %s", d.lsNamespace, err.Error())
	}
	podList := make([]string, len(pods.Items))
	for i, pod := range pods.Items {
		podList[i] = FormatPod(ctx, &pod, d.kubeClientSet, 0, d.startTime, d.endTime)
	}
	return FormatList(podList, indent)
}

// DumpDeploymentsInNamespace dumps all deployment resources in a namespace.
func (d *Dumper) DumpDeploymentsInNamespace(ctx context.Context, ns string) error {
	deployments := &appsv1.DeploymentList{}
	if err := d.kubeClient.List(ctx, deployments, client.InNamespace(ns)); err != nil {
		return fmt.Errorf("unable to list deployments for namespace %q: %w", ns, err)
	}
	for _, deploy := range deployments.Items {
		if err := DumpDeployment(d.logger, &deploy); err != nil {
			return err
		}
		d.logger.Logf("Pods: %s",
			d.FormatPodsWithSelector(ctx, 2, client.InNamespace(d.lsNamespace), client.MatchingLabels(deploy.Spec.Template.Labels)))
	}
	return nil
}

// DumpDeployment dumps information about the deployment
func DumpDeployment(logger utils.Logger, deployment *appsv1.Deployment) error {
	containerFmt := `
--- Deployment %s
Containers: %s
Status:
  Ready: %d/%d
`
	logger.Logf(containerFmt,
		client.ObjectKeyFromObject(deployment).String(),
		FormatContainers(deployment.Spec.Template.Spec.Containers, 2),
		deployment.Status.ReadyReplicas, deployment.Status.Replicas)
	return nil
}

// FormatContainers returns a pretty printed representation of a list of containers
func FormatContainers(containers []corev1.Container, indent int) string {
	if len(containers) == 0 {
		return "none"
	}
	list := make([]string, 0)
	containerFmt := `
Name: %s
Image: %s
`
	for _, container := range containers {
		list = append(list, fmt.Sprintf(containerFmt, container.Name, container.Image))
	}
	return FormatList(list, indent)
}

// FormatPod returns information about the pod.
// It also fetches the pods logs if a client is provided.
func FormatPod(ctx context.Context, pod *corev1.Pod, kubeClientSet kubernetes.Interface, indent int, start, end time.Time) string {
	podFmt := `
Name: %s
Containers: %s
Status:
  Phase: %s
  Reason: %s
  Message: %s
  Containers: %s
`
	out := fmt.Sprintf(podFmt,
		client.ObjectKeyFromObject(pod).String(),
		FormatContainers(pod.Spec.Containers, 2),
		pod.Status.Phase, pod.Status.Reason, pod.Status.Message,
		FormatContainerStatuses(ctx, pod, 4, kubeClientSet, start, end),
	)
	return ApplyIdent(out, indent)
}

// FormatContainerStatuses formats the container statuses of a pod.
func FormatContainerStatuses(ctx context.Context, pod *corev1.Pod, indent int, kubeClientSet kubernetes.Interface, start, end time.Time) string {
	statuses := make([]string, len(pod.Status.ContainerStatuses))
	for i, status := range pod.Status.ContainerStatuses {
		logs := ""
		if kubeClientSet != nil {
			containerLogs, err := GetContainerLogs(ctx, kubeClientSet, pod.GetName(), pod.GetNamespace(), status.Name, start, end)
			if err != nil {
				logs = fmt.Sprintf("error while fetching: %s", err.Error())
			} else {
				logs = ApplyIdent(string(containerLogs), 2)
			}
		}

		statuses[i] = FormatContainerStatus(status, logs, 0)
	}

	return FormatList(statuses, indent)
}

// GetContainerLogs returns the logs of a container.
func GetContainerLogs(ctx context.Context, kubeClientSet kubernetes.Interface, podName, podNamespace, containerName string, start, end time.Time) ([]byte, error) {
	req := kubeClientSet.CoreV1().Pods(podNamespace).GetLogs(podName, &corev1.PodLogOptions{
		Container:    containerName,
		Follow:       false,
		Previous:     false,
		SinceSeconds: nil,
		SinceTime:    &metav1.Time{Time: start},
		Timestamps:   false,
	})
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer podLogs.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, podLogs); err != nil {
		return nil, err
	}
	return buf.Bytes(), err
}

// FormatContainerStatus returns a formatted container status.
func FormatContainerStatus(status corev1.ContainerStatus, logs string, indent int) string {
	statusFmt := `
Name: %s
State: %s
Logs: %s
`
	return ApplyIdent(fmt.Sprintf(statusFmt, status.Name, status.State.String(), logs), indent)
}

// FormatList creates a human readable list with the given indent.
func FormatList(list []string, indent int) string {
	out := "\n"
	for _, item := range list {
		out = out + "- " + strings.ReplaceAll(strings.TrimLeft(item, "\n"), "\n", "\n  ")
	}
	return ApplyIdent(out, indent)
}

// StringIndent creates the indentation for a number
func StringIndent(indent int) string {
	return strings.Repeat(" ", indent)
}

// ApplyIdent applies the indentation to a string
func ApplyIdent(s string, indent int) string {
	if indent == 0 {
		return s
	}
	return strings.ReplaceAll(s, "\n", "\n"+StringIndent(indent))
}

// FormatContainers returns a pretty printed representation of a list of containers
func FormatContainersStatus(containers []corev1.ContainerStatus, indent string) string {
	if len(containers) == 0 {
		return "none"
	}
	list := "\n"
	containerFmt := `
Name: %s
Ready: %v
`
	for _, container := range containers {
		list = list + "\n-" + fmt.Sprintf(containerFmt, container.Name, container.Ready)
	}
	return strings.ReplaceAll(list, "\n", "\n"+indent)
}

// FormatAsYAML formats a object as yaml
func FormatAsYAML(obj interface{}, indent int) string {
	if obj == nil {
		return "none"
	}
	data, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Sprintf("Error during yaml serialization: %s", err.Error())
	}
	// add an additional newline to properly inline
	return ApplyIdent("\n"+string(data), indent)
}

// FormatLastError formats a error in a human readable format.
func FormatLastError(err *lsv1alpha1.Error, indent string) string {
	if err == nil {
		return "none"
	}
	format := `

Operation: %s
Reason: %s
Message: %s
LastUpdated: %s
`
	format = strings.ReplaceAll(format, "\n", "\n"+indent)
	return fmt.Sprintf(format, err.Operation, err.Reason, err.Message, err.LastUpdateTime.String())
}

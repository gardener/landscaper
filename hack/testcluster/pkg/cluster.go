// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package pkg

import (
	"bytes"
	"context"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/gardener/landscaper/hack/testcluster/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

// ClusterIdLabelName is the name of the label that holds the unique id for the current created cluster.
// It is used by the delete command to find the created cluster.
const ClusterIdLabelName = "cluster.test.landscaper.gardener.cloud/id"

// ComponentLabelName is the name of the label that specifies the component that is deployed
const ComponentLabelName = "cluster.test.landscaper.gardener.cloud/component"

const (
	ComponentCluster  = "cluster"
	ComponentRegistry = "registry"
)

// DefaultK8sVersion defines the default k3s version.
// The real version is appended with a "-k3s1"
const DefaultK8sVersion = "v1.20.8"

// K3sImageRepository is the repository name of the k3s images.
const K3sImageRepository = "docker.io/rancher/k3s"

// see this issue for more discussions about min resources for a k8s cluster running in a pod.
// we will stick for now with req of 1GB RAM/1 CPU and limits up to 3GB RAM/3 CPU as other requests have lead to instabilities.
const podTmpl = `
apiVersion: v1
kind: Pod
metadata:
  generateName: k3s-cluster-
spec:
  containers:
  - image: docker.io/rancher/k3s:v1.20.8-k3s1
    imagePullPolicy: IfNotPresent
    name: cluster
    stdin: true
    tty: true
    args: ["server", "--tls-san", "$(API_SERVER_ADDRESS)", "--https-listen-port", "6443", "--kube-apiserver-arg", "token-auth-file=/shared/token-auth.csv"]
    env:
    - name: K3S_KUBECONFIG_OUTPUT
      value: "/kubeconfig.yaml"
    - name: API_SERVER_ADDRESS
      valueFrom:
        fieldRef:
          fieldPath: status.podIP
    securityContext:
      privileged: true
    ports:
    - containerPort: 6443
      name: api-server-port
      protocol: TCP
    resources:
      requests:
        memory: "1G"
        cpu: "1"
      limits:
        memory: "3G"
        cpu: "3"
    volumeMounts:
    - name: shared-data
      mountPath: /shared
    readinessProbe:
      failureThreshold: 15
      httpGet:
        httpHeaders:
        - name: Authorization
          value: Bearer {{ .k8s.authToken }}
        path: /readyz
        port: api-server-port
        scheme: HTTPS
      initialDelaySeconds: 30
      periodSeconds: 20
      successThreshold: 1
      timeoutSeconds: 1
  initContainers:
  - name: set-auth-token
    image: busybox:1.28
    command: ['sh', '-c', "echo '{{ .k8s.authToken }},kube-apiserver-health-check,kube-apiserver-health-check' > /shared/token-auth.csv"]
    volumeMounts:
    - name: shared-data
      mountPath: /shared
  volumes:
  - name: shared-data
    emptyDir: {}
`

// State defines the content of the state file that is written by the create command
// and consumed by the delete command.
type State struct {
	ID string `json:"id"`
}

// CreateClusterArgs defines the arguments to create a cluster.
type CreateClusterArgs struct {
	KubeClient           client.Client
	RestConfig           *rest.Config
	Namespace            string
	ID                   string
	StateFile            string
	ExportKubeconfigPath string
	Timeout              time.Duration
	KubernetesVersion    string
}

// CreateCluster creates a new k3d cluster running in a pod.
func CreateCluster(ctx context.Context, logger utils.Logger, args CreateClusterArgs) (err error) {
	var (
		kubeClient           = args.KubeClient
		stateFile            = args.StateFile
		exportKubeconfigPath = args.ExportKubeconfigPath
		kubernetesVersion    = args.KubernetesVersion
	)
	if len(kubernetesVersion) == 0 {
		// todo: validate k8s version exists as k3s release
		kubernetesVersion = DefaultK8sVersion
	}
	// parse and template pod
	token := generateToken()
	tmpl, err := template.New("pod").Parse(podTmpl)
	if err != nil {
		return err
	}
	var podBytes bytes.Buffer
	if err := tmpl.Execute(&podBytes, map[string]interface{}{
		"k8s": map[string]interface{}{
			"authToken": token,
		},
	}); err != nil {
		return err
	}

	pod := &corev1.Pod{}
	if _, _, err := serializer.NewCodecFactory(scheme.Scheme).UniversalDecoder().Decode(podBytes.Bytes(), nil, pod); err != nil {
		return err
	}
	pod.Namespace = args.Namespace
	pod.Spec.Containers[0].Image = fmt.Sprintf("%s:%s-k3s1", K3sImageRepository, kubernetesVersion)
	kutil.SetMetaDataLabel(pod, ComponentLabelName, ComponentCluster)
	kutil.SetMetaDataLabel(pod, ClusterIdLabelName, args.ID)

	if err := kubeClient.Create(ctx, pod); err != nil {
		return fmt.Errorf("unable to create cluster pod: %w", err)
	}
	logger.Logfln("Created cluster %q from image %q running kubernetes version %q", pod.Name, pod.Spec.Containers[0].Image, kubernetesVersion)

	// register cleanup to delete the cluster if something fails
	defer func() {
		if err == nil {
			return
		}
		if err := cleanupPod(ctx, logger, ComponentCluster, kubeClient, pod, args.Timeout); err != nil {
			logger.Logfln("Error while cleanup of the cluster: %s", err.Error())
		}
	}()

	err = wait.PollImmediate(10*time.Second, args.Timeout, func() (done bool, err error) {
		updatedPod := &corev1.Pod{}
		if err := kubeClient.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, updatedPod); err != nil {
			return false, err
		}
		*pod = *updatedPod

		if updatedPod.Status.Phase != corev1.PodRunning {
			logger.Logln("Waiting for cluster pod to be up and running...")
			return false, nil
		}

		// check pod status
		if len(updatedPod.Status.ContainerStatuses) == 0 {
			logger.Logln("Waiting for cluster pod to be up and running...")
			return false, nil
		}
		if !updatedPod.Status.ContainerStatuses[0].Ready {
			logger.Logln("Waiting for cluster to be ready...")
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return err
	}
	logger.Logln("Successfully created cluster")

	logger.Logln("Get kubeconfig for cluster...")
	kubeconfigBytes, err := getKubeconfigFromCluster(logger, args.RestConfig, pod)
	if err != nil {
		return fmt.Errorf("unable to read kubeconfig from pod: %w", err)
	}

	// update the host in the kubeconfig
	kubeconfigBytes = []byte(strings.Replace(
		string(kubeconfigBytes),
		"server: https://127.0.0.1:6443",
		fmt.Sprintf("server: https://%s:6443", pod.Status.PodIP),
		1))

	// write state if path is provided
	if len(stateFile) != 0 {
		if err := os.MkdirAll(filepath.Dir(stateFile), os.ModePerm); err != nil {
			return fmt.Errorf("unable to create state directory %q: %w", filepath.Dir(stateFile), err)
		}

		data, err := json.Marshal(State{ID: args.ID})
		if err != nil {
			return fmt.Errorf("unable to marshal state: %w", err)
		}
		if err := ioutil.WriteFile(stateFile, data, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write statefile to %q: %w", stateFile, err)
		}
		logger.Logfln("Successfully written state to %q", stateFile)
	}

	if len(exportKubeconfigPath) == 0 {
		logger.Logln("No export path for the kubeconfig defined")
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(exportKubeconfigPath), os.ModePerm); err != nil {
		return fmt.Errorf("unable to create export directory %q: %w", filepath.Dir(exportKubeconfigPath), err)
	}
	if err := ioutil.WriteFile(exportKubeconfigPath, kubeconfigBytes, os.ModePerm); err != nil {
		return fmt.Errorf("unable to write kubeconfig to %q: %w", exportKubeconfigPath, err)
	}
	logger.Logfln("Successfully written kubeconfig to %q", exportKubeconfigPath)

	return nil
}

func getKubeconfigFromCluster(logger utils.Logger, restConfig *rest.Config, pod *corev1.Pod) ([]byte, error) {
	var (
		kubeconfigBytes []byte
		err             error
	)
	pollErr := wait.PollImmediate(10*time.Second, 5*time.Minute, func() (bool, error) {
		kubeconfigBytes, err = executeCommandOnPod(logger, restConfig, pod.Name, pod.Namespace, "cluster", "cat /kubeconfig.yaml")
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if pollErr != nil {
		if err == nil {
			return nil, pollErr
		}
		logger.Logfln("Error while trying to fetch kubeconfig: %s", pollErr.Error())
	}
	return kubeconfigBytes, err
}

// DeleteCluster deletes a previously deployed test cluster.
func DeleteCluster(ctx context.Context,
	logger utils.Logger,
	kubeClient client.Client,
	namespace string,
	id string,
	timeout time.Duration) error {

	podList := &corev1.PodList{}
	if err := kubeClient.List(ctx, podList, client.InNamespace(namespace), client.MatchingLabels{
		ClusterIdLabelName: id,
		ComponentLabelName: ComponentCluster,
	}); err != nil {
		return fmt.Errorf("unable to get pods for id %q in namespace %q: %w", id, namespace, err)
	}

	for _, pod := range podList.Items {
		if err := cleanupPod(ctx, logger, ComponentCluster, kubeClient, &pod, timeout); err != nil {
			return err
		}
	}

	return nil
}

// cleanupPod deletes the cluster that is running in the given pod.
func cleanupPod(ctx context.Context, logger utils.Logger, componentName string, kubeClient client.Client, pod *corev1.Pod, timeout time.Duration) error {
	logger.Logfln("Cleanup %s in pod %q", componentName, pod.GetName())
	err := wait.PollImmediate(10*time.Second, 1*time.Minute, func() (done bool, err error) {
		if err := kubeClient.Delete(ctx, pod); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			logger.Logfln("Error while trying to delete %s (%s)...", componentName, err.Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("unable to delete %s in pod %q", componentName, pod.GetName())
	}
	err = wait.PollImmediate(10*time.Second, timeout, func() (done bool, err error) {
		updatedPod := &corev1.Pod{}
		if err := kubeClient.Get(ctx, kutil.ObjectKey(pod.GetName(), pod.GetNamespace()), updatedPod); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			logger.Logfln("Error while trying to %s cluster (%s)...", componentName, err.Error())
			return false, nil
		}
		logger.Logfln("Waiting for the %s to be deleted ...", componentName)
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("unable to delete %s in pod %q", componentName, pod.GetName())
	}
	logger.Logfln("Successfully deleted %s in pod %q", componentName, pod.GetName())
	return nil
}

func executeCommandOnPod(logger utils.Logger, restConfig *rest.Config, name, namespace, containerName, command string) ([]byte, error) {
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to build kubernetes client set: %w", err)
	}
	var stdout, stderr bytes.Buffer
	request := client.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(name).
		Namespace(namespace).
		SubResource("exec").
		Param("container", containerName).
		Param("command", "/bin/sh").
		Param("stdin", "true").
		Param("stdout", "true").
		Param("stderr", "true").
		Param("tty", "false")

	executor, err := remotecommand.NewSPDYExecutor(restConfig, http.MethodPost, request.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to initialized the command exector: %v", err)
	}

	err = executor.Stream(remotecommand.StreamOptions{
		Stdin:  strings.NewReader(command),
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		logger.Logfln("Remote command %q failed", command)
		logger.Logfln("Stdout: %s", stdout.String())
		logger.Logfln("Stderr: %s", stderr.String())
		return nil, fmt.Errorf("remote command failed: %w", err)
	}
	return stdout.Bytes(), nil
}

func generateToken() string {
	const length = 20
	token := make([]byte, length)
	_, _ = rand.Read(token)
	return strings.ReplaceAll(base32.StdEncoding.EncodeToString(token), "=", "")
}

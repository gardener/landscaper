// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/gardener/landscaper/pkg/utils/simplelogger"
)

// ClusterIdLabelName is the name of the label that holds the unique id for the current created cluster.
// It is used by the delete command to find the created cluster.
const ClusterIdLabelName = "cluster.test.landscaper.gardener.cloud/id"

// see this issue for more discussions around min resources for a k8s cluster running in a pod.
// we will stick for now with req of 500M RAM/1 CPU
const podTmpl = `
apiVersion: v1
kind: Pod
metadata:
  generateName: k3s-cluster-
spec:
  containers:
  - image: docker.io/rancher/k3s:latest
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
        memory: "500M"
        cpu: "1"
      limits:
        memory: "2G"
        cpu: "2"
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

func createCluster(ctx context.Context, logger simplelogger.Logger, opts *Options) (err error) {
	kubeClient := opts.kubeClient

	// generate id if none is defined
	if len(opts.ID) == 0 {
		uid, err := uuid.NewUUID()
		if err != nil {
			return fmt.Errorf("unable to generate uuid: %w", err)
		}
		opts.ID = base64.StdEncoding.EncodeToString([]byte(uid.String()))
	}

	// parse and template pod
	token := "asdlfjasdoifjsadfasfasf"
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
	pod.Namespace = opts.Namespace
	kutil.SetMetaDataLabel(pod, ClusterIdLabelName, opts.ID)

	if err := kubeClient.Create(ctx, pod); err != nil {
		return fmt.Errorf("unable to create cluster pod: %w", err)
	}
	logger.Logfln("Created cluster %q", pod.Name)

	// register cleanup to delete the cluster if something fails
	defer func() {
		if err == nil {
			return
		}
		if err := cleanupCluster(ctx, logger, kubeClient, pod, opts.Timeout); err != nil {
			logger.Logfln("Error while cleanup of the cluster: %s", err.Error())
		}
	}()

	err = wait.PollImmediate(10*time.Second, opts.Timeout, func() (done bool, err error) {
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
	kubeconfigBytes, err := getKubeconfigFromCluster(logger, pod, opts)
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
	if len(opts.StateFile) != 0 {
		if err := os.MkdirAll(filepath.Dir(opts.StateFile), os.ModePerm); err != nil {
			return fmt.Errorf("unable to create state directory %q: %w", filepath.Dir(opts.StateFile), err)
		}

		data, err := json.Marshal(State{ID: opts.ID})
		if err != nil {
			return fmt.Errorf("unable to marshal state: %w", err)
		}
		if err := ioutil.WriteFile(opts.StateFile, data, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write statefile to %q: %w", opts.StateFile, err)
		}
		logger.Logfln("Successfully written state to %q", opts.StateFile)
	}

	if len(opts.ExportKubeconfigPath) == 0 {
		logger.Logln("No export path for the kubeconfig defined")
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(opts.ExportKubeconfigPath), os.ModePerm); err != nil {
		return fmt.Errorf("unable to create export directory %q: %w", filepath.Dir(opts.ExportKubeconfigPath), err)
	}
	if err := ioutil.WriteFile(opts.ExportKubeconfigPath, kubeconfigBytes, os.ModePerm); err != nil {
		return fmt.Errorf("unable to write kubeconfig to %q: %w", opts.ExportKubeconfigPath, err)
	}
	logger.Logfln("Successfully written kubeconfig to %q", opts.ExportKubeconfigPath)

	return nil
}

func getKubeconfigFromCluster(logger simplelogger.Logger, pod *corev1.Pod, opts *Options) ([]byte, error) {
	var (
		kubeconfigBytes []byte
		err             error
	)
	pollErr := wait.PollImmediate(10*time.Second, 5*time.Minute, func() (bool, error) {
		kubeconfigBytes, err = executeCommandOnPod(logger, opts.restConfig, pod.Name, pod.Namespace, "cluster", "cat /kubeconfig.yaml")
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

func deleteCluster(ctx context.Context, logger simplelogger.Logger, opts *Options) error {
	kubeClient := opts.kubeClient
	if len(opts.ID) == 0 {
		// statefile should be defined as it is already checked by the calling function
		data, err := ioutil.ReadFile(opts.StateFile)
		if err != nil {
			return fmt.Errorf("unable to read state file %q: %w", opts.StateFile, err)
		}
		state := State{}
		if err := json.Unmarshal(data, &state); err != nil {
			return fmt.Errorf("unable to decode state from %q: %w", opts.StateFile, err)
		}
		opts.ID = state.ID
	}

	if len(opts.ID) == 0 {
		return errors.New("no id defined by flag or statefile")
	}

	podList := &corev1.PodList{}
	if err := kubeClient.List(ctx, podList, client.InNamespace(opts.Namespace), client.MatchingLabels{
		ClusterIdLabelName: opts.ID,
	}); err != nil {
		return fmt.Errorf("unable to get pods for id %q in namespace %q: %w", opts.ID, opts.Namespace, err)
	}

	for _, pod := range podList.Items {
		if err := cleanupCluster(ctx, logger, kubeClient, &pod, opts.Timeout); err != nil {
			return err
		}
	}

	return nil
}

// cleanupCluster deletes the cluster that is running in the given pod.
func cleanupCluster(ctx context.Context, logger simplelogger.Logger, kubeClient client.Client, pod *corev1.Pod, timeout time.Duration) error {
	logger.Logfln("Cleanup cluster in pod %q", pod.GetName())
	err := wait.PollImmediate(10*time.Second, 1*time.Minute, func() (done bool, err error) {
		if err := kubeClient.Delete(ctx, pod); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			logger.Logfln("Error while trying to delete cluster (%s)...", err.Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("unable to delete cluster in pod %q", pod.GetName())
	}
	err = wait.PollImmediate(10*time.Second, timeout, func() (done bool, err error) {
		updatedPod := &corev1.Pod{}
		if err := kubeClient.Get(ctx, kutil.ObjectKey(pod.GetName(), pod.GetNamespace()), updatedPod); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			logger.Logfln("Error while trying to delete cluster (%s)...", err.Error())
			return false, nil
		}
		logger.Logln("Waiting for the cluster to be deleted ...")
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("unable to delete cluster in pod %q", pod.GetName())
	}
	logger.Logfln("Successfully deleted cluster in pod %q", pod.GetName())
	return nil
}

func executeCommandOnPod(logger simplelogger.Logger, restConfig *rest.Config, name, namespace, containerName, command string) ([]byte, error) {
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

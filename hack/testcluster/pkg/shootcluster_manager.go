// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package pkg

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/jsonpath"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/pkg/utils/simplelogger"
)

//go:embed resources/shootcluster_template.yaml
var shootClusterTemplate string

const (
	// namePrefix is the name prefix of shoot clusters used for integration tests
	namePrefix = "it-"

	// the maximum number of shoot clusters used for integration tests
	maxTestShoots = 5

	// name of the file that will be created in the auth directory, containing the name of the shoot cluster
	filenameForClusterName = "clustername"

	// name of the file that will be created in the auth directory, containing the kubeconfig of the shoot cluster
	filenameForKubeconfig = "kubeconfig.yaml"
)

var (
	shootGVR = schema.GroupVersionResource{
		Group:    "core.gardener.cloud",
		Version:  "v1beta1",
		Resource: "shoots",
	}

	secretGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "secrets",
	}
)

type ShootClusterManager struct {
	log                         simplelogger.Logger
	gardenClusterKubeconfigPath string
	namespace                   string
	authDirectoryPath           string
}

func NewShootClusterManager(log simplelogger.Logger, gardenClusterKubeconfigPath, namespace, authDirectoryPath string) *ShootClusterManager {
	return &ShootClusterManager{
		log:                         log,
		gardenClusterKubeconfigPath: gardenClusterKubeconfigPath,
		namespace:                   namespace,
		authDirectoryPath:           authDirectoryPath,
	}
}

func (o *ShootClusterManager) CreateShootCluster(ctx context.Context) error {
	gardenClient, err := o.createGardenClient()
	if err != nil {
		return err
	}

	gardenClientForShoots := gardenClient.Resource(shootGVR).Namespace(o.namespace)
	gardenClientForSecrets := gardenClient.Resource(secretGVR).Namespace(o.namespace)

	if err := o.checkNumberOfShoots(ctx, gardenClientForShoots); err != nil {
		return err
	}

	clusterName := o.generateShootName()

	if err := o.ensureAuthDirectory(); err != nil {
		return err
	}

	if err := o.writeClusterName(clusterName); err != nil {
		return err
	}

	shoot, err := o.createShootManifest(clusterName)
	if err != nil {
		return err
	}

	_, err = gardenClientForShoots.Create(ctx, shoot, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	if err := o.waitUntilShootClusterIsReady(ctx, gardenClientForShoots, clusterName); err != nil {
		return err
	}

	if err := o.writeKubeconfig(ctx, gardenClientForSecrets, clusterName); err != nil {
		return err
	}

	return nil
}

func (o *ShootClusterManager) DeleteShootCluster(ctx context.Context) error {
	gardenClient, err := o.createGardenClient()
	if err != nil {
		return err
	}

	gardenClientForShoots := gardenClient.Resource(shootGVR).Namespace(o.namespace)

	clusterName, err := o.readClusterName()
	if err != nil {
		return err
	}

	shoot, err := gardenClientForShoots.Get(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to read shoot %s: %w", clusterName, err)
	}

	o.addConfirmAnnotation(shoot)

	_, err = gardenClientForShoots.Update(ctx, shoot, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("unable to update shoot %s: %w", clusterName, err)
	}

	if err = gardenClientForShoots.Delete(ctx, clusterName, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

func (o *ShootClusterManager) createGardenClient() (dynamic.Interface, error) {
	restConfig, err := clientcmd.BuildConfigFromFlags("", o.gardenClusterKubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read kubeconfig from %s: %w", o.gardenClusterKubeconfigPath, err)
	}

	gardenClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create kubernetes client from %s: %w", o.gardenClusterKubeconfigPath, err)
	}

	return gardenClient, nil
}

func (o *ShootClusterManager) checkNumberOfShoots(ctx context.Context, gardenClient dynamic.ResourceInterface) error {
	shootList, err := gardenClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	numTestShoots := 0
	for _, shoot := range shootList.Items {
		name, err := o.getNameOfUnstructuredResource(shoot)
		if err != nil {
			return err
		}

		if o.matchesNamePattern(name) {
			numTestShoots = numTestShoots + 1
			if numTestShoots >= maxTestShoots {
				return fmt.Errorf("the maximum number of %d test clusters in namespace %s must not be exceeded", maxTestShoots, o.namespace)
			}
		}
	}

	return nil
}

func (o *ShootClusterManager) generateShootName() string {
	rand.Seed(time.Now().UnixNano())
	return namePrefix + strconv.Itoa(rand.Intn(9000)+1000)
}

func (o *ShootClusterManager) matchesNamePattern(name string) bool {
	return strings.HasPrefix(name, namePrefix)
}

func (o *ShootClusterManager) getNameOfUnstructuredResource(shoot unstructured.Unstructured) (string, error) {
	jp := jsonpath.New("name")
	if err := jp.Parse("{.metadata.name}"); err != nil {
		return "", fmt.Errorf("failed to get name of shoot cluster: template parsing failed: %w", err)
	}

	result, err := jp.FindResults(shoot.Object)
	if err != nil {
		return "", fmt.Errorf("failed to get name of shoot cluster: result not found: %w", err)
	}

	if len(result) != 1 || len(result[0]) != 1 {
		return "", fmt.Errorf("failed to get name of shoot cluster: unexpected result length")
	}

	name, ok := result[0][0].Interface().(string)
	if !ok {
		return "", fmt.Errorf("failed to get name of shoot cluster: unexpected type")
	}

	return name, nil
}

func (o *ShootClusterManager) createShootManifest(name string) (*unstructured.Unstructured, error) {
	tmpl, err := template.New("shootcluster").Parse(shootClusterTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to create manifest of shoot cluster: template parsing failed: %w", err)
	}

	// compute hour for the hibernation schedule of the shoot cluster, value between "00" and "23"
	hour := fmt.Sprintf("%02d", time.Now().Add(5*time.Hour).Hour())

	var manifestBuffer bytes.Buffer
	if err := tmpl.Execute(&manifestBuffer, map[string]interface{}{
		"namespace": o.namespace,
		"name":      name,
		"hour":      hour,
	}); err != nil {
		return nil, fmt.Errorf("failed to create manifest of shoot cluster: templating failed: %w", err)
	}

	manifest := unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	if err := yaml.Unmarshal(manifestBuffer.Bytes(), &manifest.Object); err != nil {
		return nil, fmt.Errorf("failed to create manifest of shoot cluster: unmarshaling failed: %w", err)
	}

	return &manifest, nil
}

func (o *ShootClusterManager) waitUntilShootClusterIsReady(ctx context.Context, gardenClient dynamic.ResourceInterface, clusterName string) error {

	err := wait.Poll(5*time.Second, 15*time.Minute, func() (done bool, err error) {
		shoot, getError := gardenClient.Get(ctx, clusterName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(getError) {
				return false, nil
			}

			return false, getError
		}

		// check conditions (maybe .lastOperation.state == "Succeeded" suffices?)
		jp := jsonpath.New("conditions")
		if err := jp.Parse("{.status.conditions}"); err != nil {
			return false, fmt.Errorf("failed to get cluster status: template parsing failed: %w", err)
		}

		result, err := jp.FindResults(shoot.Object)
		if err != nil {
			return false, fmt.Errorf("failed to get cluster status: result not found: %w", err)
		}

		if len(result) != 1 || len(result[0]) != 1 {
			return false, fmt.Errorf("failed to get cluster status: unexpected result length")
		}

		conditions, ok := result[0][0].Interface().([]interface{})
		if !ok {
			return false, fmt.Errorf("failed to get cluster status: unexpected type")
		}

		checkList := map[string]bool{
			"APIServerAvailable":      false,
			"ControlPlaneHealthy":     false,
			"EveryNodeReady":          false,
			"SystemComponentsHealthy": false,
		}

		for _, condition := range conditions {
			conditionMap, ok := condition.(map[string]interface{})
			if !ok {
				return false, fmt.Errorf("failed to get cluster status: unexpected condition format")
			}

			for checkListKey := range checkList {
				if conditionMap["type"] == checkListKey {
					if conditionMap["status"] == "True" {
						checkList[checkListKey] = true
					}
					break
				}
			}
		}

		statusOfAllConditions := true
		for checkListKey := range checkList {
			statusOfAllConditions = statusOfAllConditions && checkList[checkListKey]
		}

		return statusOfAllConditions, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait until cluster is ready: %w", err)
	}

	return nil
}

func (o *ShootClusterManager) ensureAuthDirectory() error {
	if len(o.authDirectoryPath) == 0 {
		return fmt.Errorf("no auth directory specified")
	}

	info, err := os.Stat(o.authDirectoryPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(o.authDirectoryPath, os.ModePerm); err != nil {
				return fmt.Errorf("failed to create the auth directory %s: %w", o.authDirectoryPath, err)
			}

			return nil
		} else {
			return fmt.Errorf("failed to read the auth directory %s: %w", o.authDirectoryPath, err)
		}
	}

	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", o.authDirectoryPath)
	}

	return nil
}

func (o *ShootClusterManager) writeClusterName(clusterName string) error {
	filePath := path.Join(o.authDirectoryPath, filenameForClusterName)
	if err := os.WriteFile(filePath, []byte(clusterName), os.ModePerm); err != nil {
		return fmt.Errorf("failed to write cluster name to file %s: %w", filePath, err)
	}

	return nil
}

func (o *ShootClusterManager) readClusterName() (string, error) {
	filePath := path.Join(o.authDirectoryPath, filenameForClusterName)
	clusterName, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read cluster name from file %s: %w", filePath, err)
	}

	return string(clusterName), nil
}

func (o *ShootClusterManager) writeKubeconfig(ctx context.Context, gardenClient dynamic.ResourceInterface, clusterName string) error {
	secretName := clusterName + ".kubeconfig"
	secret, err := gardenClient.Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: error reading secret %s: %w", secretName, err)
	}

	jp := jsonpath.New("kubeconfig")
	if err := jp.Parse("{.data.kubeconfig}"); err != nil {
		return fmt.Errorf("failed to get kubeconfig: template parsing failed: %w", err)
	}

	result, err := jp.FindResults(secret.Object)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: result not found: %w", err)
	}

	if len(result) != 1 || len(result[0]) != 1 {
		return fmt.Errorf("failed to get kubeconfig: unexpected result length")
	}

	kubeconfig64, ok := result[0][0].Interface().(string)
	if !ok {
		return fmt.Errorf("failed to get kubeconfig: unexpected type")
	}

	kubeconfig, err := base64.StdEncoding.DecodeString(kubeconfig64)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: base64 decoding failed")
	}

	filePath := path.Join(o.authDirectoryPath, filenameForKubeconfig)
	if err := os.WriteFile(filePath, []byte(kubeconfig), os.ModePerm); err != nil {
		return fmt.Errorf("failed to write kubeconfig to file %s: %w", filePath, err)
	}

	return nil
}

func (o *ShootClusterManager) addConfirmAnnotation(shoot *unstructured.Unstructured) {
	metadataIntf := shoot.Object["metadata"]
	metadata := metadataIntf.(map[string]interface{})
	annotationsIntf := metadata["annotations"]
	annotations := annotationsIntf.(map[string]interface{})
	annotations["confirmation.gardener.cloud/deletion"] = "true"
}

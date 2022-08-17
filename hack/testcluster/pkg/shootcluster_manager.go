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
	"sort"
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
	log                          simplelogger.Logger
	gardenClusterKubeconfigPath  string
	namespace                    string
	authDirectoryPath            string
	maxNumOfClusters             int
	numClustersStartDeleteOldest int
	durationForClusterDeletion   time.Duration
}

func NewShootClusterManager(log simplelogger.Logger, gardenClusterKubeconfigPath, namespace,
	authDirectoryPath string, maxNumOfClusters, numClustersStartDeleteOldest int, durationForClusterDeletion string) (*ShootClusterManager, error) {

	duration, err := time.ParseDuration(durationForClusterDeletion)
	if err != nil {
		return nil, err
	}

	return &ShootClusterManager{
		log:                          log,
		gardenClusterKubeconfigPath:  gardenClusterKubeconfigPath,
		namespace:                    namespace,
		authDirectoryPath:            authDirectoryPath,
		maxNumOfClusters:             maxNumOfClusters,
		numClustersStartDeleteOldest: numClustersStartDeleteOldest,
		durationForClusterDeletion:   duration,
	}, nil
}

func (o *ShootClusterManager) CreateShootCluster(ctx context.Context) error {
	gardenClient, err := o.createGardenClient()
	if err != nil {
		return err
	}

	gardenClientForShoots := gardenClient.Resource(shootGVR).Namespace(o.namespace)
	gardenClientForSecrets := gardenClient.Resource(secretGVR).Namespace(o.namespace)

	if err := o.checkAndDeleteExistingTestShoots(ctx, gardenClientForShoots); err != nil {
		return err
	}

	o.log.Logfln("generate shoot name")
	clusterName := o.generateShootName()

	o.log.Logfln("generate auth directory")
	if err := o.ensureAuthDirectory(); err != nil {
		return err
	}

	o.log.Logfln("write cluster name")
	if err := o.writeClusterName(clusterName); err != nil {
		return err
	}

	o.log.Logfln("create shoot manifest")
	shoot, err := o.createShootManifest(clusterName)
	if err != nil {
		return err
	}

	o.log.Logfln("create shoot cluster")
	_, err = gardenClientForShoots.Create(ctx, shoot, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	o.log.Logfln("wait for cluster is ready")
	if err := o.waitUntilShootClusterIsReady(ctx, gardenClientForShoots, clusterName); err != nil {
		return err
	}

	o.log.Logfln("write kubeconfig")
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

	if err = o.deleteShootCluster(ctx, gardenClientForShoots, clusterName); err != nil {
		return err
	}

	return nil
}

func (o *ShootClusterManager) deleteShootCluster(ctx context.Context, gardenClientForShoots dynamic.ResourceInterface, clusterName string) error {
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

func (o *ShootClusterManager) checkAndDeleteExistingTestShoots(ctx context.Context, gardenClientForShoots dynamic.ResourceInterface) error {
	shootList, err := gardenClientForShoots.List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	testShoots := []unstructured.Unstructured{}
	for _, shoot := range shootList.Items {
		name, err := o.getNameOfUnstructuredResource(shoot)
		if err != nil {
			return err
		}

		if o.matchesNamePattern(name) {
			testShoots = append(testShoots, shoot)
			if len(testShoots) >= o.maxNumOfClusters {
				return fmt.Errorf("the maximum number of %d test clusters in namespace %s must not be exceeded - please remove the test clusters manually",
					o.maxNumOfClusters, o.namespace)
			}
		}
	}

	remainingTestShoots, err := o.deleteOutdatedShootCluster(ctx, gardenClientForShoots, testShoots)
	if err != nil {
		return err
	}

	o.deleteOldestShootCluster(ctx, gardenClientForShoots, remainingTestShoots)

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

func (o *ShootClusterManager) getCreationTimestampOfUnstructuredResource(shoot unstructured.Unstructured) (*time.Time, error) {
	jp := jsonpath.New("creationTimestamp")
	if err := jp.Parse("{.metadata.creationTimestamp}"); err != nil {
		return nil, fmt.Errorf("failed to get creation timestamp of shoot cluster: template parsing failed: %w", err)
	}

	result, err := jp.FindResults(shoot.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to get creation timestamp of shoot cluster: result not found: %w", err)
	}

	if len(result) != 1 || len(result[0]) != 1 {
		return nil, fmt.Errorf("failed to get creation timestamp of shoot cluster: unexpected result length")
	}

	timestampString, ok := result[0][0].Interface().(string)
	if !ok {
		return nil, fmt.Errorf("failed to get creation timestamp of shoot cluster")
	}

	creationTimestamp, err := time.Parse("2006-01-02T15:04:05Z", timestampString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse creation timestamp %s: %w", timestampString, err)
	}

	return &creationTimestamp, nil
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

func (o *ShootClusterManager) deleteOutdatedShootCluster(ctx context.Context, gardenClientForShoots dynamic.ResourceInterface,
	testShoots []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
	o.log.Logfln("Starting to delete old clusters")

	remainingTestShoots := []unstructured.Unstructured{}
	for _, shoot := range testShoots {
		clusterName, err := o.getNameOfUnstructuredResource(shoot)
		if err != nil {
			return nil, err
		}

		creationTimestamp, err := o.getCreationTimestampOfUnstructuredResource(shoot)
		if err != nil {
			return nil, err
		} else {
			o.log.Logfln("test shoot cluster %s found with creation timestamp: %s", clusterName, creationTimestamp.String())
		}

		durationOfCluster := time.Now().Sub(*creationTimestamp)

		if durationOfCluster > o.durationForClusterDeletion {
			o.log.Logfln("test shoot cluster %s will be deleted because it lives for %s which is longer that the border %s",
				clusterName, durationOfCluster.String(), o.durationForClusterDeletion.String())

			if err = o.deleteShootCluster(ctx, gardenClientForShoots, clusterName); err != nil {
				o.log.Logfln("outdated test shoot cluster %s could not be deleted: %s", clusterName, err.Error())
			}
		} else {
			remainingTestShoots = append(remainingTestShoots, shoot)
		}
	}

	return remainingTestShoots, nil
}

func (o *ShootClusterManager) deleteOldestShootCluster(ctx context.Context, gardenClientForShoots dynamic.ResourceInterface,
	testShoots []unstructured.Unstructured) {

	numOfOldestClustersToDelete := len(testShoots) - o.numClustersStartDeleteOldest + 1

	if numOfOldestClustersToDelete > 0 {

		sort.SliceStable(testShoots, func(i, j int) bool {
			time1, err := o.getCreationTimestampOfUnstructuredResource(testShoots[i])
			if err != nil {
				// could not happen
				o.log.Logfln("not able to sort test shoot cluster by creation timestamp")
				panic(err)
			}
			time2, err := o.getCreationTimestampOfUnstructuredResource(testShoots[j])
			if err != nil {
				// could not happen
				o.log.Logfln("not able to sort test shoot cluster by creation timestamp")
				panic(err)
			}
			return time1.Before(*time2)
		})

		for i := 0; i < numOfOldestClustersToDelete; i++ {
			name, err := o.getNameOfUnstructuredResource(testShoots[i])
			if err != nil {
				o.log.Logfln("not able to fetch name from oldest test shoot cluster to delete")
			}

			o.log.Logfln("deleting the oldest shoot cluster %s", name)
			if err = o.deleteShootCluster(ctx, gardenClientForShoots, name); err != nil {
				o.log.Logfln("not able to trigger delete for oldest test shoot cluster")
			}
		}
	}
}

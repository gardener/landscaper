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
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gardener/landscaper/hack/testcluster/pkg/utils"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/jsonpath"
	"sigs.k8s.io/yaml"
)

//go:embed resources/shootcluster_template.yaml
var shootClusterTemplate string

const (
	// namePrefix is the name prefix of shoot clusters used for integration tests
	namePrefix = "it-"

	// prPrefix is the second part of shoot clusters used for integration tests
	prPrefix = "pr"

	ocmlibIdentifier = "o"
	cnudieIdentifier = "c"

	localStartPrefix = namePrefix + prPrefix + "0-"
	headUpdatePrefix = namePrefix + prPrefix + "1-"

	// name of the file that will be created in the auth directory, containing the name of the shoot cluster
	filenameForClusterName = "clustername"

	subresourceAdminKubeconfig  = "adminkubeconfig"
	kubeconfigExpirationSeconds = 24 * 60 * 60

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
	log                          utils.Logger
	gardenClusterKubeconfigPath  string
	namespace                    string
	authDirectoryPath            string
	maxNumOfClusters             int
	numClustersStartDeleteOldest int
	durationForClusterDeletion   time.Duration
	prID                         string
	useOCMLib                    bool
}

func NewShootClusterManager(log utils.Logger, gardenClusterKubeconfigPath, namespace,
	authDirectoryPath string, maxNumOfClusters, numClustersStartDeleteOldest int, durationForClusterDeletion, prID string,
	useOCMLib bool) (*ShootClusterManager, error) {

	log.Logfln("Create cluster manager with:")
	log.Logfln("  GardenClusterKubeconfigPath: " + gardenClusterKubeconfigPath)
	log.Logfln("  Namespace: " + namespace)
	log.Logfln("  AuthDirectoryPath: " + authDirectoryPath)
	log.Logfln("  MaxNumOfClusters: " + strconv.Itoa(maxNumOfClusters))
	log.Logfln("  NumClustersStartDeleteOldest: " + strconv.Itoa(numClustersStartDeleteOldest))
	log.Logfln("  DurationForClusterDeletion: " + durationForClusterDeletion)
	log.Logfln("  PrID: " + prID)
	log.Logfln("  UseOCMLib: " + strconv.FormatBool(useOCMLib))

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
		prID:                         prID,
		useOCMLib:                    useOCMLib,
	}, nil
}

func (o *ShootClusterManager) CreateShootCluster(ctx context.Context) error {
	gardenClient, err := o.createGardenClient()
	if err != nil {
		return err
	}

	gardenClientForShoots := gardenClient.Resource(shootGVR).Namespace(o.namespace)
	gardenClientForSecrets := gardenClient.Resource(secretGVR).Namespace(o.namespace)

	o.log.Logfln("generate shoot name")
	clusterName := o.generateShootName()

	if err := o.checkAndDeleteExistingTestShoots(ctx, gardenClientForShoots, gardenClientForSecrets, clusterName); err != nil {
		return err
	}

	o.log.Logfln("generate auth directory")
	if err := o.ensureAuthDirectory(); err != nil {
		return err
	}

	o.log.Logfln("write cluster name: %s", clusterName)
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

	o.log.Logfln("wait until test cluster is ready")
	if err := o.waitUntilShootClusterIsReady(ctx, gardenClientForShoots, clusterName); err != nil {
		return err
	}

	o.log.Logfln("create short-lived kubeconfig for test cluster")
	shootKubeconfigBase64, err := o.createShootAdminKubeconfig(ctx, gardenClientForShoots, clusterName, kubeconfigExpirationSeconds)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	o.log.Logfln("write kubeconfig for test cluster")
	filePath := path.Join(o.authDirectoryPath, filenameForKubeconfig)
	if err := o.writeKubeconfig(ctx, shootKubeconfigBase64, filePath); err != nil {
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
	gardenClientForSecrets := gardenClient.Resource(secretGVR).Namespace(o.namespace)

	clusterName, err := o.readClusterName()
	if err != nil {
		return err
	}

	if err = o.deleteShootCluster(ctx, gardenClientForShoots, gardenClientForSecrets, clusterName); err != nil {
		return err
	}

	return nil
}

func (o *ShootClusterManager) deleteShootCluster(ctx context.Context, gardenClientForShoots,
	gardenClientForSecrets dynamic.ResourceInterface, clusterName string) error {

	o.log.Logln("start deleting shoot cluster: " + clusterName)

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

	o.log.Logln("finished deleting shoot cluster: " + clusterName)

	return nil
}

func (o *ShootClusterManager) createGardenClient() (dynamic.Interface, error) {
	restConfig, err := clientcmd.BuildConfigFromFlags("", o.gardenClusterKubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read kubeconfig from %s: %w", o.gardenClusterKubeconfigPath, err)
	}

	restConfig.Burst = 60
	restConfig.QPS = 40

	gardenClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create kubernetes client from %s: %w", o.gardenClusterKubeconfigPath, err)
	}

	return gardenClient, nil
}

func (o *ShootClusterManager) checkAndDeleteExistingTestShoots(ctx context.Context,
	gardenClientForShoots, gardenClientForSecrets dynamic.ResourceInterface, newClusterName string) error {

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

	remainingTestShoots, err := o.deleteOutdatedShootCluster(ctx, gardenClientForShoots, gardenClientForSecrets, testShoots, newClusterName)
	if err != nil {
		return err
	}

	o.deleteOldestShootCluster(ctx, gardenClientForShoots, gardenClientForSecrets, remainingTestShoots)

	return nil
}

func (o *ShootClusterManager) generateShootName() string {
	var libID string
	if o.useOCMLib {
		libID = ocmlibIdentifier
	} else {
		libID = cnudieIdentifier
	}
	return namePrefix + libID + "-" + prPrefix + o.prID + "-" + strconv.Itoa(rng.Intn(9000)+1000)
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

	err := wait.PollUntilContextTimeout(ctx, 10*time.Second, 35*time.Minute, false, func(ctx context.Context) (done bool, err error) {
		o.log.Logfln("wait for cluster is ready")
		shoot, getError := gardenClient.Get(ctx, clusterName, metav1.GetOptions{})
		if getError != nil {
			if errors.IsNotFound(getError) {
				o.log.Logfln("is not found")
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
			o.log.Logfln("failed to get cluster status: result not found: %w", err)
			return false, nil
		}

		if len(result) != 1 || len(result[0]) != 1 {
			return false, fmt.Errorf("failed to get cluster status: unexpected result length")
		}

		conditions, ok := result[0][0].Interface().([]interface{})
		if !ok {
			return false, fmt.Errorf("failed to get cluster status: unexpected type")
		}
		o.log.Logfln("conditions: ", conditions)

		checkList := map[string]bool{
			"APIServerAvailable":      false,
			"ControlPlaneHealthy":     false,
			"EveryNodeReady":          false,
			"SystemComponentsHealthy": false,
		}

		o.log.Logfln("checklist: ", checkList)
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

func (o *ShootClusterManager) writeKubeconfig(_ context.Context, shootKubeconfigBase64, filePath string) error {
	kubeconfig, err := base64.StdEncoding.DecodeString(shootKubeconfigBase64)
	if err != nil {
		return fmt.Errorf("base64 decoding of lubeconfig for test cluster failed")
	}

	if err := os.WriteFile(filePath, []byte(kubeconfig), os.ModePerm); err != nil {
		return fmt.Errorf("failed to write kubeconfig for test cluster to file %s: %w", filePath, err)
	}

	return nil
}

// createShootAdminKubeconfig returns a short-lived admin kubeconfig for the specified shoot as base64 encoded string.
func (o *ShootClusterManager) createShootAdminKubeconfig(ctx context.Context, dynamicGardenClient dynamic.ResourceInterface,
	shootName string, kubeconfigExpirationSeconds int64) (string, error) {

	adminKubeconfigRequest := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "authentication.gardener.cloud/v1alpha1",
			"kind":       "AdminKubeconfigRequest",
			"metadata": map[string]interface{}{
				"namespace": o.namespace,
				"name":      shootName,
			},
			"spec": map[string]interface{}{
				"expirationSeconds": kubeconfigExpirationSeconds,
			},
		},
	}

	result, err := dynamicGardenClient.Create(ctx, &adminKubeconfigRequest, metav1.CreateOptions{}, subresourceAdminKubeconfig)
	if err != nil {
		return "", fmt.Errorf("admin kubeconfig request for test cluster failed: %w", err)
	}

	shootKubeconfigBase64, found, err := unstructured.NestedString(result.Object, "status", "kubeconfig")
	if err != nil {
		return "", fmt.Errorf("could not get kubeconfig for test cluster from result: %w", err)
	} else if !found {
		return "", fmt.Errorf("could not find kubeconfig for test cluster in result")
	}

	return shootKubeconfigBase64, nil
}

func (o *ShootClusterManager) addConfirmAnnotation(shoot *unstructured.Unstructured) {
	metadataIntf := shoot.Object["metadata"]
	metadata := metadataIntf.(map[string]interface{})
	annotationsIntf := metadata["annotations"]
	annotations := annotationsIntf.(map[string]interface{})
	annotations["confirmation.gardener.cloud/deletion"] = "true"
}

func (o *ShootClusterManager) hasDeletionTimestamp(shoot *unstructured.Unstructured) bool {
	metadataIntf := shoot.Object["metadata"]
	metadata := metadataIntf.(map[string]interface{})
	return metadata["deletionTimestamp"] != nil
}

func (o *ShootClusterManager) deleteOutdatedShootCluster(ctx context.Context, gardenClientForShoots,
	gardenClientForSecrets dynamic.ResourceInterface, testShoots []unstructured.Unstructured, newClusterName string) ([]unstructured.Unstructured, error) {
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

		durationOfCluster := time.Since(*creationTimestamp)

		hasDeletionTimestamp := o.hasDeletionTimestamp(&shoot)

		clusterNamePrefix := clusterName[:len(clusterName)-4]
		newClusterNamePrefix := newClusterName[:len(newClusterName)-4]

		libID := clusterName[3:4]
		newLibID := newClusterName[3:4]

		if hasDeletionTimestamp {
			continue
		} else if !strings.HasPrefix(clusterName, localStartPrefix) && !strings.HasPrefix(clusterName, headUpdatePrefix) &&
			clusterNamePrefix == newClusterNamePrefix && libID == newLibID {
			o.log.Logfln("test shoot cluster %s will be deleted because a new test for the same PR is triggered", clusterName)

			if err = o.deleteShootCluster(ctx, gardenClientForShoots, gardenClientForSecrets, clusterName); err != nil {
				o.log.Logfln("outdated test shoot cluster %s could not be deleted: %s", clusterName, err.Error())
			}
		} else if durationOfCluster > o.durationForClusterDeletion {
			o.log.Logfln("test shoot cluster %s will be deleted because it lives for %s which is longer than the border %s",
				clusterName, durationOfCluster.String(), o.durationForClusterDeletion.String())

			if err = o.deleteShootCluster(ctx, gardenClientForShoots, gardenClientForSecrets, clusterName); err != nil {
				o.log.Logfln("outdated test shoot cluster %s could not be deleted: %s", clusterName, err.Error())
			}
		} else {
			remainingTestShoots = append(remainingTestShoots, shoot)
		}
	}

	return remainingTestShoots, nil
}

func (o *ShootClusterManager) deleteOldestShootCluster(ctx context.Context, gardenClientForShoots, gardenClientForSecrets dynamic.ResourceInterface,
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
			if err = o.deleteShootCluster(ctx, gardenClientForShoots, gardenClientForSecrets, name); err != nil {
				o.log.Logfln("not able to trigger delete for oldest test shoot cluster")
			}
		}
	}
}

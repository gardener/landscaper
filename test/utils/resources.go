// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/golang/mock/gomock"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/gardener/landscaper/apis/core/install"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/container"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	k8smock "github.com/gardener/landscaper/pkg/utils/kubernetes/mock"
)

// TestInstallationConfig defines a installation configuration which can be used to create
// a test environment with a installation, a blueprint and a operation.
type TestInstallationConfig struct {
	// +optional
	MockClient *k8smock.MockClient
	// Defines the installation that should be used to create a blueprint and operations
	// If it is not defined a default one is created with the given name and namespace
	// +optional
	Installation *lsv1alpha1.Installation

	// Configures the default created installation
	InstallationName             string
	InstallationNamespace        string
	RemoteBlueprintComponentName string
	RemoteBlueprintResourceName  string
	RemoteBlueprintVersion       string
	RemoteBlueprintBaseURL       string

	BlueprintContentPath string
	// BlueprintFilePath defines the filepath to the blueprint definition.
	// Will be defaulted to <BlueprintContentPath>/blueprint.yaml if not defined.
	BlueprintFilePath string
}

// CreateTestInstallationResources creates a test environment with a installation, a blueprint and a operation.
// Should only be used for root installation as other installations may be created during runtime.
func CreateTestInstallationResources(op lsoperation.Interface, cfg TestInstallationConfig) (*lsv1alpha1.Installation, *installations.Installation, *blueprints.Blueprint, *installations.Operation) {
	// apply defaults
	if len(cfg.BlueprintFilePath) == 0 {
		cfg.BlueprintFilePath = filepath.Join(cfg.BlueprintContentPath, "blueprint.yaml")
	}

	if cfg.MockClient != nil {
		cfg.MockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
			func(ctx context.Context, instList *lsv1alpha1.InstallationList, _ ...interface{}) error {
				*instList = lsv1alpha1.InstallationList{}
				return nil
			})
	}

	rootInst := cfg.Installation
	if rootInst == nil {
		rootInst = &lsv1alpha1.Installation{}
		rootInst.Name = cfg.InstallationName
		rootInst.Namespace = cfg.InstallationNamespace
		rootInst.Spec.ComponentDescriptor = LocalRemoteComponentDescriptorRef(cfg.RemoteBlueprintComponentName, cfg.RemoteBlueprintVersion, cfg.RemoteBlueprintBaseURL)
		rootInst.Spec.Blueprint = LocalRemoteBlueprintRef(cfg.RemoteBlueprintResourceName)
	}

	rootBlueprint := CreateBlueprintFromFile(cfg.BlueprintFilePath, cfg.BlueprintContentPath)

	rootIntInst, err := installations.New(rootInst, rootBlueprint)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	rootInstOp, err := installations.NewInstallationOperationFromOperation(context.TODO(), op, rootIntInst)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	return rootInst, rootIntInst, rootBlueprint, rootInstOp
}

// LocalRemoteComponentDescriptorRef creates a new default local remote component descriptor reference
func LocalRemoteComponentDescriptorRef(componentName, version, baseURL string) *lsv1alpha1.ComponentDescriptorDefinition {
	return &lsv1alpha1.ComponentDescriptorDefinition{
		Reference: &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: &cdv2.RepositoryContext{
				Type:    "local",
				BaseURL: baseURL,
			},
			ComponentName: componentName,
			Version:       version,
		},
	}
}

// LocalRemoteBlueprintRef creates a new default local remote blueprint reference
func LocalRemoteBlueprintRef(resourceName string) lsv1alpha1.BlueprintDefinition {
	return lsv1alpha1.BlueprintDefinition{
		Reference: &lsv1alpha1.RemoteBlueprintReference{
			ResourceName: resourceName,
		},
	}
}

// ReadResourceFromFile reads a file and parses it to the given object
func ReadResourceFromFile(obj runtime.Object, testfile string) error {
	data, err := ioutil.ReadFile(testfile)
	if err != nil {
		return err
	}

	landscaperScheme := runtime.NewScheme()
	install.Install(landscaperScheme)
	decoder := serializer.NewCodecFactory(landscaperScheme).UniversalDecoder()
	if _, _, err := decoder.Decode(data, nil, obj); err != nil {
		return err
	}
	return nil
}

// ReadBlueprintFromFile reads a file and parses it to a Blueprint
func ReadBlueprintFromFile(testfile string) (*lsv1alpha1.Blueprint, error) {
	data, err := ioutil.ReadFile(testfile)
	if err != nil {
		return nil, err
	}

	landscaperScheme := runtime.NewScheme()
	install.Install(landscaperScheme)
	decoder := serializer.NewCodecFactory(landscaperScheme).UniversalDecoder()

	component := &lsv1alpha1.Blueprint{}
	if _, _, err := decoder.Decode(data, nil, component); err != nil {
		return nil, err
	}
	return component, nil
}

// CreateBlueprintFromFile reads a blueprint from the given file and creates a internal blueprint object.
func CreateBlueprintFromFile(filePath, contentPath string) *blueprints.Blueprint {
	def, err := ReadBlueprintFromFile(filePath)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	contentPath, err = filepath.Abs(contentPath)
	gomega.Expect(err).To(gomega.Succeed())

	fs, err := projectionfs.New(osfs.New(), contentPath)
	gomega.Expect(err).To(gomega.Succeed())
	return blueprints.New(def, fs)
}

// CreateOrUpdateTarget creates or updates a target with specific name, namespace and type
func CreateOrUpdateTarget(ctx context.Context, client client.Client, namespace, name, ttype string, config interface{}) (*lsv1alpha1.Target, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	target := &lsv1alpha1.Target{}
	target.Name = name
	target.Namespace = namespace

	_, err = controllerutil.CreateOrUpdate(ctx, client, target, func() error {
		target.Spec.Type = lsv1alpha1.TargetType(ttype)
		target.Spec.Configuration = lsv1alpha1.NewAnyJSON(data)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return target, err
}

// CreateKubernetesTarget creates a new target of type kubernetes
func CreateKubernetesTarget(namespace, name string, restConfig *rest.Config) (*lsv1alpha1.Target, error) {
	data, err := kutil.GenerateKubeconfigBytes(restConfig)
	if err != nil {
		return nil, err
	}

	config := lsv1alpha1.KubernetesClusterTargetConfig{
		Kubeconfig: string(data),
	}
	data, err = json.Marshal(config)
	if err != nil {
		return nil, err
	}

	target := &lsv1alpha1.Target{}
	target.Name = name
	target.Namespace = namespace

	target.Spec.Type = lsv1alpha1.KubernetesClusterTargetType
	target.Spec.Configuration = lsv1alpha1.NewAnyJSON(data)

	return target, nil
}

// CreateInternalKubernetesTarget creates a new target of type kubernetes
// whereas the hostname of the cluster will be set to the cluster internal host.
// It is expected that the controller runs inside the same cluster where it also deploys to.
func CreateInternalKubernetesTarget(ctx context.Context, kubeClient client.Client, namespace, name string, restConfig *rest.Config, internal bool) (*lsv1alpha1.Target, error) {
	if internal {
		oldHost := restConfig.Host
		defer func() {
			restConfig.Host = oldHost
		}()

		// get the kubernetes internal port of the kubernetes svc
		// it is expected that the kubernetes svc is in the default namespace with the name "kubernetes"
		const (
			kubernetesSvcName      = "kubernetes"
			kubernetesSvcNamespace = "default"
		)
		svc := &corev1.Service{}
		if err := kubeClient.Get(ctx, kutil.ObjectKey(kubernetesSvcName, kubernetesSvcNamespace), svc); err != nil {
			return nil, err
		}
		if len(svc.Spec.Ports) != 1 {
			return nil, fmt.Errorf("unexpected number of ports of the kubernetes service %d", len(svc.Spec.Ports))
		}
		u, err := url.Parse(oldHost)
		if err != nil {
			return nil, err
		}
		u.Host = fmt.Sprintf("%s.%s:%d", kubernetesSvcName, kubernetesSvcNamespace, svc.Spec.Ports[0].Port)
		restConfig.Host = u.String()
	}
	return CreateKubernetesTarget(namespace, name, restConfig)
}

// BuildContainerDeployItem builds a new deploy item of type container.
func BuildContainerDeployItem(configuration *containerv1alpha1.ProviderConfiguration) *lsv1alpha1.DeployItem {
	di, err := container.NewDeployItemBuilder().
		ProviderConfig(configuration).
		Build()
	ExpectNoErrorWithOffset(1, err)
	return di
}

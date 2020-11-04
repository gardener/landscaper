// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"io/ioutil"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/golang/mock/gomock"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/gardener/landscaper/pkg/apis/core/install"
	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
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
		rootInst.Spec.Blueprint = LocalRemoteBlueprintRef(cfg.RemoteBlueprintComponentName,
			cfg.RemoteBlueprintResourceName, cfg.RemoteBlueprintVersion, cfg.RemoteBlueprintBaseURL)
	}

	rootBlueprint := CreateBlueprintFromFile(cfg.BlueprintFilePath, cfg.BlueprintContentPath)

	rootIntInst, err := installations.New(rootInst, rootBlueprint)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	rootInstOp, err := installations.NewInstallationOperationFromOperation(context.TODO(), op, rootIntInst)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	return rootInst, rootIntInst, rootBlueprint, rootInstOp
}

// LocalRemoteBlueprintRef creates a new default local remote blueprint reference
func LocalRemoteBlueprintRef(componentName, resourceName, version, baseUrl string) lsv1alpha1.BlueprintDefinition {
	return lsv1alpha1.BlueprintDefinition{
		Reference: &lsv1alpha1.RemoteBlueprintReference{
			RepositoryContext: &cdv2.RepositoryContext{
				Type:    "local",
				BaseURL: baseUrl,
			},
			VersionedResourceReference: lsv1alpha1.VersionedResourceReference{
				ResourceReference: lsv1alpha1.ResourceReference{
					ComponentName: componentName,
					ResourceName:  resourceName,
				},
				Version: version,
			},
		},
	}
}

// ReadInstallationFromFile reads a file and parses it to a Installation
func ReadInstallationFromFile(testfile string) (*lsv1alpha1.Installation, error) {
	data, err := ioutil.ReadFile(testfile)
	if err != nil {
		return nil, err
	}

	landscaperScheme := runtime.NewScheme()
	install.Install(landscaperScheme)
	decoder := serializer.NewCodecFactory(landscaperScheme).UniversalDecoder()

	component := &lsv1alpha1.Installation{}
	if _, _, err := decoder.Decode(data, nil, component); err != nil {
		return nil, err
	}
	return component, nil
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
	blue, err := blueprints.New(def, fs)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	return blue
}

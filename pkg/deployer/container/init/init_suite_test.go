// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/golang/mock/gomock"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/container"
	mock_client "github.com/gardener/landscaper/controller-utils/pkg/kubernetes/mock"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/test/utils"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installations Imports Test Suite")
}

var _ = Describe("Constructor", func() {

	var (
		ctrl       *gomock.Controller
		fakeClient *mock_client.MockClient
		fs         vfs.FileSystem
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		fakeClient = mock_client.NewMockClient(ctrl)
		// set default env vars
		Expect(os.Setenv(container.ConfigurationPathName, container.ConfigurationPath)).To(Succeed())
		Expect(os.Setenv(container.ImportsPathName, container.ImportsPath)).To(Succeed())
		Expect(os.Setenv(container.ExportsPathName, container.ExportsPath)).To(Succeed())
		Expect(os.Setenv(container.StatePathName, container.StatePath)).To(Succeed())
		Expect(os.Setenv(container.ContentPathName, container.ContentPath)).To(Succeed())
		Expect(os.Setenv(container.ComponentDescriptorPathName, container.ComponentDescriptorPath)).To(Succeed())
		Expect(os.Setenv(container.TargetPathName, container.TargetPath)).To(Succeed())

		utils.ExpectNoError(os.Setenv(container.DeployItemName, "dummy"))
		utils.ExpectNoError(os.Setenv(container.DeployItemNamespaceName, "val"))
		utils.ExpectNoError(os.Setenv(container.PodNamespaceName, "default"))

		fs = memoryfs.New()
		Expect(fs.MkdirAll(container.TargetInitDir, os.ModePerm)).To(Succeed())
		Expect(vfs.WriteFile(fs, filepath.Join(container.TargetInitDir, container.TargetFileName), []byte(`{"foo": "bar"}`), os.ModePerm)).To(Succeed()) // fake target file
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should fetch import values from DeployItem's configuration and write them to 'import.json'", func() {
		ctx := context.Background()
		defer ctx.Done()
		fakeClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
		opts := &options{}
		opts.Complete()

		file, err := os.ReadFile("./testdata/00-di-simple.yaml")
		Expect(err).ToNot(HaveOccurred())
		Expect(fs.MkdirAll(filepath.Dir(container.ConfigurationPath), os.ModePerm)).To(Succeed())
		Expect(vfs.WriteFile(fs, container.ConfigurationPath, file, os.ModePerm)).To(Succeed())
		Expect(run(ctx, opts, fakeClient, fs)).To(Succeed())

		dataBytes, err := vfs.ReadFile(fs, container.ImportsPath)
		Expect(err).ToNot(HaveOccurred())
		var data interface{}
		Expect(json.Unmarshal(dataBytes, &data)).To(Succeed())
		Expect(data).To(HaveKeyWithValue("key", "val1"))
	})

	It("should fetch a blueprint from a DeployItem's configuration and write them to the content path", func() {
		ctx := context.Background()
		defer ctx.Done()
		fakeClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
		opts := &options{}
		opts.Complete()

		file, err := os.ReadFile("./testdata/01-di-blueprint.yaml")
		Expect(err).ToNot(HaveOccurred())
		Expect(fs.MkdirAll(filepath.Dir(container.ConfigurationPath), os.ModePerm)).To(Succeed())
		Expect(vfs.WriteFile(fs, container.ConfigurationPath, file, os.ModePerm)).To(Succeed())
		Expect(run(ctx, opts, fakeClient, fs)).To(Succeed())

		info, err := vfs.ReadDir(fs, container.ContentPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(info).To(HaveLen(1))
		Expect(info[0].Name()).To(Equal(v1alpha1.BlueprintFileName))
	})

	It("should fetch an inline blueprint from a DeployItem's configuration and write them to the content path", func() {
		ctx := context.Background()
		defer ctx.Done()
		fakeClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
		opts := &options{}
		opts.Complete()

		file, err := os.ReadFile("./testdata/02-di-inline-blueprint.yaml")
		Expect(err).ToNot(HaveOccurred())
		Expect(fs.MkdirAll(filepath.Dir(container.ConfigurationPath), os.ModePerm)).To(Succeed())
		Expect(vfs.WriteFile(fs, container.ConfigurationPath, file, os.ModePerm)).To(Succeed())
		Expect(run(ctx, opts, fakeClient, fs)).To(Succeed())

		info, err := vfs.ReadDir(fs, container.ContentPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(info).To(HaveLen(1))
		Expect(info[0].Name()).To(Equal(v1alpha1.BlueprintFileName))
	})

	It("should fetch component descriptor from DeployItem's configuration and write them to the component descriptor path", func() {
		ctx := context.Background()
		defer ctx.Done()
		fakeClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
		opts := &options{}
		opts.Complete()

		file, err := os.ReadFile("./testdata/01-di-blueprint.yaml")
		Expect(err).ToNot(HaveOccurred())
		Expect(fs.MkdirAll(filepath.Dir(container.ConfigurationPath), os.ModePerm)).To(Succeed())
		Expect(vfs.WriteFile(fs, container.ConfigurationPath, file, os.ModePerm)).To(Succeed())
		Expect(run(ctx, opts, fakeClient, fs)).To(Succeed())

		data, err := vfs.ReadFile(fs, container.ComponentDescriptorPath)
		Expect(err).ToNot(HaveOccurred())

		cd := &types.ComponentDescriptorList{}
		Expect(codec.Decode(data, cd)).To(Succeed())
		Expect(cd.Components).To(HaveLen(1))
	})

	It("should fetch an inline component descriptor from DeployItem's configuration and write them to the component descriptor path", func() {
		ctx := context.Background()
		defer ctx.Done()
		fakeClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
		opts := &options{}
		opts.Complete()

		file, err := os.ReadFile("./testdata/03-di-inline-cd.yaml")
		Expect(err).ToNot(HaveOccurred())
		Expect(fs.MkdirAll(filepath.Dir(container.ConfigurationPath), os.ModePerm)).To(Succeed())
		Expect(vfs.WriteFile(fs, container.ConfigurationPath, file, os.ModePerm)).To(Succeed())
		Expect(run(ctx, opts, fakeClient, fs)).To(Succeed())

		data, err := vfs.ReadFile(fs, container.ComponentDescriptorPath)
		Expect(err).ToNot(HaveOccurred())

		cd := &types.ComponentDescriptorList{}
		Expect(codec.Decode(data, cd)).To(Succeed())
		Expect(cd.Components).To(HaveLen(1))
	})

	It("should fetch an inline blueprint from a DeployItem's configuration with no Component Descriptor at all and write them to the content path", func() {
		ctx := context.Background()
		defer ctx.Done()
		fakeClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
		opts := &options{}
		opts.Complete()

		file, err := os.ReadFile("./testdata/04-di-inline-bp-no-cd.yaml")
		Expect(err).ToNot(HaveOccurred())
		Expect(fs.MkdirAll(filepath.Dir(container.ConfigurationPath), os.ModePerm)).To(Succeed())
		Expect(vfs.WriteFile(fs, container.ConfigurationPath, file, os.ModePerm)).To(Succeed())
		Expect(run(ctx, opts, fakeClient, fs)).To(Succeed())

		info, err := vfs.ReadDir(fs, container.ContentPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(info).To(HaveLen(1))
		Expect(info[0].Name()).To(Equal(v1alpha1.BlueprintFileName))
	})

	It("should fetch an inline blueprint and an inline Component Descriptor from a DeployItem's configuration and write them to the content path", func() {
		ctx := context.Background()
		defer ctx.Done()
		fakeClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
		opts := &options{}
		opts.Complete()

		file, err := os.ReadFile("./testdata/05-di-inline-bp-inline-cd.yaml")
		Expect(err).ToNot(HaveOccurred())
		Expect(fs.MkdirAll(filepath.Dir(container.ConfigurationPath), os.ModePerm)).To(Succeed())
		Expect(vfs.WriteFile(fs, container.ConfigurationPath, file, os.ModePerm)).To(Succeed())
		Expect(run(ctx, opts, fakeClient, fs)).To(Succeed())

		info, err := vfs.ReadDir(fs, container.ContentPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(info).To(HaveLen(1))
		Expect(info[0].Name()).To(Equal(v1alpha1.BlueprintFileName))

		data, err := vfs.ReadFile(fs, container.ComponentDescriptorPath)
		Expect(err).ToNot(HaveOccurred())
		cd := &types.ComponentDescriptorList{}
		Expect(codec.Decode(data, cd)).To(Succeed())
		Expect(cd.Components).To(HaveLen(1))
	})

	It("should ignore if the registry secrets path does not exist", func() {
		ctx := context.Background()
		defer ctx.Done()
		fakeClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
		opts := &options{}
		opts.Complete()
		opts.RegistrySecretBasePath = "/unexisting/path"

		file, err := os.ReadFile("./testdata/01-di-blueprint.yaml")
		Expect(err).ToNot(HaveOccurred())
		Expect(fs.MkdirAll(filepath.Dir(container.ConfigurationPath), os.ModePerm)).To(Succeed())
		Expect(vfs.WriteFile(fs, container.ConfigurationPath, file, os.ModePerm)).To(Succeed())
		Expect(run(ctx, opts, fakeClient, fs)).To(Succeed())

		info, err := vfs.ReadDir(fs, container.ContentPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(info).To(HaveLen(1))
		Expect(info[0].Name()).To(Equal(v1alpha1.BlueprintFileName))
	})
})

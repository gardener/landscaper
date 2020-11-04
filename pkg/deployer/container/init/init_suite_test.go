// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	logtesting "github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/gardener/landscaper/pkg/apis/deployer/container"
	blueprintsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
	mock_client "github.com/gardener/landscaper/pkg/utils/kubernetes/mock"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installations Imports Test Suite")
}

var _ = Describe("Constructor", func() {

	var (
		ctrl       *gomock.Controller
		fakeClient *mock_client.MockClient
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
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should fetch import values from DeployItem's configuration and write them to 'import.json'", func() {
		ctx := context.Background()
		defer ctx.Done()
		fakeClient.EXPECT().List(ctx, gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
		opts := &options{}
		opts.Complete(ctx)
		memFs := memoryfs.New()

		file, err := ioutil.ReadFile("./testdata/00-di-simple.yaml")
		Expect(err).ToNot(HaveOccurred())
		Expect(memFs.MkdirAll(filepath.Dir(container.ConfigurationPath), os.ModePerm)).To(Succeed())
		Expect(vfs.WriteFile(memFs, container.ConfigurationPath, file, os.ModePerm)).To(Succeed())
		Expect(run(ctx, logtesting.NullLogger{}, opts, fakeClient, memFs)).To(Succeed())

		dataBytes, err := vfs.ReadFile(memFs, container.ImportsPath)
		Expect(err).ToNot(HaveOccurred())
		var data interface{}
		Expect(json.Unmarshal(dataBytes, &data)).To(Succeed())
		Expect(data).To(HaveKeyWithValue("key", "val1"))
	})

	It("should fetch blueprint values from DeployItem's configuration and write them to the content path", func() {
		ctx := context.Background()
		defer ctx.Done()
		fakeClient.EXPECT().List(ctx, gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
		opts := &options{}
		opts.Complete(ctx)
		memFs := memoryfs.New()

		file, err := ioutil.ReadFile("./testdata/01-di-blueprint.yaml")
		Expect(err).ToNot(HaveOccurred())
		Expect(memFs.MkdirAll(filepath.Dir(container.ConfigurationPath), os.ModePerm)).To(Succeed())
		Expect(vfs.WriteFile(memFs, container.ConfigurationPath, file, os.ModePerm)).To(Succeed())
		Expect(run(ctx, logtesting.NullLogger{}, opts, fakeClient, memFs)).To(Succeed())

		info, err := vfs.ReadDir(memFs, container.ContentPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(info).To(HaveLen(1))
		Expect(info[0].Name()).To(Equal(blueprintsregistry.BlueprintFileName))
	})
})

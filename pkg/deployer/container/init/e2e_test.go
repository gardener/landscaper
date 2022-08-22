// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/container"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/deployer/container/state"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var (
	testenv     *envtest.Environment
	projectRoot = filepath.Join("../../../../")
)

var _ = BeforeSuite(func() {
	var err error
	testenv, err = envtest.New(projectRoot)
	Expect(err).ToNot(HaveOccurred())

	_, err = testenv.Start()
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(testenv.Stop()).ToNot(HaveOccurred())
})

var _ = Describe("Init e2e", func() {

	var testState *envtest.State

	BeforeEach(func() {
		var err error
		testState, err = testenv.InitState(context.TODO())
		Expect(err).ToNot(HaveOccurred())

		// set default env vars
		Expect(os.Setenv(container.ConfigurationPathName, container.ConfigurationPath)).To(Succeed())
		Expect(os.Setenv(container.ImportsPathName, container.ImportsPath)).To(Succeed())
		Expect(os.Setenv(container.ExportsPathName, container.ExportsPath)).To(Succeed())
		Expect(os.Setenv(container.StatePathName, container.StatePath)).To(Succeed())
		Expect(os.Setenv(container.ContentPathName, container.ContentPath)).To(Succeed())
		Expect(os.Setenv(container.ComponentDescriptorPathName, container.ComponentDescriptorPath)).To(Succeed())
	})

	AfterEach(func() {
		Expect(testenv.CleanupState(context.TODO(), testState)).To(Succeed())
	})

	It("should restore a backup", func() {
		ctx := logging.NewContextWithDiscard(context.Background())
		defer ctx.Done()
		var (
			fs           = memoryfs.New()
			resFs        = memoryfs.New()
			testFilePath = path.Join(container.StatePath, "my-file")
			testData     = []byte("text")
			di           = lsv1alpha1.ObjectReference{
				Name:      "testname",
				Namespace: "testns",
			}
		)
		utils.ExpectNoError(os.Setenv(container.DeployItemName, di.Name))
		utils.ExpectNoError(os.Setenv(container.DeployItemNamespaceName, di.Namespace))
		utils.ExpectNoError(os.Setenv(container.PodNamespaceName, testState.Namespace))

		utils.ExpectNoError(fs.MkdirAll(container.StatePath, os.ModePerm))
		utils.ExpectNoError(vfs.WriteFile(fs, testFilePath, testData, os.ModePerm))

		s := state.New(testenv.Client, testState.Namespace, di, container.StatePath).WithFs(fs)

		utils.ExpectNoError(s.Backup(ctx))

		file, err := ioutil.ReadFile("./testdata/00-di-simple.yaml")
		Expect(err).ToNot(HaveOccurred())
		Expect(resFs.MkdirAll(filepath.Dir(container.ConfigurationPath), os.ModePerm)).To(Succeed())
		Expect(vfs.WriteFile(resFs, container.ConfigurationPath, file, os.ModePerm)).To(Succeed())

		opts := &options{}
		opts.Complete()
		Expect(run(ctx, opts, testenv.Client, resFs)).To(Succeed())

		resData, err := vfs.ReadFile(resFs, testFilePath)
		utils.ExpectNoError(err)
		Expect(resData).To(Equal(testData))
	})

})

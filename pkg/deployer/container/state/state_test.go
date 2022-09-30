// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package state_test

import (
	"context"
	"os"
	"path"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/deployer/container/state"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Container Deployer State", func() {

	var testState *envtest.State

	BeforeEach(func() {
		var err error
		testState, err = testenv.InitState(context.TODO())
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(testenv.CleanupState(context.TODO(), testState)).To(Succeed())
	})

	It("should save a file and restore it", func() {
		ctx := logging.NewContextWithDiscard(context.Background())
		defer ctx.Done()
		var (
			fs           = memoryfs.New()
			resFs        = memoryfs.New()
			testDir      = "/mystate"
			testFilePath = path.Join(testDir, "my-file")
			testData     = []byte("text")
		)

		utils.ExpectNoError(fs.MkdirAll(testDir, os.ModePerm))
		utils.ExpectNoError(vfs.WriteFile(fs, testFilePath, testData, os.ModePerm))

		s := state.New(testenv.Client, testState.Namespace, lsv1alpha1.ObjectReference{
			Name:      "testname",
			Namespace: "testns",
		}, testDir).WithFs(fs)

		utils.ExpectNoError(s.Backup(ctx))
		s.WithFs(resFs)
		utils.ExpectNoError(s.Restore(ctx))

		resData, err := vfs.ReadFile(resFs, testFilePath)
		utils.ExpectNoError(err)
		Expect(resData).To(Equal(testData))
	})

	It("should garbage collect old state secrets", func() {
		ctx := logging.NewContextWithDiscard(context.Background())
		defer ctx.Done()
		var (
			fs           = memoryfs.New()
			testDir      = "/mystate"
			testFilePath = path.Join(testDir, "my-file")
			testData     = []byte("text")
		)

		utils.ExpectNoError(fs.MkdirAll(testDir, os.ModePerm))
		utils.ExpectNoError(vfs.WriteFile(fs, testFilePath, testData, os.ModePerm))

		s := state.New(testenv.Client, testState.Namespace, lsv1alpha1.ObjectReference{
			Name:      "testname",
			Namespace: "testns",
		}, testDir).WithFs(fs)

		utils.ExpectNoError(s.Backup(ctx))
		// expect that there is exactly one state secret
		secretList := &corev1.SecretList{}
		utils.ExpectNoError(testenv.Client.List(ctx, secretList, client.InNamespace(testState.Namespace)))
		Expect(secretList.Items).To(HaveLen(1))

		utils.ExpectNoError(s.WithFs(memoryfs.New()).Restore(ctx))
		// expect that there is exactly one state secret
		secretList = &corev1.SecretList{}
		utils.ExpectNoError(testenv.Client.List(ctx, secretList, client.InNamespace(testState.Namespace)))
		Expect(secretList.Items).To(HaveLen(1))

		utils.ExpectNoError(s.WithFs(fs).Backup(ctx))
		// expect that there are now 2 state secrets until the next gc run
		secretList = &corev1.SecretList{}
		utils.ExpectNoError(testenv.Client.List(ctx, secretList, client.InNamespace(testState.Namespace)))
		Expect(secretList.Items).To(HaveLen(2))

		utils.ExpectNoError(s.WithFs(memoryfs.New()).Restore(ctx))
		// expect that there is exactly one state secret
		secretList = &corev1.SecretList{}
		utils.ExpectNoError(testenv.Client.List(ctx, secretList, client.InNamespace(testState.Namespace)))
		Expect(secretList.Items).To(HaveLen(1))
	})

	It("should cleanup the state", func() {
		ctx := logging.NewContextWithDiscard(context.Background())
		defer ctx.Done()
		var (
			fs           = memoryfs.New()
			testDir      = "/mystate"
			testFilePath = path.Join(testDir, "my-file")
			testData     = []byte("text")
		)

		utils.ExpectNoError(fs.MkdirAll(testDir, os.ModePerm))
		utils.ExpectNoError(vfs.WriteFile(fs, testFilePath, testData, os.ModePerm))

		s := state.New(testenv.Client, testState.Namespace, lsv1alpha1.ObjectReference{
			Name:      "testname",
			Namespace: "testns",
		}, testDir).WithFs(fs)

		utils.ExpectNoError(s.Backup(ctx))

		err := state.CleanupState(ctx, logging.Discard(), testenv.Client, testState.Namespace, lsv1alpha1.ObjectReference{
			Name:      "testname",
			Namespace: "testns",
		})
		utils.ExpectNoError(err)

		secretList := &corev1.SecretList{}
		utils.ExpectNoError(testenv.Client.List(ctx, secretList, client.InNamespace(testState.Namespace)))
		Expect(secretList.Items).To(HaveLen(0))
	})

})

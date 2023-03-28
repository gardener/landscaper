// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/cnudie"
	installationsctl "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	lsutils "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/simplelogger"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Delete", func() {

	Context("reconciler", func() {
		var (
			op   *lsoperation.Operation
			ctrl reconcile.Reconciler

			state *envtest.State
		)

		BeforeEach(func() {
			var err error
			registryAccess, err := cnudie.NewLocalRegistryAccess("./testdata")
			Expect(err).ToNot(HaveOccurred())
			op = lsoperation.NewOperation(testenv.Client, api.LandscaperScheme, record.NewFakeRecorder(1024)).SetComponentsRegistry(registryAccess)

			ctrl = installationsctl.NewTestActuator(*op, logging.Discard(), clock.RealClock{}, &config.LandscaperConfiguration{
				Registry: config.RegistryConfiguration{
					Local: &config.LocalRegistryConfiguration{
						RootPath: "./testdata",
					},
				},
			})
		})

		AfterEach(func() {
			if state != nil {
				ctx := context.Background()
				defer ctx.Done()
				Expect(testenv.CleanupState(ctx, state)).ToNot(HaveOccurred())
				state = nil
			}
		})

		It("should propagate the delete-without-uninstall annotation to an execution", func() {
			ctx := context.Background()

			var err error
			state, err = testenv.InitResources(ctx, "./testdata/state/test3")
			Expect(err).ToNot(HaveOccurred())
			Expect(testutils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())

			inst := &lsv1alpha1.Installation{}
			inst.Name = "root"
			inst.Namespace = state.Namespace
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
			metav1.SetMetaDataAnnotation(&inst.ObjectMeta, lsv1alpha1.DeleteWithoutUninstallAnnotation, "true")
			testutils.ExpectNoError(testenv.Client.Update(ctx, inst))
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
			testutils.ExpectNoError(testenv.Client.Delete(ctx, inst))

			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))
			_ = testutils.ShouldNotReconcile(ctx, ctrl, testutils.RequestFromObject(inst))

			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
			Expect(lsutils.IsInstallationPhase(inst, lsv1alpha1.InstallationPhases.Deleting)).To(BeTrue())
			Expect(lsutils.IsInstallationJobIDsIdentical(inst)).To(BeFalse())

			exec := &lsv1alpha1.Execution{}
			exec.Name = "root"
			exec.Namespace = state.Namespace
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))
			Expect(exec.DeletionTimestamp).ToNot(BeNil())
			ann, ok := exec.Annotations[lsv1alpha1.DeleteWithoutUninstallAnnotation]
			Expect(ok).To(BeTrue())
			Expect(ann).To(Equal("true"))

		})

		It("should fail if a successor has failed", func() {
			ctx := context.Background()

			var err error
			state, err = testenv.InitResources(ctx, "./testdata/state/test8")
			Expect(err).ToNot(HaveOccurred())
			Expect(testutils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())

			inst := state.Installations[state.Namespace+"/a"]
			_ = testutils.ShouldNotReconcile(ctx, ctrl, testutils.RequestFromObject(inst))

			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
			Expect(lsutils.IsInstallationPhase(inst, lsv1alpha1.InstallationPhases.DeleteFailed)).To(BeTrue())
			Expect(lsutils.IsInstallationJobIDsIdentical(inst)).To(BeTrue())

			inst = state.Installations[state.Namespace+"/root"]
			_ = testutils.ShouldNotReconcile(ctx, ctrl, testutils.RequestFromObject(inst))

			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
			Expect(lsutils.IsInstallationPhase(inst, lsv1alpha1.InstallationPhases.DeleteFailed)).To(BeTrue())
			Expect(lsutils.IsInstallationJobIDsIdentical(inst)).To(BeTrue())
		})
	})

	Context("Controller", func() {

		var (
			state  *envtest.State
			mgr    manager.Manager
			ctx    context.Context
			cancel context.CancelFunc
		)

		BeforeEach(func() {
			ctx, cancel = context.WithCancel(context.Background())
			var err error
			mgr, err = manager.New(testenv.Env.Config, manager.Options{
				Scheme:             api.LandscaperScheme,
				MetricsBindAddress: "0",
				NewClient:          lsutils.NewUncachedClient,
			})
			Expect(err).ToNot(HaveOccurred())

			state, err = testenv.InitState(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(testutils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())

			Expect(installationsctl.AddControllerToManager(logging.Wrap(simplelogger.NewIOLogger(GinkgoWriter)), mgr, &config.LandscaperConfiguration{})).To(Succeed())
			go func() {
				Expect(mgr.Start(ctx)).To(Succeed())
			}()
			Expect(mgr.GetCache().WaitForCacheSync(ctx)).To(BeTrue())
		})

		AfterEach(func() {
			cancel()
		})

		It("should not delete if another installation still imports a exported value", func() {
			var err error
			state, err = testenv.InitResources(ctx, "./testdata/state/test1")
			Expect(err).ToNot(HaveOccurred())
			Expect(testutils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())

			inst := state.Installations[state.Namespace+"/a"]
			Expect(testenv.Client.Delete(ctx, inst)).To(Succeed())
			Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst)).ToNot(HaveOccurred())
			inst.Status.InstallationPhase = lsv1alpha1.InstallationPhases.Succeeded
			Expect(testutils.UpdateJobIdForInstallation(ctx, testenv, inst)).ToNot(HaveOccurred())

			Eventually(func() error {
				i := &lsv1alpha1.Installation{}
				if err := testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), i); err != nil {
					return err
				}
				if i.Status.LastError == nil {
					return errors.New("no error present")
				}
				if !strings.Contains(i.Status.LastError.Message, installationsctl.SiblingImportError.Error()) {
					return fmt.Errorf("expected the reported error to be %q but got %q", installationsctl.SiblingImportError.Error(), i.Status.LastError.Message)
				}
				return nil
			}, 20*time.Second, 1*time.Second).Should(Succeed(), "should error with a sibling import error")

			instC := &lsv1alpha1.Installation{}
			Expect(testenv.Client.Get(ctx, client.ObjectKey{Name: "c", Namespace: state.Namespace}, instC)).ToNot(HaveOccurred())
			Expect(instC.DeletionTimestamp.IsZero()).To(BeTrue())
		})

		It("should be able to delete a installation with an erroneous component descriptor", func() {
			inst := &lsv1alpha1.Installation{}
			inst.GenerateName = "test-"
			inst.Namespace = state.Namespace
			inst.Spec.ComponentDescriptor = &lsv1alpha1.ComponentDescriptorDefinition{
				Reference: &lsv1alpha1.ComponentDescriptorReference{
					RepositoryContext: testutils.ExampleRepositoryContext(),
					ComponentName:     "not-a-component",
					Version:           "v0.0.0",
				},
			}

			Expect(state.Create(ctx, inst)).To(Succeed())
			Eventually(func() error {
				i := &lsv1alpha1.Installation{}
				if err := testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), i); err != nil {
					return err
				}
				if len(i.Finalizers) == 0 {
					return errors.New("no finalizers exist on the installation")
				}
				return nil
			}, 20*time.Second, 1*time.Second).Should(Succeed(), "the installation should have been reconciled once")

			// patch status to be failed
			old := inst.DeepCopy()
			Expect(testenv.Client.Status().Patch(ctx, inst, client.MergeFrom(old)))
			Expect(testenv.Client.Delete(ctx, inst)).To(Succeed())

			Eventually(func() error {
				i := &lsv1alpha1.Installation{}
				if err := testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), i); err != nil {
					if apierrors.IsNotFound(err) {
						return nil
					}
					return err
				}
				return errors.New("installation still exist")
			}, 20*time.Second, 1*time.Second).Should(Succeed(), "the installation should be deleted")
		})
	})
})

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

	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/pkg/utils/simplelogger"

	"github.com/gardener/landscaper/apis/config"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	testutils "github.com/gardener/landscaper/test/utils"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	installationsctl "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Delete", func() {

	Context("reconciler", func() {
		var (
			op   *lsoperation.Operation
			ctrl reconcile.Reconciler

			state        *envtest.State
			fakeCompRepo ctf.ComponentResolver
		)

		BeforeEach(func() {
			var err error
			fakeCompRepo, err = componentsregistry.NewLocalClient(logr.Discard(), "./testdata")
			Expect(err).ToNot(HaveOccurred())

			op = lsoperation.NewOperation(logr.Discard(), testenv.Client, api.LandscaperScheme, record.NewFakeRecorder(1024)).SetComponentsRegistry(fakeCompRepo)

			ctrl = installationsctl.NewTestActuator(*op, &config.LandscaperConfiguration{
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

		It("should block deletion if there are still subinstallations", func() {
			ctx := context.Background()

			var err error
			state, err = testenv.InitResources(ctx, "./testdata/state/test1")
			Expect(err).ToNot(HaveOccurred())

			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), state.Installations[state.Namespace+"/root"])
			Expect(err).ToNot(HaveOccurred())

			instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inInstRoot, nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(installationsctl.DeleteExecutionAndSubinstallations(ctx, instOp)).To(Succeed())

			instA := &lsv1alpha1.Installation{}
			Expect(testenv.Client.Get(ctx, client.ObjectKey{Name: "a", Namespace: state.Namespace}, instA)).ToNot(HaveOccurred())
			instB := &lsv1alpha1.Installation{}
			Expect(testenv.Client.Get(ctx, client.ObjectKey{Name: "b", Namespace: state.Namespace}, instB)).ToNot(HaveOccurred())

			Expect(instA.DeletionTimestamp.IsZero()).To(BeFalse())
			Expect(instB.DeletionTimestamp.IsZero()).To(BeFalse())
		})

		It("should not block deletion if there are no subinstallations left", func() {
			ctx := context.Background()

			var err error
			state, err = testenv.InitResources(ctx, "./testdata/state/test1")
			Expect(err).ToNot(HaveOccurred())

			inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), state.Installations[state.Namespace+"/b"])
			Expect(err).ToNot(HaveOccurred())

			instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inInstB, nil)
			Expect(err).ToNot(HaveOccurred())

			err = installationsctl.DeleteExecutionAndSubinstallations(ctx, instOp)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete subinstallations if no one imports exported values", func() {
			ctx := context.Background()

			var err error
			state, err = testenv.InitResources(ctx, "./testdata/state/test2")
			Expect(err).ToNot(HaveOccurred())

			inInstB, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), state.Installations[state.Namespace+"/a"])
			Expect(err).ToNot(HaveOccurred())

			instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inInstB, nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(installationsctl.DeleteExecutionAndSubinstallations(ctx, instOp)).To(Succeed())

			instC := &lsv1alpha1.Installation{}
			Expect(testenv.Client.Get(ctx, client.ObjectKey{Name: "c", Namespace: state.Namespace}, instC)).ToNot(HaveOccurred())
			Expect(instC.DeletionTimestamp.IsZero()).To(BeFalse())
		})

		It("should propagate the force deletion annotation to an execution in deletion state", func() {
			ctx := context.Background()

			var err error
			state, err = testenv.InitResources(ctx, "./testdata/state/test3")
			Expect(err).ToNot(HaveOccurred())

			inst := &lsv1alpha1.Installation{}
			inst.Name = "root"
			inst.Namespace = state.Namespace
			testutils.ExpectNoError(testenv.Client.Delete(ctx, inst))
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))

			exec := &lsv1alpha1.Execution{}
			exec.Name = "root"
			exec.Namespace = state.Namespace
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))
			Expect(exec.DeletionTimestamp).ToNot(BeNil())

			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst))
			metav1.SetMetaDataAnnotation(&inst.ObjectMeta, lsv1alpha1.OperationAnnotation, string(lsv1alpha1.ForceReconcileOperation))
			testutils.ExpectNoError(testenv.Client.Update(ctx, inst))
			testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(inst))

			// execution should have the force reconcile annotation
			testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))
			ann, ok := exec.Annotations[lsv1alpha1.OperationAnnotation]
			Expect(ok).To(BeTrue())
			Expect(ann).To(Equal(string(lsv1alpha1.ForceReconcileOperation)))
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
			})
			Expect(err).ToNot(HaveOccurred())

			state, err = testenv.InitState(ctx)
			Expect(err).ToNot(HaveOccurred())

			Expect(installationsctl.AddControllerToManager(simplelogger.NewIOLogger(GinkgoWriter), mgr, nil, &config.LandscaperConfiguration{})).To(Succeed())
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

			inst := state.Installations[state.Namespace+"/a"]
			Expect(testenv.Client.Delete(ctx, inst)).To(Succeed())

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
			}, 20*time.Second, 2*time.Second).Should(Succeed(), "should error with a sibling import error")

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

			Expect(state.Create(ctx, testenv.Client, inst)).To(Succeed())
			Eventually(func() error {
				i := &lsv1alpha1.Installation{}
				if err := testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), i); err != nil {
					return err
				}
				if len(i.Finalizers) == 0 {
					return errors.New("no finalizers exist on the installation")
				}
				return nil
			}, 20*time.Second, 2*time.Second).Should(Succeed(), "the installation should have been reconciled once")

			// patch status to be failed
			old := inst.DeepCopy()
			inst.Status.Phase = lsv1alpha1.ComponentPhaseFailed
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
			}, 20*time.Second, 2*time.Second).Should(Succeed(), "the installation should be deleted")
		})
	})

})

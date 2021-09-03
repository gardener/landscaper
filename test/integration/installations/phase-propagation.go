// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func PhasePropagationTests(f *framework.Framework) {
	var (
		testdataDir = filepath.Join(f.RootPath, "test", "integration", "installations", "testdata", "test2")
	)

	Describe("Phase Propagation", func() {

		var (
			state = f.Register()
			ctx   context.Context
		)

		BeforeEach(func() {
			ctx = context.Background()
		})

		AfterEach(func() {
			ctx.Done()
		})

		It("should propagate phase changes in deploy items up to the installation", func() {
			By("Create root installation")
			root := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(root, path.Join(testdataDir, "00-root-installation.yaml")))
			root.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.CreateWithClient(ctx, f.Client, root))

			By("verify that execution has been created and fetch deploy item")
			exec := &lsv1alpha1.Execution{}
			execDi := &lsv1alpha1.DeployItem{}
			Eventually(func() (bool, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(root), root)
				if err != nil || root.Status.ExecutionReference == nil {
					return false, err
				}
				err = f.Client.Get(ctx, root.Status.ExecutionReference.NamespacedName(), exec)
				if err != nil || len(exec.Status.DeployItemReferences) == 0 {
					return false, err
				}
				err = f.Client.Get(ctx, exec.Status.DeployItemReferences[0].Reference.NamespacedName(), execDi)
				if err != nil {
					return false, err
				}
				return true, nil
			}, timeoutTime, resyncTime).Should(BeTrue(), "unable to fetch execution or deploy item")

			By("verify that subinstallation has been created")
			subinst := &lsv1alpha1.Installation{}
			Eventually(func() (bool, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(root), root)
				if err != nil || len(root.Status.InstallationReferences) == 0 {
					return false, err
				}
				err = f.Client.Get(ctx, root.Status.InstallationReferences[0].Reference.NamespacedName(), subinst)
				if err != nil {
					return false, err
				}
				return true, nil
			}, timeoutTime, resyncTime).Should(BeTrue(), "unable to fetch subinstallation")

			By("verify that subinstallation execution has been created and fetch deploy item")
			subinstExec := &lsv1alpha1.Execution{}
			subinstExecDi := &lsv1alpha1.DeployItem{}
			Eventually(func() (bool, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(subinst), subinst)
				if err != nil || subinst.Status.ExecutionReference == nil {
					return false, err
				}
				err = f.Client.Get(ctx, subinst.Status.ExecutionReference.NamespacedName(), subinstExec)
				if err != nil || len(subinstExec.Status.DeployItemReferences) == 0 {
					return false, err
				}
				err = f.Client.Get(ctx, subinstExec.Status.DeployItemReferences[0].Reference.NamespacedName(), subinstExecDi)
				if err != nil {
					return false, err
				}
				return true, nil
			}, timeoutTime, resyncTime).Should(BeTrue(), "unable to fetch execution or deploy item")

			By("verify that installations are succeeded")
			Eventually(func() (lsv1alpha1.ComponentInstallationPhase, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(root), root)
				if err != nil {
					return "", err
				}
				return root.Status.Phase, nil
			}, timeoutTime, resyncTime).Should(BeEquivalentTo(lsv1alpha1.ComponentPhaseSucceeded), "root installation should be in phase %q", string(lsv1alpha1.ComponentPhaseSucceeded))
			Eventually(func() (lsv1alpha1.ComponentInstallationPhase, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(subinst), subinst)
				if err != nil {
					return "", err
				}
				return subinst.Status.Phase, nil
			}, timeoutTime, resyncTime).Should(BeEquivalentTo(lsv1alpha1.ComponentPhaseSucceeded), "subinstallation should be in phase %q", string(lsv1alpha1.ComponentPhaseSucceeded))

			By("set execution deploy item to Failed and verify phase propagation")
			execDi.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
			utils.ExpectNoError(f.Client.Status().Update(ctx, execDi))
			Eventually(func() (lsv1alpha1.ExecutionPhase, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)
				if err != nil {
					return "", err
				}
				return exec.Status.Phase, nil
			}, timeoutTime, resyncTime).Should(BeEquivalentTo(lsv1alpha1.ExecutionPhaseFailed), "execution should be in phase %q", string(lsv1alpha1.ExecutionPhaseFailed))
			Eventually(func() (lsv1alpha1.ComponentInstallationPhase, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(root), root)
				if err != nil {
					return "", err
				}
				return root.Status.Phase, nil
			}, timeoutTime, resyncTime).Should(BeEquivalentTo(lsv1alpha1.ComponentPhaseFailed), "root installation should be in phase %q", string(lsv1alpha1.ComponentPhaseFailed))

			By("set execution deploy item to Succeeded and verify phase propagation")
			execDi.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
			utils.ExpectNoError(f.Client.Status().Update(ctx, execDi))
			Eventually(func() (lsv1alpha1.ExecutionPhase, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)
				if err != nil {
					return "", err
				}
				return exec.Status.Phase, nil
			}, timeoutTime, resyncTime).Should(BeEquivalentTo(lsv1alpha1.ExecutionPhaseSucceeded), "execution should be in phase %q", string(lsv1alpha1.ExecutionPhaseSucceeded))
			Eventually(func() (lsv1alpha1.ComponentInstallationPhase, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(root), root)
				if err != nil {
					return "", err
				}
				return root.Status.Phase, nil
			}, timeoutTime, resyncTime).Should(BeEquivalentTo(lsv1alpha1.ComponentPhaseSucceeded), "root installation should be in phase %q", string(lsv1alpha1.ComponentPhaseSucceeded))

			By("set subinstallation deploy item to Failed and verify phase propagation")
			subinstExecDi.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
			utils.ExpectNoError(f.Client.Status().Update(ctx, subinstExecDi))
			Eventually(func() (lsv1alpha1.ExecutionPhase, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(subinstExec), subinstExec)
				if err != nil {
					return "", err
				}
				return subinstExec.Status.Phase, nil
			}, timeoutTime, resyncTime).Should(BeEquivalentTo(lsv1alpha1.ExecutionPhaseFailed), "subinstallation execution should be in phase %q", string(lsv1alpha1.ExecutionPhaseFailed))
			Eventually(func() (lsv1alpha1.ComponentInstallationPhase, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(subinst), subinst)
				if err != nil {
					return "", err
				}
				return subinst.Status.Phase, nil
			}, timeoutTime, resyncTime).Should(BeEquivalentTo(lsv1alpha1.ComponentPhaseFailed), "subinstallation should be in phase %q", string(lsv1alpha1.ComponentPhaseFailed))
			Eventually(func() (lsv1alpha1.ComponentInstallationPhase, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(root), root)
				if err != nil {
					return "", err
				}
				return root.Status.Phase, nil
			}, timeoutTime, resyncTime).Should(BeEquivalentTo(lsv1alpha1.ComponentPhaseFailed), "root installation should be in phase %q", string(lsv1alpha1.ComponentPhaseFailed))

			By("set subinstallation deploy item to Succeeded and verify phase propagation")
			subinstExecDi.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
			utils.ExpectNoError(f.Client.Status().Update(ctx, subinstExecDi))
			Eventually(func() (lsv1alpha1.ExecutionPhase, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(subinstExec), subinstExec)
				if err != nil {
					return "", err
				}
				return subinstExec.Status.Phase, nil
			}, timeoutTime, resyncTime).Should(BeEquivalentTo(lsv1alpha1.ExecutionPhaseSucceeded), "subinstallation execution should be in phase %q", string(lsv1alpha1.ExecutionPhaseSucceeded))
			Eventually(func() (lsv1alpha1.ComponentInstallationPhase, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(subinst), subinst)
				if err != nil {
					return "", err
				}
				return subinst.Status.Phase, nil
			}, timeoutTime, resyncTime).Should(BeEquivalentTo(lsv1alpha1.ComponentPhaseSucceeded), "subinstallation should be in phase %q", string(lsv1alpha1.ComponentPhaseSucceeded))
			Eventually(func() (lsv1alpha1.ComponentInstallationPhase, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(root), root)
				if err != nil {
					return "", err
				}
				return root.Status.Phase, nil
			}, timeoutTime, resyncTime).Should(BeEquivalentTo(lsv1alpha1.ComponentPhaseSucceeded), "root installation should be in phase %q", string(lsv1alpha1.ComponentPhaseSucceeded))

		})

	})
}

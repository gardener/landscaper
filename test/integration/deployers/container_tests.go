// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployers

import (
	"context"
	"path"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func ContainerDeployerTestsForNewReconcile(f *framework.Framework) {
	Describe("Container Deployer", func() {
		var (
			state      = f.Register()
			exampleDir = path.Join(f.RootPath, "examples/deploy-items")

			ctx context.Context
		)

		BeforeEach(func() {
			ctx = context.Background()
		})

		AfterEach(func() {
			ctx.Done()
		})

		It("should run a simple docker image with a sleep command", func() {

			By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			target.Name = "my-cluster-target"
			target.Namespace = state.Namespace
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			di := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(utils.ReadResourceFromFile(di, path.Join(exampleDir, "30-DeployItem-Container-sleep.yaml")))
			di.SetName("")
			di.SetGenerateName("container-sleep-")
			di.SetNamespace(state.Namespace)
			di.Spec.Target = &lsv1alpha1.ObjectReference{
				Name:      target.Name,
				Namespace: target.Namespace,
			}

			By("Create container deploy item")
			utils.ExpectNoError(state.Create(ctx, di))
			// Set a new jobID to trigger a reconcile of the deploy item
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
			Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())
			utils.ExpectNoError(lsutils.WaitForDeployItemToFinish(ctx, f.Client, di, lsv1alpha1.DeployItemPhaseSucceeded, 2*time.Minute))

			By("Delete container deploy item")
			utils.ExpectNoError(f.Client.Delete(ctx, di))
			// Set a new jobID to trigger a reconcile of the deploy item
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
			Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())
		})

		It("should detect when a image cannot be pulled and succeed when the deploy item is updated", func() {
			By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			target.Name = "my-cluster-target"
			target.Namespace = state.Namespace
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			di := utils.BuildContainerDeployItem(&containerv1alpha1.ProviderConfiguration{
				Image: "example.com/some-invalid/image:v0.0.1",
			})
			di.SetName("")
			di.SetGenerateName("container-sleep-")
			di.SetNamespace(state.Namespace)
			di.Spec.Target = &lsv1alpha1.ObjectReference{
				Name:      target.Name,
				Namespace: target.Namespace,
			}

			By("Create erroneous container deploy item")
			utils.ExpectNoError(state.Create(ctx, di))
			// Set a new jobID to trigger a reconcile of the deploy item
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
			Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())
			utils.ExpectNoError(lsutils.WaitForDeployItemToFinish(ctx, f.Client, di, lsv1alpha1.DeployItemPhaseFailed, 2*time.Minute))

			By("update the DeployItem and set a valid image")
			utils.ExpectNoError(f.Client.Get(ctx, kutil.ObjectKey(di.Name, di.Namespace), di))
			updatedDi := utils.BuildContainerDeployItem(&containerv1alpha1.ProviderConfiguration{
				Image: "alpine",
			})
			di.Spec.Configuration = updatedDi.Spec.Configuration
			utils.ExpectNoError(f.Client.Update(ctx, di))

			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
			di.Status.SetJobID("2")
			Expect(state.Client.Status().Update(ctx, di)).To(Succeed())

			utils.ExpectNoError(lsutils.WaitForDeployItemToFinish(ctx, f.Client, di, lsv1alpha1.DeployItemPhaseSucceeded, 2*time.Minute))

			By("Delete container deploy item")
			utils.ExpectNoError(f.Client.Delete(ctx, di))
			// Set a new jobID to trigger a reconcile of the deploy item
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
			Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())
		})

		It("should export data", func() {
			By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			target.Name = "my-cluster-target"
			target.Namespace = state.Namespace
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			di := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(utils.ReadResourceFromFile(di, path.Join(exampleDir, "31-DeployItem-Container-export.yaml")))
			di.SetName("")
			di.SetGenerateName("container-export-")
			di.SetNamespace(state.Namespace)
			di.Spec.Target = &lsv1alpha1.ObjectReference{
				Name:      target.Name,
				Namespace: target.Namespace,
			}

			By("Create container deploy item")
			utils.ExpectNoError(state.Create(ctx, di))
			// Set a new jobID to trigger a reconcile of the deploy item
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
			Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())
			utils.ExpectNoError(lsutils.WaitForDeployItemToFinish(ctx, f.Client, di, lsv1alpha1.DeployItemPhaseSucceeded, 2*time.Minute))

			// expect that the export contains a valid json with { "my-val": true }
			Expect(di.Status.ExportReference).ToNot(BeNil())
			exportData, err := lsutils.GetDeployItemExport(ctx, f.Client, di)
			utils.ExpectNoError(err)
			Expect(exportData).To(MatchJSON(`{ "my-val": true }`))

			By("Delete container deploy item")
			utils.ExpectNoError(f.Client.Delete(ctx, di))
			// Set a new jobID to trigger a reconcile of the deploy item
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
			Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())
		})

		It("should write and read data from the state", func() {
			By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			target.Name = "my-cluster-target"
			target.Namespace = state.Namespace
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			di := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(utils.ReadResourceFromFile(di, path.Join(exampleDir, "32-DeployItem-Container-state.yaml")))
			di.SetName("")
			di.SetGenerateName("container-export-")
			di.SetNamespace(state.Namespace)
			di.Spec.Target = &lsv1alpha1.ObjectReference{
				Name:      target.Name,
				Namespace: target.Namespace,
			}

			By("Create container deploy item")
			utils.ExpectNoError(state.Create(ctx, di))
			// Set a new jobID to trigger a reconcile of the deploy item
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
			Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())
			utils.ExpectNoError(lsutils.WaitForDeployItemToFinish(ctx, f.Client, di, lsv1alpha1.DeployItemPhaseSucceeded, 2*time.Minute))

			// expect that the export contains a valid json with { "counter": 1 }
			Expect(di.Status.ExportReference).ToNot(BeNil())
			exportData, err := lsutils.GetDeployItemExport(ctx, f.Client, di)
			utils.ExpectNoError(err)
			Expect(exportData).To(MatchJSON(`{ "counter": 1 }`))

			By("Rerun the deployitem")
			// Set a new jobID to trigger a reconcile of the deploy item
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
			Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())

			utils.ExpectNoError(lsutils.WaitForDeployItemToFinish(ctx, f.Client, di, lsv1alpha1.DeployItemPhaseSucceeded, 2*time.Minute))
			// expect that the export contains a valid json with { "counter": 2 }
			Expect(di.Status.ExportReference).ToNot(BeNil())
			exportData, err = lsutils.GetDeployItemExport(ctx, f.Client, di)
			utils.ExpectNoError(err)
			Expect(exportData).To(MatchJSON(`{ "counter": 2 }`))

			By("Delete container deploy item")
			utils.ExpectNoError(f.Client.Delete(ctx, di))
			// Set a new jobID to trigger a reconcile of the deploy item
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
			Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())
		})
	})
}

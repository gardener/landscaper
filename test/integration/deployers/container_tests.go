// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployers

import (
	"context"
	"path"
	"time"

	"github.com/onsi/ginkgo"
	g "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

// RegisterTests registers all tests of this package
func RegisterTests(f *framework.Framework) {
	ManifestDeployerTests(f)
}

func ManifestDeployerTests(f *framework.Framework) {
	var (
		dumper     = f.Register()
		exampleDir = path.Join(f.RootPath, "examples/deploy-items")
	)

	ginkgo.Describe("Manifest Deployer", func() {

		ginkgo.It("should run a simple docker image with a sleep command", func() {
			ctx := context.Background()
			defer ctx.Done()
			state, cleanup, err := f.NewState(ctx)
			utils.ExpectNoError(err)
			dumper.AddNamespaces(state.Namespace)
			defer func() {
				g.Expect(cleanup(ctx)).ToNot(g.HaveOccurred())
			}()

			ginkgo.By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			target.Name = "my-cluster-target"
			target.Namespace = state.Namespace
			target, err = utils.CreateInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, f.Client, target))

			di := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(utils.ReadResourceFromFile(di, path.Join(exampleDir, "30-DeployItem-Container-sleep.yaml")))
			di.SetName("")
			di.SetGenerateName("container-sleep-")
			di.SetNamespace(state.Namespace)
			di.Spec.Target = &lsv1alpha1.ObjectReference{
				Name:      target.Name,
				Namespace: target.Namespace,
			}

			ginkgo.By("Create container deploy item")
			utils.ExpectNoError(state.Create(ctx, f.Client, di))
			utils.ExpectNoError(utils.WaitForDeployItemToBeInPhase(ctx, f.Client, di, lsv1alpha1.ExecutionPhaseSucceeded, 2*time.Minute))

			ginkgo.By("Delete container deploy item")
			utils.ExpectNoError(f.Client.Delete(ctx, di))
		})

		ginkgo.It("should detect when a image cannot be pulled and succeed when the deploy item is updated", func() {
			ctx := context.Background()
			defer ctx.Done()
			state, cleanup, err := f.NewState(ctx)
			utils.ExpectNoError(err)
			dumper.AddNamespaces(state.Namespace)
			defer func() {
				g.Expect(cleanup(ctx)).ToNot(g.HaveOccurred())
			}()

			ginkgo.By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			target.Name = "my-cluster-target"
			target.Namespace = state.Namespace
			target, err = utils.CreateInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, f.Client, target))

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

			ginkgo.By("Create erroneous container deploy item")
			utils.ExpectNoError(state.Create(ctx, f.Client, di))
			utils.ExpectNoError(utils.WaitForDeployItemToBeInPhase(ctx, f.Client, di, lsv1alpha1.ExecutionPhaseFailed, 2*time.Minute))

			ginkgo.By("update the DeployItem and set the correct image")
			utils.ExpectNoError(f.Client.Get(ctx, kutil.ObjectKey(di.Name, di.Namespace), di))
			updatedDi := utils.BuildContainerDeployItem(&containerv1alpha1.ProviderConfiguration{
				Image: "alpine",
			})
			di.Spec.Configuration = updatedDi.Spec.Configuration
			utils.ExpectNoError(f.Client.Update(ctx, di))

			utils.ExpectNoError(utils.WaitForDeployItemToBeInPhase(ctx, f.Client, di, lsv1alpha1.ExecutionPhaseSucceeded, 2*time.Minute))

			ginkgo.By("Delete container deploy item")
			utils.ExpectNoError(f.Client.Delete(ctx, di))
		})

		ginkgo.FIt("should export data", func() {
			ctx := context.Background()
			defer ctx.Done()
			state, cleanup, err := f.NewState(ctx)
			utils.ExpectNoError(err)
			dumper.AddNamespaces(state.Namespace)
			defer func() {
				g.Expect(cleanup(ctx)).ToNot(g.HaveOccurred())
			}()

			ginkgo.By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			target.Name = "my-cluster-target"
			target.Namespace = state.Namespace
			target, err = utils.CreateInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, f.Client, target))

			di := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(utils.ReadResourceFromFile(di, path.Join(exampleDir, "31-DeployItem-Container-export.yaml")))
			di.SetName("")
			di.SetGenerateName("container-export-")
			di.SetNamespace(state.Namespace)
			di.Spec.Target = &lsv1alpha1.ObjectReference{
				Name:      target.Name,
				Namespace: target.Namespace,
			}

			ginkgo.By("Create container deploy item")
			utils.ExpectNoError(state.Create(ctx, f.Client, di))
			utils.ExpectNoError(utils.WaitForDeployItemToBeInPhase(ctx, f.Client, di, lsv1alpha1.ExecutionPhaseSucceeded, 2*time.Minute))

			// expect that the export contains a valid json with { "my-val": true }
			g.Expect(di.Status.ExportReference).ToNot(g.BeNil())
			exportData, err := utils.GetDeployItemExport(ctx, f.Client, di)
			utils.ExpectNoError(err)
			g.Expect(exportData).To(g.MatchJSON(`{ "my-val": true }`))

			ginkgo.By("Delete container deploy item")
			utils.ExpectNoError(f.Client.Delete(ctx, di))
		})

	})

}

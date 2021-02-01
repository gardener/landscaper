// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package tutorial

import (
	"context"
	"path/filepath"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

// RegisterTests registers all tests of the package
func RegisterTests(f *framework.Framework) {
	SimpleIngressNginxTest(f)
}

func SimpleIngressNginxTest(f *framework.Framework) {
	dumper := f.Register()

	_ = ginkgo.Describe("SimpleIngressNginxTest", func() {

		ginkgo.It("should deploy a nginx ingress controller", func() {
			var (
				tutorialResourcesRootDir = filepath.Join(f.RootPath, "/docs/tutorials/resources/ingress-nginx")
				targetResource           = filepath.Join(tutorialResourcesRootDir, "my-target.yaml")
				instResource             = filepath.Join(tutorialResourcesRootDir, "installation.yaml")
			)
			ctx := context.Background()
			defer ctx.Done()
			state, cleanup, err := f.NewState(ctx)
			utils.ExpectNoError(err)
			dumper.AddNamespaces(state.Namespace)
			defer func() {
				gomega.Expect(cleanup(ctx)).ToNot(gomega.HaveOccurred())
			}()

			ginkgo.By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			utils.ExpectNoError(utils.ReadResourceFromFile(target, targetResource))
			target, err = utils.CreateInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, f.Client, target))

			ginkgo.By("Create Installation")
			inst := &lsv1alpha1.Installation{}
			inst.SetNamespace(state.Namespace)
			gomega.Expect(utils.ReadResourceFromFile(inst, instResource)).To(gomega.Succeed())

			utils.ExpectNoError(state.Create(ctx, f.Client, inst))

			// wait for installation to finish
			utils.ExpectNoError(utils.WaitForInstallationToBeInPhase(ctx, f.Client, inst, lsv1alpha1.ComponentPhaseSucceeded, 2*time.Minute))

			// todo: check if the deploy item is succeeded and that the nginx is successfully running in the default namespace
			deployItems, err := utils.GetDeployItemsOfInstallation(ctx, f.Client, inst)
			utils.ExpectNoError(err)
			gomega.Expect(deployItems).To(gomega.HaveLen(1))
			gomega.Expect(deployItems[0].Status.Phase).To(gomega.Equal(lsv1alpha1.ExecutionPhaseSucceeded))

			// todo: make namespace configurable for deployed resources
			// expect that the nginx deployment is successfully running
			nginxIngressDeploymentName := "test-ingress-nginx-controller"
			utils.ExpectNoError(utils.WaitForDeploymentToBeReady(ctx, f.Client, kutil.ObjectKey(nginxIngressDeploymentName, "default"), 2*time.Minute))

			ginkgo.By("Delete installation")
			utils.ExpectNoError(f.Client.Delete(ctx, inst))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, inst, 2*time.Minute))

			// expect that the nginx deployment is alread deleted or has an deletion timestamp
			nginxDeployment := &appsv1.Deployment{}
			err = f.Client.Get(ctx, kutil.ObjectKey(nginxIngressDeploymentName, "default"), nginxDeployment)
			if err != nil && !apierrors.IsNotFound(err) {
				utils.ExpectNoError(err)
			} else if err == nil {
				gomega.Expect(nginxDeployment.DeletionTimestamp.IsZero()).To(gomega.BeTrue())
			}
		})
	})
}

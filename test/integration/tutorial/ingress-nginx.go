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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func NginxIngressTest(f *framework.Framework) {
	_ = ginkgo.Describe("SimpleNginxTest", func() {
		state := f.Register()

		ginkgo.It("should deploy a nginx ingress controller", func() {
			var (
				tutorialResourcesRootDir = filepath.Join(f.RootPath, "/docs/tutorials/resources/ingress-nginx")
				targetResource           = filepath.Join(tutorialResourcesRootDir, "my-target.yaml")
				importResource           = filepath.Join(tutorialResourcesRootDir, "configmap.yaml")
				instResource             = filepath.Join(tutorialResourcesRootDir, "installation.yaml")
			)
			ctx := context.Background()
			defer ctx.Done()

			ginkgo.By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			utils.ExpectNoError(utils.ReadResourceFromFile(target, targetResource))
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.CreateWithClient(ctx, f.Client, target))

			ginkgo.By("Create ConfigMap with imports for the installation")
			cm := &corev1.ConfigMap{}
			utils.ExpectNoError(utils.ReadResourceFromFile(cm, importResource))
			cm.SetNamespace(state.Namespace)
			cm.Data["namespace"] = state.Namespace
			utils.ExpectNoError(state.CreateWithClient(ctx, f.Client, cm))

			ginkgo.By("Create Installation")
			inst := &lsv1alpha1.Installation{}
			gomega.Expect(utils.ReadResourceFromFile(inst, instResource)).To(gomega.Succeed())
			inst.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.CreateWithClient(ctx, f.Client, inst))

			// wait for installation to finish
			utils.ExpectNoError(lsutils.WaitForInstallationToBeHealthy(ctx, f.Client, inst, 2*time.Minute))

			deployItems, err := lsutils.GetDeployItemsOfInstallation(ctx, f.Client, inst)
			utils.ExpectNoError(err)
			gomega.Expect(deployItems).To(gomega.HaveLen(1))
			gomega.Expect(deployItems[0].Status.Phase).To(gomega.Equal(lsv1alpha1.ExecutionPhaseSucceeded))

			// expect that the nginx deployment is successfully running
			nginxIngressDeploymentName := "test-ingress-nginx-controller"
			nginxIngressObjectKey := kutil.ObjectKey(nginxIngressDeploymentName, state.Namespace)
			utils.ExpectNoError(utils.WaitForDeploymentToBeReady(ctx, f.TestLog(), f.Client, nginxIngressObjectKey, 2*time.Minute))

			ginkgo.By("Delete installation")
			utils.ExpectNoError(f.Client.Delete(ctx, inst))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, inst, 2*time.Minute))

			// expect that the nginx deployment is already deleted or has an deletion timestamp
			nginxDeployment := &appsv1.Deployment{}
			err = f.Client.Get(ctx, nginxIngressObjectKey, nginxDeployment)
			if err != nil && !apierrors.IsNotFound(err) {
				utils.ExpectNoError(err)
			} else if err == nil {
				gomega.Expect(nginxDeployment.DeletionTimestamp.IsZero()).To(gomega.BeTrue())
			}
		})
	})

	_ = ginkgo.Describe("LocalIngressNginxTest", func() {
		state := f.Register()

		ginkgo.It("should deploy a nginx ingress controller with local artifacts", func() {
			var (
				tutorialResourcesRootDir = filepath.Join(f.RootPath, "/docs/tutorials/resources/local-ingress-nginx")
				targetResource           = filepath.Join(tutorialResourcesRootDir, "my-target.yaml")
				importResource           = filepath.Join(tutorialResourcesRootDir, "configmap.yaml")
				instResource             = filepath.Join(tutorialResourcesRootDir, "installation.yaml")
			)
			ctx := context.Background()
			defer ctx.Done()

			ginkgo.By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			utils.ExpectNoError(utils.ReadResourceFromFile(target, targetResource))
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.CreateWithClient(ctx, f.Client, target))

			ginkgo.By("Create ConfigMap with imports for the installation")
			cm := &corev1.ConfigMap{}
			utils.ExpectNoError(utils.ReadResourceFromFile(cm, importResource))
			cm.SetNamespace(state.Namespace)
			cm.Data["namespace"] = state.Namespace
			utils.ExpectNoError(state.CreateWithClient(ctx, f.Client, cm))

			ginkgo.By("Create Installation")
			inst := &lsv1alpha1.Installation{}
			gomega.Expect(utils.ReadResourceFromFile(inst, instResource)).To(gomega.Succeed())
			inst.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.CreateWithClient(ctx, f.Client, inst))

			// wait for installation to finish
			utils.ExpectNoError(lsutils.WaitForInstallationToBeHealthy(ctx, f.Client, inst, 2*time.Minute))

			deployItems, err := lsutils.GetDeployItemsOfInstallation(ctx, f.Client, inst)
			utils.ExpectNoError(err)
			gomega.Expect(deployItems).To(gomega.HaveLen(1))
			gomega.Expect(deployItems[0].Status.Phase).To(gomega.Equal(lsv1alpha1.ExecutionPhaseSucceeded))

			// expect that the nginx deployment is successfully running
			nginxIngressDeploymentName := "test-ingress-nginx-controller"
			nginxIngressObjectKey := kutil.ObjectKey(nginxIngressDeploymentName, state.Namespace)
			utils.ExpectNoError(utils.WaitForDeploymentToBeReady(ctx, f.TestLog(), f.Client, nginxIngressObjectKey, 2*time.Minute))

			ginkgo.By("Delete installation")
			utils.ExpectNoError(f.Client.Delete(ctx, inst))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, inst, 2*time.Minute))

			// expect that the nginx deployment is already deleted or has an deletion timestamp
			nginxDeployment := &appsv1.Deployment{}
			err = f.Client.Get(ctx, nginxIngressObjectKey, nginxDeployment)
			if err != nil && !apierrors.IsNotFound(err) {
				utils.ExpectNoError(err)
			} else if err == nil {
				gomega.Expect(nginxDeployment.DeletionTimestamp.IsZero()).To(gomega.BeTrue())
			}
		})
	})
}

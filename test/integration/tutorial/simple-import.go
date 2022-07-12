// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package tutorial

import (
	"context"
	"path/filepath"
	"time"

	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func SimpleImport(f *framework.Framework) {
	_ = Describe("SimpleImport", func() {
		state := f.Register()

		It("should deploy a nginx ingress controller and a echo-server", func() {
			var (
				nginxTutorialResourcesRootDir      = filepath.Join(f.RootPath, "/docs/tutorials/resources/local-ingress-nginx")
				echoServerTutorialResourcesRootDir = filepath.Join(f.RootPath, "/docs/tutorials/resources/echo-server")
				targetResource                     = filepath.Join(nginxTutorialResourcesRootDir, "my-target.yaml")
				importResource                     = filepath.Join(nginxTutorialResourcesRootDir, "configmap.yaml")
				nginxInstResource                  = filepath.Join(nginxTutorialResourcesRootDir, "installation.yaml")
				echoServerInstResource             = filepath.Join(echoServerTutorialResourcesRootDir, "installation.yaml")
			)
			ctx := context.Background()
			defer ctx.Done()

			By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			utils.ExpectNoError(utils.ReadResourceFromFile(target, targetResource))
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create ConfigMap with imports for the installation")
			cm := &corev1.ConfigMap{}
			cm.SetNamespace(state.Namespace)
			utils.ExpectNoError(utils.ReadResourceFromFile(cm, importResource))
			cm.Data["namespace"] = state.Namespace
			utils.ExpectNoError(state.Create(ctx, cm))

			By("Create Nginx Ingress Installation")
			nginxInst := &lsv1alpha1.Installation{}
			Expect(utils.ReadResourceFromFile(nginxInst, nginxInstResource)).To(Succeed())
			nginxInst.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, nginxInst))

			// wait for installation to finish
			utils.ExpectNoError(lsutils.WaitForInstallationToBeHealthy(ctx, f.Client, nginxInst, 2*time.Minute))

			By("Create echo server Installation")
			inst := &lsv1alpha1.Installation{}
			Expect(utils.ReadResourceFromFile(inst, echoServerInstResource)).To(Succeed())
			inst.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, inst))

			// wait for installation to finish
			utils.ExpectNoError(lsutils.WaitForInstallationToBeHealthy(ctx, f.Client, inst, 2*time.Minute))

			deployItems, err := lsutils.GetDeployItemsOfInstallation(ctx, f.Client, inst)
			utils.ExpectNoError(err)
			Expect(deployItems).To(HaveLen(1))
			Expect(deployItems[0].Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))

			// expect that the echo server deployment is successfully running
			echoServerDeploymentName := "echo-server"
			echoServerObjectKey := kutil.ObjectKey(echoServerDeploymentName, state.Namespace)
			utils.ExpectNoError(utils.WaitForDeploymentToBeReady(ctx, f.TestLog(), f.Client, echoServerObjectKey, 2*time.Minute))

			// todo check if the echo server can be pinged

			By("Delete echo server installation")
			utils.ExpectNoError(f.Client.Delete(ctx, inst))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, inst, 2*time.Minute))

			// expect that the echo server deployment is already deleted or has an deletion timestamp
			echoServerDeployment := &appsv1.Deployment{}
			err = f.Client.Get(ctx, echoServerObjectKey, echoServerDeployment)
			if err != nil && !apierrors.IsNotFound(err) {
				utils.ExpectNoError(err)
			} else if err == nil {
				Expect(echoServerDeployment.DeletionTimestamp.IsZero()).To(BeTrue())
			}

			By("Delete nginx installation")
			utils.ExpectNoError(f.Client.Delete(ctx, nginxInst))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, nginxInst, 2*time.Minute))
		})
	})

}

func SimpleImportForNewReconcile(f *framework.Framework) {
	_ = Describe("SimpleImport", func() {
		state := f.Register()

		It("should deploy a nginx ingress controller and a echo-server", func() {
			var (
				nginxTutorialResourcesRootDir      = filepath.Join(f.RootPath, "/docs/tutorials/resources/local-ingress-nginx")
				echoServerTutorialResourcesRootDir = filepath.Join(f.RootPath, "/docs/tutorials/resources/echo-server")
				targetResource                     = filepath.Join(nginxTutorialResourcesRootDir, "my-target.yaml")
				importResource                     = filepath.Join(nginxTutorialResourcesRootDir, "configmap.yaml")
				nginxInstResource                  = filepath.Join(nginxTutorialResourcesRootDir, "installation.yaml")
				echoServerInstResource             = filepath.Join(echoServerTutorialResourcesRootDir, "installation.yaml")
			)
			ctx := context.Background()
			defer ctx.Done()

			By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			utils.ExpectNoError(utils.ReadResourceFromFile(target, targetResource))
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create ConfigMap with imports for the installation")
			cm := &corev1.ConfigMap{}
			cm.SetNamespace(state.Namespace)
			utils.ExpectNoError(utils.ReadResourceFromFile(cm, importResource))
			cm.Data["namespace"] = state.Namespace
			utils.ExpectNoError(state.Create(ctx, cm))

			By("Create Nginx Ingress Installation")
			nginxInst := &lsv1alpha1.Installation{}
			Expect(utils.ReadResourceFromFile(nginxInst, nginxInstResource)).To(Succeed())
			nginxInst.SetNamespace(state.Namespace)
			lsv1alpha1helper.SetOperation(&nginxInst.ObjectMeta, lsv1alpha1.ReconcileOperation)
			utils.ExpectNoError(state.Create(ctx, nginxInst))

			// wait for installation to finish
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, nginxInst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Create echo server Installation")
			inst := &lsv1alpha1.Installation{}
			Expect(utils.ReadResourceFromFile(inst, echoServerInstResource)).To(Succeed())
			inst.SetNamespace(state.Namespace)
			lsv1alpha1helper.SetOperation(&inst.ObjectMeta, lsv1alpha1.ReconcileOperation)
			utils.ExpectNoError(state.Create(ctx, inst))

			// wait for installation to finish
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			deployItems, err := lsutils.GetDeployItemsOfInstallation(ctx, f.Client, inst)
			utils.ExpectNoError(err)
			Expect(deployItems).To(HaveLen(1))
			Expect(deployItems[0].Status.DeployItemPhase).To(Equal(lsv1alpha1.DeployItemPhaseSucceeded))
			Expect(deployItems[0].Status.JobIDFinished).To(Equal(deployItems[0].Status.JobID))

			// expect that the echo server deployment is successfully running
			echoServerDeploymentName := "echo-server"
			echoServerObjectKey := kutil.ObjectKey(echoServerDeploymentName, state.Namespace)
			utils.ExpectNoError(utils.WaitForDeploymentToBeReady(ctx, f.TestLog(), f.Client, echoServerObjectKey, 2*time.Minute))

			// todo check if the echo server can be pinged

			By("Delete echo server installation")
			utils.ExpectNoError(f.Client.Delete(ctx, inst))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, inst, 2*time.Minute))

			// expect that the echo server deployment is already deleted or has an deletion timestamp
			echoServerDeployment := &appsv1.Deployment{}
			err = f.Client.Get(ctx, echoServerObjectKey, echoServerDeployment)
			if err != nil && !apierrors.IsNotFound(err) {
				utils.ExpectNoError(err)
			} else if err == nil {
				Expect(echoServerDeployment.DeletionTimestamp.IsZero()).To(BeTrue())
			}

			By("Delete nginx installation")
			utils.ExpectNoError(f.Client.Delete(ctx, nginxInst))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, nginxInst, 2*time.Minute))
		})
	})

}

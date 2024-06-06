// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package tutorial

import (
	"context"
	"path/filepath"
	"time"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func SimpleImportForNewReconcile(f *framework.Framework) {
	_ = Describe("SimpleImport", func() {
		var (
			state = f.Register()
			ctx   context.Context
		)

		log, err := logging.GetLogger()
		if err != nil {
			f.Log().Logfln("Error fetching logger: %w", err)
			return
		}

		BeforeEach(func() {
			ctx = context.Background()
			ctx = logging.NewContext(ctx, log)
		})

		It("should deploy a nginx ingress controller and a echo-server", func() {
			var (
				nginxTutorialResourcesRootDir      = filepath.Join(f.RootPath, "/test/integration/testdata/tutorial/local-ingress-nginx")
				echoServerTutorialResourcesRootDir = filepath.Join(f.RootPath, "/test/integration/testdata/tutorial/echo-server")
				targetResource                     = filepath.Join(nginxTutorialResourcesRootDir, "my-target.yaml")
				importResource                     = filepath.Join(nginxTutorialResourcesRootDir, "configmap.yaml")
				nginxInstResource                  = filepath.Join(nginxTutorialResourcesRootDir, "installation.yaml")
				echoServerInstResource             = filepath.Join(echoServerTutorialResourcesRootDir, "installation.yaml")
				echoServerInstResourceWrong        = filepath.Join(echoServerTutorialResourcesRootDir, "installation-wrong.yaml")
			)

			defer ctx.Done()

			By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			utils.ExpectNoError(utils.ReadResourceFromFile(target, targetResource))
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig)
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
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, nginxInst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Create echo server Installation")
			inst := &lsv1alpha1.Installation{}
			Expect(utils.ReadResourceFromFile(inst, echoServerInstResource)).To(Succeed())
			inst.SetNamespace(state.Namespace)
			lsv1alpha1helper.SetOperation(&inst.ObjectMeta, lsv1alpha1.ReconcileOperation)
			utils.ExpectNoError(state.Create(ctx, inst))

			// wait for installation to finish
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			deployItems, err := utils.GetDeployItemsOfInstallation(ctx, f.Client, inst)
			utils.ExpectNoError(err)
			Expect(deployItems).To(HaveLen(1))
			Expect(deployItems[0].Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.Succeeded))
			Expect(deployItems[0].Status.JobIDFinished).To(Equal(deployItems[0].Status.GetJobID()))

			// expect that the echo server deployment is successfully running
			echoServerDeploymentName := "echo-server"
			echoServerObjectKey := kutil.ObjectKey(echoServerDeploymentName, state.Namespace)
			utils.ExpectNoError(utils.WaitForDeploymentToBeReady(ctx, f.TestLog(), f.Client, echoServerObjectKey, 2*time.Minute))

			// todo check if the echo server can be pinged

			By("Delete echo server installation")
			utils.ExpectNoError(f.Client.Delete(ctx, inst))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, inst, 2*time.Minute))

			// expect that the echo server deployment will be deleted
			echoServerDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      echoServerObjectKey.Name,
					Namespace: echoServerObjectKey.Namespace,
				},
			}
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, echoServerDeployment, 2*time.Minute))

			By("Create invalid echo server Installation")
			wrongEchoInst := &lsv1alpha1.Installation{}
			Expect(utils.ReadResourceFromFile(wrongEchoInst, echoServerInstResourceWrong)).To(Succeed())
			wrongEchoInst.SetNamespace(state.Namespace)
			lsv1alpha1helper.SetOperation(&wrongEchoInst.ObjectMeta, lsv1alpha1.ReconcileOperation)
			utils.ExpectNoError(state.Create(ctx, wrongEchoInst))

			// wait for installation to finish
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, wrongEchoInst, lsv1alpha1.InstallationPhases.Failed, 2*time.Minute))

			By("Delete invalid echo server installation")
			utils.ExpectNoError(f.Client.Delete(ctx, wrongEchoInst))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, wrongEchoInst, 2*time.Minute))

			By("Delete nginx installation")
			utils.ExpectNoError(f.Client.Delete(ctx, nginxInst))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, nginxInst, 2*time.Minute))

			// expect that the nginx deployment will be deleted
			nginxDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ingress-nginx-controller",
					Namespace: state.Namespace,
				},
			}
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, nginxDeployment, 2*time.Minute))
		})
	})
}

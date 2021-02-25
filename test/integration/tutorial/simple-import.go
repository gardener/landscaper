// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package tutorial

import (
	"context"
	"path/filepath"
	"time"

	"github.com/onsi/ginkgo"
	g "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func SimpleImport(f *framework.Framework) {
	_ = ginkgo.Describe("SimpleImport", func() {
		dumper := f.Register()

		ginkgo.It("should deploy a nginx ingress controller and a echo-server", func() {
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
			state, cleanup, err := f.NewState(ctx)
			utils.ExpectNoError(err)
			dumper.AddNamespaces(state.Namespace)
			defer func() {
				g.Expect(cleanup(ctx)).ToNot(g.HaveOccurred())
			}()

			ginkgo.By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			utils.ExpectNoError(utils.ReadResourceFromFile(target, targetResource))
			target, err = utils.CreateInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, f.Client, target))

			ginkgo.By("Create ConfigMap with imports for the installation")
			cm := &corev1.ConfigMap{}
			cm.SetNamespace(state.Namespace)
			utils.ExpectNoError(utils.ReadResourceFromFile(cm, importResource))
			cm.Data["namespace"] = state.Namespace
			utils.ExpectNoError(state.Create(ctx, f.Client, cm))

			ginkgo.By("Create Nginx Ingress Installation")
			nginxInst := &lsv1alpha1.Installation{}
			nginxInst.SetNamespace(state.Namespace)
			g.Expect(utils.ReadResourceFromFile(nginxInst, nginxInstResource)).To(g.Succeed())
			utils.ExpectNoError(state.Create(ctx, f.Client, nginxInst))

			// wait for installation to finish
			utils.ExpectNoError(utils.WaitForInstallationToBeInPhase(ctx, f.Client, nginxInst, lsv1alpha1.ComponentPhaseSucceeded, 2*time.Minute))

			ginkgo.By("Create echo server Installation")
			inst := &lsv1alpha1.Installation{}
			inst.SetNamespace(state.Namespace)
			g.Expect(utils.ReadResourceFromFile(inst, echoServerInstResource)).To(g.Succeed())
			utils.ExpectNoError(state.Create(ctx, f.Client, inst))

			// wait for installation to finish
			utils.ExpectNoError(utils.WaitForInstallationToBeInPhase(ctx, f.Client, inst, lsv1alpha1.ComponentPhaseSucceeded, 2*time.Minute))

			deployItems, err := utils.GetDeployItemsOfInstallation(ctx, f.Client, inst)
			utils.ExpectNoError(err)
			g.Expect(deployItems).To(g.HaveLen(1))
			g.Expect(deployItems[0].Status.Phase).To(g.Equal(lsv1alpha1.ExecutionPhaseSucceeded))

			// expect that the echo server deployment is successfully running
			echoServerDeploymentName := "echo-server"
			echoServerObjectKey := kutil.ObjectKey(echoServerDeploymentName, state.Namespace)
			utils.ExpectNoError(utils.WaitForDeploymentToBeReady(ctx, f.TestLog(), f.Client, echoServerObjectKey, 2*time.Minute))

			// todo check if the echo server can be pinged

			ginkgo.By("Delete echo server installation")
			utils.ExpectNoError(f.Client.Delete(ctx, inst))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, inst, 2*time.Minute))

			// expect that the echo server deployment is already deleted or has an deletion timestamp
			echoServerDeployment := &appsv1.Deployment{}
			err = f.Client.Get(ctx, echoServerObjectKey, echoServerDeployment)
			if err != nil && !apierrors.IsNotFound(err) {
				utils.ExpectNoError(err)
			} else if err == nil {
				g.Expect(echoServerDeployment.DeletionTimestamp.IsZero()).To(g.BeTrue())
			}

			ginkgo.By("Delete nginx installation")
			utils.ExpectNoError(f.Client.Delete(ctx, nginxInst))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, nginxInst, 2*time.Minute))
		})
	})

}

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

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func AggregatedBlueprint(f *framework.Framework) {
	dumper := f.Register()

	_ = ginkgo.Describe("AggregatedBlueprint", func() {

		ginkgo.It("should deploy a nginx ingress controller and a echo-server together with an aggregated blueprint", func() {
			var (
				tutorialResourcesRootDir = filepath.Join(f.RootPath, "/docs/tutorials/resources/aggregated")
				targetResource           = filepath.Join(tutorialResourcesRootDir, "my-target.yaml")
				importResource           = filepath.Join(tutorialResourcesRootDir, "configmap.yaml")
				nginxInstResource        = filepath.Join(tutorialResourcesRootDir, "installation.yaml")
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
			target, err = utils.CreateInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, f.Client, target))

			ginkgo.By("Create ConfigMap with imports for the installation")
			cm := &corev1.ConfigMap{}
			cm.SetNamespace(state.Namespace)
			utils.ExpectNoError(utils.ReadResourceFromFile(cm, importResource))
			cm.Data["namespace"] = state.Namespace
			utils.ExpectNoError(state.Create(ctx, f.Client, cm))

			ginkgo.By("Create Aggregated Installation")
			aggInst := &lsv1alpha1.Installation{}
			aggInst.SetNamespace(state.Namespace)
			g.Expect(utils.ReadResourceFromFile(aggInst, nginxInstResource)).To(g.Succeed())
			utils.ExpectNoError(state.Create(ctx, f.Client, aggInst))

			// wait for installation to finish
			utils.ExpectNoError(utils.WaitForInstallationToBeInPhase(ctx, f.Client, aggInst, lsv1alpha1.ComponentPhaseSucceeded, 2*time.Minute))

			subInstallations, err := utils.GetSubInstallationsOfInstallation(ctx, f.Client, aggInst)
			utils.ExpectNoError(err)
			g.Expect(subInstallations).To(g.HaveLen(2))
			g.Expect(subInstallations[0].Status.Phase).To(g.Equal(lsv1alpha1.ComponentPhaseSucceeded))
			g.Expect(subInstallations[1].Status.Phase).To(g.Equal(lsv1alpha1.ComponentPhaseSucceeded))

			// expect that the nginx deployment is successfully running
			nginxDeployment := &appsv1.Deployment{}
			nginxDeployment.Name = "test-ingress-nginx-controller"
			nginxDeployment.Namespace = state.Namespace
			utils.ExpectNoError(utils.WaitForDeploymentToBeReady(ctx, f.TestLog(), f.Client, kutil.ObjectKeyFromObject(nginxDeployment), 2*time.Minute))

			// expect that the echo server deployment is successfully running
			echoServerDeployment := &appsv1.Deployment{}
			echoServerDeployment.Name = "echo-server"
			echoServerDeployment.Namespace = state.Namespace
			utils.ExpectNoError(utils.WaitForDeploymentToBeReady(ctx, f.TestLog(), f.Client, kutil.ObjectKeyFromObject(echoServerDeployment), 2*time.Minute))

			// todo check if the echo server can be pinged

			ginkgo.By("Delete aggregated installation")
			utils.ExpectNoError(f.Client.Delete(ctx, aggInst))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, aggInst, 2*time.Minute))

			// expect that the echo server deployment is already deleted or has an deletion timestamp
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, echoServerDeployment, 2*time.Minute))

			// expect that the nginx deployment is already deleted or has an deletion timestamp
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, nginxDeployment, 2*time.Minute))
		})
	})

}

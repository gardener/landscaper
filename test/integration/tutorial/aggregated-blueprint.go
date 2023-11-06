// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package tutorial

import (
	"context"
	"path/filepath"
	"time"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

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

func AggregatedBlueprintForNewReconcile(f *framework.Framework) {
	_ = Describe("AggregatedBlueprint", func() {
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

		It("should deploy a nginx ingress controller and a echo-server together with an aggregated blueprint", func() {
			var (
				tutorialResourcesRootDir = filepath.Join(f.RootPath, "/docs/tutorials/resources/aggregated")
				targetResource           = filepath.Join(tutorialResourcesRootDir, "my-target.yaml")
				importResource           = filepath.Join(tutorialResourcesRootDir, "configmap.yaml")
				nginxInstResource        = filepath.Join(tutorialResourcesRootDir, "installation.yaml")
			)

			defer ctx.Done()

			By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			utils.ExpectNoError(utils.ReadResourceFromFile(target, targetResource))
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create ConfigMap with imports for the installation")
			cm := &corev1.ConfigMap{}
			utils.ExpectNoError(utils.ReadResourceFromFile(cm, importResource))
			cm.SetNamespace(state.Namespace)
			cm.Data["namespace"] = state.Namespace
			utils.ExpectNoError(state.Create(ctx, cm))

			By("Create Aggregated Installation")
			aggInst := &lsv1alpha1.Installation{}
			Expect(utils.ReadResourceFromFile(aggInst, nginxInstResource)).To(Succeed())
			aggInst.SetNamespace(state.Namespace)
			lsv1alpha1helper.SetOperation(&aggInst.ObjectMeta, lsv1alpha1.ReconcileOperation)
			utils.ExpectNoError(state.Create(ctx, aggInst))

			// wait for installation to finish
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, aggInst, lsv1alpha1.InstallationPhases.Succeeded, 4*time.Minute))

			subInstallations, err := lsutils.GetSubInstallationsOfInstallation(ctx, f.Client, aggInst)
			utils.ExpectNoError(err)
			Expect(subInstallations).To(HaveLen(2))
			Expect(subInstallations[0].Status.InstallationPhase).To(Equal(lsv1alpha1.InstallationPhases.Succeeded))
			Expect(subInstallations[0].Status.JobIDFinished).To(Equal(subInstallations[0].Status.JobID))
			Expect(subInstallations[1].Status.InstallationPhase).To(Equal(lsv1alpha1.InstallationPhases.Succeeded))
			Expect(subInstallations[1].Status.JobIDFinished).To(Equal(subInstallations[1].Status.JobID))

			// expect that the nginx deployment is successfully running
			nginxDeployment := &appsv1.Deployment{}
			nginxDeployment.Name = "test-ingress-nginx-controller"
			nginxDeployment.Namespace = state.Namespace
			utils.ExpectNoError(utils.WaitForDeploymentToBeReady(ctx, f.TestLog(), f.Client, kutil.ObjectKeyFromObject(nginxDeployment), 4*time.Minute))

			// expect that the echo server deployment is successfully running
			echoServerDeployment := &appsv1.Deployment{}
			echoServerDeployment.Name = "echo-server"
			echoServerDeployment.Namespace = state.Namespace
			utils.ExpectNoError(utils.WaitForDeploymentToBeReady(ctx, f.TestLog(), f.Client, kutil.ObjectKeyFromObject(echoServerDeployment), 2*time.Minute))

			// todo check if the echo server can be pinged

			By("Delete aggregated installation")
			utils.ExpectNoError(f.Client.Delete(ctx, aggInst))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, aggInst, 2*time.Minute))

			// expect that the echo server deployment is already deleted or has an deletion timestamp
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, echoServerDeployment, 2*time.Minute))

			// expect that the nginx deployment is already deleted or has an deletion timestamp
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, nginxDeployment, 2*time.Minute))
		})
	})
}

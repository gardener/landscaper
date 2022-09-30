// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package tutorial

import (
	"context"
	"path/filepath"
	"time"

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

func NginxIngressTestForNewReconcile(f *framework.Framework) {
	_ = Describe("SimpleNginxTest", func() {
		state := f.Register()

		It("should deploy a nginx ingress controller", func() {
			var (
				tutorialResourcesRootDir = filepath.Join(f.RootPath, "/docs/tutorials/resources/ingress-nginx")
				targetResource           = filepath.Join(tutorialResourcesRootDir, "my-target.yaml")
				importResource           = filepath.Join(tutorialResourcesRootDir, "configmap.yaml")
				instResource             = filepath.Join(tutorialResourcesRootDir, "installation.yaml")
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
			utils.ExpectNoError(utils.ReadResourceFromFile(cm, importResource))
			cm.SetNamespace(state.Namespace)
			cm.Data["namespace"] = state.Namespace
			utils.ExpectNoError(state.Create(ctx, cm))

			By("Create Installation")
			inst := &lsv1alpha1.Installation{}
			Expect(utils.ReadResourceFromFile(inst, instResource)).To(Succeed())
			inst.SetNamespace(state.Namespace)
			lsv1alpha1helper.SetOperation(&inst.ObjectMeta, lsv1alpha1.ReconcileOperation)
			utils.ExpectNoError(state.Create(ctx, inst))

			// wait for installation to finish
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			deployItems, err := lsutils.GetDeployItemsOfInstallation(ctx, f.Client, inst)
			utils.ExpectNoError(err)
			Expect(deployItems).To(HaveLen(1))
			Expect(deployItems[0].Status.DeployItemPhase).To(Equal(lsv1alpha1.DeployItemPhaseSucceeded))
			Expect(deployItems[0].Status.JobIDFinished).To(Equal(deployItems[0].Status.GetJobID()))

			// expect that the nginx deployment is successfully running
			nginxIngressDeploymentName := "test-ingress-nginx-controller"
			nginxIngressObjectKey := kutil.ObjectKey(nginxIngressDeploymentName, state.Namespace)
			utils.ExpectNoError(utils.WaitForDeploymentToBeReady(ctx, f.TestLog(), f.Client, nginxIngressObjectKey, 2*time.Minute))

			// Update the installation to use the next component descriptor version (see
			// docs/tutorials/resources/ingress-nginx-upgrade/component-descriptor.yaml) with the next nginx version.
			By("Upgrade installation")
			instKey := kutil.ObjectKey(inst.Name, inst.Namespace)
			inst = &lsv1alpha1.Installation{}
			utils.ExpectNoError(f.Client.Get(ctx, instKey, inst))
			inst.Spec.ComponentDescriptor.Reference.Version = "v0.3.3"
			lsv1alpha1helper.SetOperation(&inst.ObjectMeta, lsv1alpha1.ReconcileOperation)
			err = f.Client.Update(ctx, inst)
			utils.ExpectNoError(err)

			// wait for installation to finish
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			deployItems, err = lsutils.GetDeployItemsOfInstallation(ctx, f.Client, inst)
			utils.ExpectNoError(err)
			Expect(deployItems).To(HaveLen(1))
			Expect(deployItems[0].Status.DeployItemPhase).To(Equal(lsv1alpha1.DeployItemPhaseSucceeded))
			Expect(deployItems[0].Status.JobIDFinished).To(Equal(deployItems[0].Status.GetJobID()))

			// expect that the nginx deployment is successfully running
			utils.ExpectNoError(utils.WaitForDeploymentToBeReady(ctx, f.TestLog(), f.Client, nginxIngressObjectKey, 2*time.Minute))

			// check the new chart version in label "helm.sh/chart" of the deployment
			deploy := &appsv1.Deployment{}
			utils.ExpectNoError(f.Client.Get(ctx, nginxIngressObjectKey, deploy))
			Expect(deploy.GetLabels()).To(HaveKeyWithValue("helm.sh/chart", "ingress-nginx-4.0.18"))

			By("Delete installation")
			utils.ExpectNoError(f.Client.Delete(ctx, inst))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, inst, 2*time.Minute))

			// expect that the nginx deployment will be deleted
			nginxDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nginxIngressObjectKey.Name,
					Namespace: nginxIngressObjectKey.Namespace,
				},
			}
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, nginxDeployment, 2*time.Minute))
		})
	})

	_ = Describe("LocalIngressNginxTest", func() {
		state := f.Register()

		It("should deploy a nginx ingress controller with local artifacts", func() {
			var (
				tutorialResourcesRootDir = filepath.Join(f.RootPath, "/docs/tutorials/resources/local-ingress-nginx")
				targetResource           = filepath.Join(tutorialResourcesRootDir, "my-target.yaml")
				importResource           = filepath.Join(tutorialResourcesRootDir, "configmap.yaml")
				instResource             = filepath.Join(tutorialResourcesRootDir, "installation.yaml")
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
			utils.ExpectNoError(utils.ReadResourceFromFile(cm, importResource))
			cm.SetNamespace(state.Namespace)
			cm.Data["namespace"] = state.Namespace
			utils.ExpectNoError(state.Create(ctx, cm))

			By("Create Installation")
			inst := &lsv1alpha1.Installation{}
			Expect(utils.ReadResourceFromFile(inst, instResource)).To(Succeed())
			inst.SetNamespace(state.Namespace)
			lsv1alpha1helper.SetOperation(&inst.ObjectMeta, lsv1alpha1.ReconcileOperation)
			utils.ExpectNoError(state.Create(ctx, inst))

			// wait for installation to finish
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			deployItems, err := lsutils.GetDeployItemsOfInstallation(ctx, f.Client, inst)
			utils.ExpectNoError(err)
			Expect(deployItems).To(HaveLen(1))
			Expect(deployItems[0].Status.DeployItemPhase).To(Equal(lsv1alpha1.DeployItemPhaseSucceeded))
			Expect(deployItems[0].Status.JobIDFinished).To(Equal(deployItems[0].Status.GetJobID()))

			// expect that the nginx deployment is successfully running
			nginxIngressDeploymentName := "test-ingress-nginx-controller"
			nginxIngressObjectKey := kutil.ObjectKey(nginxIngressDeploymentName, state.Namespace)
			utils.ExpectNoError(utils.WaitForDeploymentToBeReady(ctx, f.TestLog(), f.Client, nginxIngressObjectKey, 2*time.Minute))

			// check
			deploy := &appsv1.Deployment{}
			utils.ExpectNoError(f.Client.Get(ctx, nginxIngressObjectKey, deploy))
			Expect(deploy.Spec.Template.Spec.Containers[0].LivenessProbe.FailureThreshold == 4).To(BeTrue())

			// delete
			By("Delete installation")
			utils.ExpectNoError(f.Client.Delete(ctx, inst))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, inst, 2*time.Minute))

			// expect that the nginx deployment will be deleted
			nginxDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nginxIngressObjectKey.Name,
					Namespace: nginxIngressObjectKey.Namespace,
				},
			}
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, nginxDeployment, 2*time.Minute))
		})
	})
}

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

// ExternalJSONSchemaTest tests the jsonschema tutorial.
func ExternalJSONSchemaTest(f *framework.Framework) {
	_ = ginkgo.Describe("ExternalJSONSchemaTest", func() {
		dumper := f.Register()

		ginkgo.It("should deploy an echo server with resources defined by an external jsonschema", func() {
			var (
				tutorialResourcesRootDir = filepath.Join(f.RootPath, "/docs/tutorials/resources/external-jsonschema")
				targetResource           = filepath.Join(tutorialResourcesRootDir, "my-target.yaml")
				importResource           = filepath.Join(tutorialResourcesRootDir, "configmap.yaml")
				instResource             = filepath.Join(tutorialResourcesRootDir, "installation.yaml")
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
			utils.ExpectNoError(state.Create(ctx, f.Client, cm))

			ginkgo.By("Create Installation")
			inst := &lsv1alpha1.Installation{}
			inst.SetNamespace(state.Namespace)
			g.Expect(utils.ReadResourceFromFile(inst, instResource)).To(g.Succeed())
			utils.ExpectNoError(state.Create(ctx, f.Client, inst))

			// wait for installation to finish
			utils.ExpectNoError(utils.WaitForInstallationToBeInPhase(ctx, f.Client, inst, lsv1alpha1.ComponentPhaseSucceeded, 2*time.Minute))

			deployItems, err := utils.GetDeployItemsOfInstallation(ctx, f.Client, inst)
			utils.ExpectNoError(err)
			g.Expect(deployItems).To(g.HaveLen(1))
			g.Expect(deployItems[0].Status.Phase).To(g.Equal(lsv1alpha1.ExecutionPhaseSucceeded))

			// todo: make namespace configurable for deployed resources
			// expect that the echo server deployment is successfully running
			echoServerDeploymentName := "echo-server"
			echoServerDeploymentObjectKey := kutil.ObjectKey(echoServerDeploymentName, "default")
			utils.ExpectNoError(utils.WaitForDeploymentToBeReady(ctx, f.TestLog(), f.Client, echoServerDeploymentObjectKey, 2*time.Minute))
			// expect that the deployment has the correct resource requests and limits
			echoServerDeploy := &appsv1.Deployment{}
			utils.ExpectNoError(f.Client.Get(ctx, echoServerDeploymentObjectKey, echoServerDeploy))
			g.Expect(echoServerDeploy.Spec.Template.Spec.Containers).To(g.HaveLen(1))
			g.Expect(echoServerDeploy.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String()).To(g.Equal("50Mi"))
			g.Expect(echoServerDeploy.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String()).To(g.Equal("100Mi"))

			ginkgo.By("Delete installation")
			utils.ExpectNoError(f.Client.Delete(ctx, inst))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, inst, 2*time.Minute))

			// expect that the nginx deployment is alread deleted or has an deletion timestamp
			err = f.Client.Get(ctx, echoServerDeploymentObjectKey, echoServerDeploy)
			if err != nil && !apierrors.IsNotFound(err) {
				utils.ExpectNoError(err)
			} else if err == nil {
				g.Expect(echoServerDeploy.DeletionTimestamp.IsZero()).To(g.BeTrue())
			}
		})
	})
}

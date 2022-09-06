// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployers

import (
	"context"
	"path"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"

	manifestv1alpha1 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	manifest "github.com/gardener/landscaper/pkg/deployer/manifest"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func ManifestDeployerTestsForNewReconcile(f *framework.Framework) {
	Describe("Manifest Deployer", func() {

		var (
			state       = f.Register()
			exampleDir  = path.Join(f.RootPath, "examples", "deploy-items")
			testDataDir = path.Join(f.RootPath, "test", "testdata")

			ctx context.Context
		)

		const (
			timeout = 2 * time.Minute
		)

		BeforeEach(func() {
			ctx = context.Background()
		})

		AfterEach(func() {
			defer ctx.Done()
		})

		It("should deploy Kubernetes objects through their v1alpha2 manifests", func() {
			By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			target.Name = "my-cluster-target"
			target.Namespace = state.Namespace
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			di := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(utils.ReadResourceFromFile(di, path.Join(exampleDir, "40-DeployItem-Manifest-secret.yaml")))
			di.SetName("")
			di.SetGenerateName("secret-manifest-")
			di.SetNamespace(state.Namespace)
			di.Spec.Target = &lsv1alpha1.ObjectReference{
				Name:      target.Name,
				Namespace: target.Namespace,
			}

			By("Create Manifest (v1alpha2) deploy item")
			utils.ExpectNoError(state.Create(ctx, di))
			// Set a new jobID to trigger a reconcile of the deploy item
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
			Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())
			utils.ExpectNoError(lsutils.WaitForDeployItemToFinish(ctx, f.Client, di, lsv1alpha1.DeployItemPhaseSucceeded, timeout))

			By("Check presence of Kubernetes Objects")
			config := &manifestv1alpha2.ProviderConfiguration{}
			manifestDecoder := serializer.NewCodecFactory(manifest.Scheme).UniversalDecoder()
			_, _, err = manifestDecoder.Decode(di.Spec.Configuration.Raw, nil, config)
			Expect(err).ToNot(HaveOccurred())

			var objectsToBeDeleted []*unstructured.Unstructured

			for _, m := range config.Manifests {
				manifestObject := &unstructured.Unstructured{}
				_, _, err = manifestDecoder.Decode(m.Manifest.Raw, nil, manifestObject)
				Expect(err).ToNot(HaveOccurred())

				apiObject := &unstructured.Unstructured{}
				apiObject.GetObjectKind().SetGroupVersionKind(manifestObject.GetObjectKind().GroupVersionKind())
				key := kutil.ObjectKey(manifestObject.GetName(), manifestObject.GetNamespace())
				// if this returns without error it means the object exists in the API and thus the manifest has been applied
				utils.ExpectNoError(f.Client.Get(ctx, key, apiObject))

				objectsToBeDeleted = append(objectsToBeDeleted, manifestObject)
			}

			By("Delete Manifest (v1alpha2) deploy item")
			utils.ExpectNoError(state.Client.Delete(ctx, di))
			// Set a new jobID to trigger a reconcile of the deploy item
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
			Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, state.Client, di, timeout))

			By("Check successful deletion Kubernetes objects")
			for _, o := range objectsToBeDeleted {
				key := kutil.ObjectKey(o.GetName(), o.GetNamespace())
				apiObject := &unstructured.Unstructured{}
				apiObject.GetObjectKind().SetGroupVersionKind(o.GetObjectKind().GroupVersionKind())
				err = f.Client.Get(ctx, key, apiObject)
				Expect(err).NotTo(BeNil())
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}
		})

		It("should deploy Kubernetes objects through their v1alpha1 manifests", func() {
			By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			target.Name = "my-cluster-target"
			target.Namespace = state.Namespace
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			di := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(utils.ReadResourceFromFile(di, path.Join(testDataDir, "00-DeployItem-Manifest-v1alpha1.yaml")))
			di.SetName("")
			di.SetGenerateName("v1alpha1-manifest-")
			di.SetNamespace(state.Namespace)
			di.Spec.Target = &lsv1alpha1.ObjectReference{
				Name:      target.Name,
				Namespace: target.Namespace,
			}

			By("Create Manifest (v1alpha1) deploy item")
			utils.ExpectNoError(state.Create(ctx, di))
			// Set a new jobID to trigger a reconcile of the deploy item
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
			Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())
			utils.ExpectNoError(lsutils.WaitForDeployItemToFinish(ctx, f.Client, di, lsv1alpha1.DeployItemPhaseSucceeded, timeout))

			By("Check presence of Kubernetes objects")
			config := &manifestv1alpha1.ProviderConfiguration{}
			manifestDecoder := serializer.NewCodecFactory(manifest.Scheme).UniversalDecoder()
			_, _, err = manifestDecoder.Decode(di.Spec.Configuration.Raw, nil, config)
			Expect(err).ToNot(HaveOccurred())

			for _, m := range config.Manifests {
				manifestObject := &unstructured.Unstructured{}
				_, _, err = manifestDecoder.Decode(m.Raw, nil, manifestObject)
				Expect(err).ToNot(HaveOccurred())

				apiObject := &unstructured.Unstructured{}
				apiObject.GetObjectKind().SetGroupVersionKind(manifestObject.GetObjectKind().GroupVersionKind())
				key := kutil.ObjectKey(manifestObject.GetName(), manifestObject.GetNamespace())
				// if this returns without error it means the object exists in the API and thus the manifest has been applied
				utils.ExpectNoError(f.Client.Get(ctx, key, apiObject))
			}

			By("Delete Manifest (v1alpha1) deploy item")
			utils.ExpectNoError(f.Client.Delete(ctx, di))
			// Set a new jobID to trigger a reconcile of the deploy item
			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
			Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, di, timeout))
		})
	})
}

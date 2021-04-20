// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployers

import (
	"context"
	"path"
	"time"

	"github.com/onsi/ginkgo"
	g "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"

	manifestv1alpha1 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	manifest "github.com/gardener/landscaper/pkg/deployer/manifest"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func ManifestDeployerTests(f *framework.Framework) {
	ginkgo.Describe("Manifest Deployer", func() {

		var (
			dumper      = f.Register()
			exampleDir  = path.Join(f.RootPath, "examples", "deploy-items")
			testDataDir = path.Join(f.RootPath, "test", "testdata")

			ctx     context.Context
			state   *envtest.State
			cleanup framework.CleanupFunc
		)

		const (
			timeout = 2 * time.Minute
		)

		ginkgo.BeforeEach(func() {
			ctx = context.Background()
			var err error
			state, cleanup, err = f.NewState(ctx)
			utils.ExpectNoError(err)
			dumper.AddNamespaces(state.Namespace)
		})

		ginkgo.AfterEach(func() {
			defer ctx.Done()
			g.Expect(cleanup(ctx)).ToNot(g.HaveOccurred())
		})

		ginkgo.It("should deploy Kubernetes objects through their v1alpha2 manifests", func() {
			ginkgo.By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			target.Name = "my-cluster-target"
			target.Namespace = state.Namespace
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, f.Client, target))

			di := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(utils.ReadResourceFromFile(di, path.Join(exampleDir, "40-DeployItem-Manifest-secret.yaml")))
			di.SetName("")
			di.SetGenerateName("secret-manifest-")
			di.SetNamespace(state.Namespace)
			di.Spec.Target = &lsv1alpha1.ObjectReference{
				Name:      target.Name,
				Namespace: target.Namespace,
			}

			ginkgo.By("Create Manifest (v1alpha2) deploy item")
			utils.ExpectNoError(state.Create(ctx, f.Client, di))
			utils.ExpectNoError(lsutils.WaitForDeployItemToBeInPhase(ctx, f.Client, di, lsv1alpha1.ExecutionPhaseSucceeded, timeout))

			ginkgo.By("Check presence of Kubernetes Objects")
			config := &manifestv1alpha2.ProviderConfiguration{}
			manifestDecoder := serializer.NewCodecFactory(manifest.ManifestScheme).UniversalDecoder()
			_, _, err = manifestDecoder.Decode(di.Spec.Configuration.Raw, nil, config)
			g.Expect(err).ToNot(g.HaveOccurred())

			var objectsToBeDeleted []*unstructured.Unstructured

			for _, m := range config.Manifests {
				manifestObject := &unstructured.Unstructured{}
				_, _, err = manifestDecoder.Decode(m.Manifest.Raw, nil, manifestObject)
				g.Expect(err).ToNot(g.HaveOccurred())

				apiObject := &unstructured.Unstructured{}
				apiObject.GetObjectKind().SetGroupVersionKind(manifestObject.GetObjectKind().GroupVersionKind())
				key := kutil.ObjectKey(manifestObject.GetName(), manifestObject.GetNamespace())
				// if this returns without error it means the object exists in the API and thus the manifest has been applied
				utils.ExpectNoError(f.Client.Get(ctx, key, apiObject))

				objectsToBeDeleted = append(objectsToBeDeleted, manifestObject)
			}

			ginkgo.By("Delete Manifest (v1alpha2) deploy item")
			utils.ExpectNoError(f.Client.Delete(ctx, di))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, di, timeout))

			ginkgo.By("Check successful deletion Kubernetes objects")
			for _, o := range objectsToBeDeleted {
				key := kutil.ObjectKey(o.GetName(), o.GetNamespace())
				apiObject := &unstructured.Unstructured{}
				apiObject.GetObjectKind().SetGroupVersionKind(o.GetObjectKind().GroupVersionKind())
				err = f.Client.Get(ctx, key, apiObject)
				g.Expect(err).NotTo(g.BeNil())
				g.Expect(apierrors.IsNotFound(err)).To(g.BeTrue())
			}
		})

		ginkgo.It("should deploy Kubernetes objects through their v1alpha1 manifests", func() {
			ginkgo.By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			target.Name = "my-cluster-target"
			target.Namespace = state.Namespace
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, f.Client, target))

			di := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(utils.ReadResourceFromFile(di, path.Join(testDataDir, "00-DeployItem-Manifest-v1alpha1.yaml")))
			di.SetName("")
			di.SetGenerateName("v1alpha1-manifest-")
			di.SetNamespace(state.Namespace)
			di.Spec.Target = &lsv1alpha1.ObjectReference{
				Name:      target.Name,
				Namespace: target.Namespace,
			}

			ginkgo.By("Create Manifest (v1alpha1) deploy item")
			utils.ExpectNoError(state.Create(ctx, f.Client, di))
			utils.ExpectNoError(lsutils.WaitForDeployItemToBeInPhase(ctx, f.Client, di, lsv1alpha1.ExecutionPhaseSucceeded, timeout))

			ginkgo.By("Check presence of Kubernetes objects")
			config := &manifestv1alpha1.ProviderConfiguration{}
			manifestDecoder := serializer.NewCodecFactory(manifest.ManifestScheme).UniversalDecoder()
			_, _, err = manifestDecoder.Decode(di.Spec.Configuration.Raw, nil, config)
			g.Expect(err).ToNot(g.HaveOccurred())

			for _, m := range config.Manifests {
				manifestObject := &unstructured.Unstructured{}
				_, _, err = manifestDecoder.Decode(m.Raw, nil, manifestObject)
				g.Expect(err).ToNot(g.HaveOccurred())

				apiObject := &unstructured.Unstructured{}
				apiObject.GetObjectKind().SetGroupVersionKind(manifestObject.GetObjectKind().GroupVersionKind())
				key := kutil.ObjectKey(manifestObject.GetName(), manifestObject.GetNamespace())
				// if this returns without error it means the object exists in the API and thus the manifest has been applied
				utils.ExpectNoError(f.Client.Get(ctx, key, apiObject))
			}

			ginkgo.By("Delete Manifest (v1alpha1) deploy item")
			utils.ExpectNoError(f.Client.Delete(ctx, di))
			utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, di, timeout))
		})
	})
}

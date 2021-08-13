// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/gardener/component-cli/pkg/commands/componentarchive/input"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/pointer"

	"github.com/gardener/landscaper/apis/mediatype"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/framework"
	testutils "github.com/gardener/landscaper/test/utils"
)

func RegistryTest(f *framework.Framework) {
	if !f.IsRegistryEnabled() {
		f.Log().Log("No registry configured skipping the registry tests...")
		return
	}

	_ = ginkgo.Describe("RegistryTest", func() {

		var (
			state = f.Register()
			ctx   context.Context
		)

		ginkgo.BeforeEach(func() {
			ctx = context.Background()
		})

		ginkgo.AfterEach(func() {
			ctx.Done()
		})

		ginkgo.It("should upload a component descriptor and blueprint to a private registry and install that blueprint", func() {
			var (
				tutorialResourcesRootDir = filepath.Join(f.RootPath, "/docs/tutorials/resources/local-ingress-nginx")
				targetResource           = filepath.Join(tutorialResourcesRootDir, "my-target.yaml")
				importResource           = filepath.Join(tutorialResourcesRootDir, "configmap.yaml")
				instResource             = filepath.Join(tutorialResourcesRootDir, "installation.yaml")

				componentName    = "example.com/test-ingress"
				componentVersion = "v0.0.1"
			)

			ginkgo.By("upload component descriptor, blueprint and helm chart")
			cd := buildAndUploadComponentDescriptorWithArtifacts(ctx, f, componentName, componentVersion)
			repoCtx := cd.GetEffectiveRepositoryContext()

			ginkgo.By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			testutils.ExpectNoError(testutils.ReadResourceFromFile(target, targetResource))
			target, err := testutils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, false)
			testutils.ExpectNoError(err)
			testutils.ExpectNoError(state.Create(ctx, f.Client, target))

			ginkgo.By("Create ConfigMap with imports for the installation")
			cm := &corev1.ConfigMap{}
			cm.SetNamespace(state.Namespace)
			testutils.ExpectNoError(testutils.ReadResourceFromFile(cm, importResource))
			cm.Data["namespace"] = state.Namespace
			testutils.ExpectNoError(state.Create(ctx, f.Client, cm))

			ginkgo.By("Create Installation")
			inst := &lsv1alpha1.Installation{}
			gomega.Expect(testutils.ReadResourceFromFile(inst, instResource)).To(gomega.Succeed())
			inst.SetNamespace(state.Namespace)
			inst.Spec.ComponentDescriptor = &lsv1alpha1.ComponentDescriptorDefinition{
				Reference: &lsv1alpha1.ComponentDescriptorReference{
					RepositoryContext: repoCtx,
					ComponentName:     componentName,
					Version:           componentVersion,
				},
			}
			inst.Spec.Blueprint.Reference.ResourceName = "my-blueprint"

			testutils.ExpectNoError(state.Create(ctx, f.Client, inst))

			// wait for installation to finish
			testutils.ExpectNoError(lsutils.WaitForInstallationToBeHealthy(ctx, f.Client, inst, 2*time.Minute))

			deployItems, err := lsutils.GetDeployItemsOfInstallation(ctx, f.Client, inst)
			testutils.ExpectNoError(err)
			gomega.Expect(deployItems).To(gomega.HaveLen(1))
			gomega.Expect(deployItems[0].Status.Phase).To(gomega.Equal(lsv1alpha1.ExecutionPhaseSucceeded))

			// expect that the nginx deployment is successfully running
			nginxIngressDeploymentName := "test-ingress-nginx-controller"
			nginxIngressObjectKey := kutil.ObjectKey(nginxIngressDeploymentName, state.Namespace)
			testutils.ExpectNoError(testutils.WaitForDeploymentToBeReady(ctx, f.TestLog(), f.Client, nginxIngressObjectKey, 2*time.Minute))

			ginkgo.By("Delete installation")
			testutils.ExpectNoError(f.Client.Delete(ctx, inst))
			testutils.ExpectNoError(testutils.WaitForObjectDeletion(ctx, f.Client, inst, 2*time.Minute))

			// expect that the nginx deployment is already deleted or has an deletion timestamp
			nginxDeployment := &appsv1.Deployment{}
			err = f.Client.Get(ctx, nginxIngressObjectKey, nginxDeployment)
			if err != nil && !apierrors.IsNotFound(err) {
				testutils.ExpectNoError(err)
			} else if err == nil {
				gomega.Expect(nginxDeployment.DeletionTimestamp.IsZero()).To(gomega.BeTrue())
			}
		})
	})
}

func buildAndUploadComponentDescriptorWithArtifacts(ctx context.Context, f *framework.Framework, name, version string) *cdv2.ComponentDescriptor {
	// define component descriptor
	var (
		tutorialResourcesRootDir = filepath.Join(f.RootPath, "/docs/tutorials/resources/local-ingress-nginx")
		blueprintDir             = filepath.Join(tutorialResourcesRootDir, "blueprint")
		helmChartDir             = filepath.Join(tutorialResourcesRootDir, "chart")
		cd                       = &cdv2.ComponentDescriptor{}
		fs                       = memoryfs.New()
	)
	cd.Name = name
	cd.Version = version
	cd.Provider = cdv2.InternalProvider
	repoCtx := cdv2.OCIRegistryRepository{
		ObjectType: cdv2.ObjectType{
			Type: cdv2.OCIRegistryType,
		},
		BaseURL:              f.RegistryBasePath,
		ComponentNameMapping: cdv2.OCIRegistryURLPathMapping,
	}
	testutils.ExpectNoError(cdv2.InjectRepositoryContext(cd, &repoCtx))
	testutils.ExpectNoError(fs.MkdirAll("blobs", os.ModePerm))

	// gzip and add helm chart
	helmInput := input.BlobInput{
		Type:             input.DirInputType,
		Path:             helmChartDir,
		CompressWithGzip: pointer.BoolPtr(true),
	}
	blob, err := helmInput.Read(osfs.New(), "")
	testutils.ExpectNoError(err)
	file, err := fs.Create("blobs/chart")
	testutils.ExpectNoError(err)
	_, err = io.Copy(file, blob.Reader)
	testutils.ExpectNoError(err)
	testutils.ExpectNoError(file.Close())
	testutils.ExpectNoError(blob.Reader.Close())

	cd.Resources = append(cd.Resources, testutils.BuildLocalFilesystemResource("ingress-nginx-chart", "helm", input.MediaTypeGZip, "chart"))

	blueprintInput := input.BlobInput{
		Type:             input.DirInputType,
		Path:             blueprintDir,
		MediaType:        mediatype.NewBuilder(mediatype.BlueprintArtifactsLayerMediaTypeV1).Compression(mediatype.GZipCompression).String(),
		CompressWithGzip: pointer.BoolPtr(true),
	}
	blob, err = blueprintInput.Read(osfs.New(), "")
	testutils.ExpectNoError(err)
	defer blob.Reader.Close()
	file, err = fs.Create("blobs/bp")
	testutils.ExpectNoError(err)
	_, err = io.Copy(file, blob.Reader)
	testutils.ExpectNoError(err)
	testutils.ExpectNoError(file.Close())
	testutils.ExpectNoError(blob.Reader.Close())

	cd.Resources = append(cd.Resources, testutils.BuildLocalFilesystemResource("my-blueprint", mediatype.BlueprintType, blueprintInput.MediaType, "bp"))

	testutils.ExpectNoError(cdv2.DefaultComponent(cd))

	ca := ctf.NewComponentArchive(cd, fs)
	manifest, err := cdoci.NewManifestBuilder(f.OCICache, ca).Build(ctx)
	testutils.ExpectNoError(err)

	ref, err := cdoci.OCIRef(repoCtx, cd.Name, cd.Version)
	testutils.ExpectNoError(err)
	testutils.ExpectNoError(f.OCIClient.PushManifest(ctx, ref, manifest))
	return cd
}

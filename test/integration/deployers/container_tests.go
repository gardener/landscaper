// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployers

import (
	"context"
	"io"
	"os"
	"path"
	"time"

	"github.com/gardener/component-cli/pkg/commands/componentarchive/input"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/onsi/ginkgo"
	g "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"github.com/gardener/landscaper/apis/mediatype"
	"github.com/gardener/landscaper/pkg/deployer/container"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func ContainerDeployerTests(f *framework.Framework) {
	ginkgo.Describe("Container Deployer", func() {
		var (
			state      = f.Register()
			exampleDir = path.Join(f.RootPath, "examples/deploy-items")
			testdataFs vfs.FileSystem

			ctx context.Context
		)

		ginkgo.BeforeEach(func() {
			ctx = context.Background()
			var err error
			testdataFs, err = projectionfs.New(osfs.New(), path.Join(f.RootPath, "test/integration/deployers/testdata"))
			utils.ExpectNoError(err)
		})

		ginkgo.It("should run a simple docker image with a sleep command", func() {

			ginkgo.By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			target.Name = "my-cluster-target"
			target.Namespace = state.Namespace
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, f.Client, target))

			di := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(utils.ReadResourceFromFile(di, path.Join(exampleDir, "30-DeployItem-Container-sleep.yaml")))
			di.SetName("")
			di.SetGenerateName("container-sleep-")
			di.SetNamespace(state.Namespace)
			di.Spec.Target = &lsv1alpha1.ObjectReference{
				Name:      target.Name,
				Namespace: target.Namespace,
			}

			ginkgo.By("Create container deploy item")
			utils.ExpectNoError(state.Create(ctx, f.Client, di))
			utils.ExpectNoError(lsutils.WaitForDeployItemToBeInPhase(ctx, f.Client, di, lsv1alpha1.ExecutionPhaseSucceeded, 2*time.Minute))

			ginkgo.By("Delete container deploy item")
			utils.ExpectNoError(f.Client.Delete(ctx, di))
		})

		ginkgo.It("should detect when a image cannot be pulled and succeed when the deploy item is updated", func() {
			ginkgo.By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			target.Name = "my-cluster-target"
			target.Namespace = state.Namespace
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, f.Client, target))

			di := utils.BuildContainerDeployItem(&containerv1alpha1.ProviderConfiguration{
				Image: "example.com/some-invalid/image:v0.0.1",
			})
			di.SetName("")
			di.SetGenerateName("container-sleep-")
			di.SetNamespace(state.Namespace)
			di.Spec.Target = &lsv1alpha1.ObjectReference{
				Name:      target.Name,
				Namespace: target.Namespace,
			}

			ginkgo.By("Create erroneous container deploy item")
			utils.ExpectNoError(state.Create(ctx, f.Client, di))
			utils.ExpectNoError(lsutils.WaitForDeployItemToBeInPhase(ctx, f.Client, di, lsv1alpha1.ExecutionPhaseFailed, 2*time.Minute))

			ginkgo.By("update the DeployItem and set a valid image")
			utils.ExpectNoError(f.Client.Get(ctx, kutil.ObjectKey(di.Name, di.Namespace), di))
			updatedDi := utils.BuildContainerDeployItem(&containerv1alpha1.ProviderConfiguration{
				Image: "alpine",
			})
			di.Spec.Configuration = updatedDi.Spec.Configuration
			utils.ExpectNoError(f.Client.Update(ctx, di))

			utils.ExpectNoError(lsutils.WaitForDeployItemToBeInPhase(ctx, f.Client, di, lsv1alpha1.ExecutionPhaseSucceeded, 2*time.Minute))

			ginkgo.By("Delete container deploy item")
			utils.ExpectNoError(f.Client.Delete(ctx, di))
		})

		ginkgo.It("should export data", func() {
			ginkgo.By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			target.Name = "my-cluster-target"
			target.Namespace = state.Namespace
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, f.Client, target))

			di := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(utils.ReadResourceFromFile(di, path.Join(exampleDir, "31-DeployItem-Container-export.yaml")))
			di.SetName("")
			di.SetGenerateName("container-export-")
			di.SetNamespace(state.Namespace)
			di.Spec.Target = &lsv1alpha1.ObjectReference{
				Name:      target.Name,
				Namespace: target.Namespace,
			}

			ginkgo.By("Create container deploy item")
			utils.ExpectNoError(state.Create(ctx, f.Client, di))
			utils.ExpectNoError(lsutils.WaitForDeployItemToBeInPhase(ctx, f.Client, di, lsv1alpha1.ExecutionPhaseSucceeded, 2*time.Minute))

			// expect that the export contains a valid json with { "my-val": true }
			g.Expect(di.Status.ExportReference).ToNot(g.BeNil())
			exportData, err := lsutils.GetDeployItemExport(ctx, f.Client, di)
			utils.ExpectNoError(err)
			g.Expect(exportData).To(g.MatchJSON(`{ "my-val": true }`))

			ginkgo.By("Delete container deploy item")
			utils.ExpectNoError(f.Client.Delete(ctx, di))
		})

		ginkgo.It("should write and read data from the state", func() {
			ginkgo.By("Create Target for the installation")
			target := &lsv1alpha1.Target{}
			target.Name = "my-cluster-target"
			target.Namespace = state.Namespace
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, f.Client, target))

			di := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(utils.ReadResourceFromFile(di, path.Join(exampleDir, "32-DeployItem-Container-state.yaml")))
			di.SetName("")
			di.SetGenerateName("container-export-")
			di.SetNamespace(state.Namespace)
			di.Spec.Target = &lsv1alpha1.ObjectReference{
				Name:      target.Name,
				Namespace: target.Namespace,
			}

			ginkgo.By("Create container deploy item")
			utils.ExpectNoError(state.Create(ctx, f.Client, di))
			utils.ExpectNoError(lsutils.WaitForDeployItemToBeInPhase(ctx, f.Client, di, lsv1alpha1.ExecutionPhaseSucceeded, 2*time.Minute))

			// expect that the export contains a valid json with { "counter": 1 }
			g.Expect(di.Status.ExportReference).ToNot(g.BeNil())
			exportData, err := lsutils.GetDeployItemExport(ctx, f.Client, di)
			utils.ExpectNoError(err)
			g.Expect(exportData).To(g.MatchJSON(`{ "counter": 1 }`))

			ginkgo.By("Rerun the deployitem")
			metav1.SetMetaDataAnnotation(&di.ObjectMeta, lsv1alpha1.OperationAnnotation, string(lsv1alpha1.ReconcileOperation))
			utils.ExpectNoError(f.Client.Update(ctx, di))
			utils.ExpectNoError(lsutils.WaitForDeployItemToBeInPhase(ctx, f.Client, di, lsv1alpha1.ExecutionPhaseSucceeded, 2*time.Minute))
			// expect that the export contains a valid json with { "counter": 2 }
			g.Expect(di.Status.ExportReference).ToNot(g.BeNil())
			exportData, err = lsutils.GetDeployItemExport(ctx, f.Client, di)
			utils.ExpectNoError(err)
			g.Expect(exportData).To(g.MatchJSON(`{ "counter": 2 }`))

			ginkgo.By("Delete container deploy item")
			utils.ExpectNoError(f.Client.Delete(ctx, di))
		})

		ginkgo.It("should read data from the content path", func() {
			if !f.IsRegistryEnabled() {
				ginkgo.Skip("No registry configured skipping the registry tests...")
			}

			ginkgo.By("upload blueprint")
			cd := buildAndUploadComponentDescriptorWithArtifacts(ctx, f, testdataFs,
				"example.com/container-test-1", "v0.0.1", "/blueprints/container-deployer-example1")

			di, err := container.NewDeployItemBuilder().ProviderConfig(&containerv1alpha1.ProviderConfiguration{
				ComponentDescriptor: &lsv1alpha1.ComponentDescriptorDefinition{
					Reference: &lsv1alpha1.ComponentDescriptorReference{
						RepositoryContext: cd.GetEffectiveRepositoryContext(),
						ComponentName:     cd.GetName(),
						Version:           cd.GetVersion(),
					},
				},
				Blueprint: &lsv1alpha1.BlueprintDefinition{
					Reference: &lsv1alpha1.RemoteBlueprintReference{
						ResourceName: "my-blueprint",
					},
				},
				Image:   "alpine",
				Command: []string{"/bin/sh", "-c"},
				Args: []string{`
echo "{ \"my-val\": \"$(cat $CONTENT_PATH/file1)\" }" > $EXPORTS_PATH
`},
			}).Build()
			utils.ExpectNoError(err)
			di.SetName("")
			di.SetGenerateName("container-content-")
			di.SetNamespace(state.Namespace)

			ginkgo.By("Create container deploy item")
			utils.ExpectNoError(state.Create(ctx, f.Client, di))
			utils.ExpectNoError(lsutils.WaitForDeployItemToBeInPhase(ctx, f.Client, di, lsv1alpha1.ExecutionPhaseSucceeded, 2*time.Minute))

			// expect that the export contains a valid json with { "my-val": "val1" }
			g.Expect(di.Status.ExportReference).ToNot(g.BeNil())
			exportData, err := lsutils.GetDeployItemExport(ctx, f.Client, di)
			utils.ExpectNoError(err)
			g.Expect(exportData).To(g.MatchJSON(`{ "my-val": "val1" }`))

			ginkgo.By("Delete container deploy item")
			utils.ExpectNoError(f.Client.Delete(ctx, di))
		})

	})
}

func buildAndUploadComponentDescriptorWithArtifacts(
	ctx context.Context,
	f *framework.Framework,
	baseFs vfs.FileSystem,
	name, version string, blueprintDir string) *cdv2.ComponentDescriptor {
	// define component descriptor
	var (
		cd = &cdv2.ComponentDescriptor{}
		fs = memoryfs.New()
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
	utils.ExpectNoError(cdv2.InjectRepositoryContext(cd, &repoCtx))
	utils.ExpectNoError(fs.MkdirAll("blobs", os.ModePerm))

	blueprintInput := input.BlobInput{
		Type:             input.DirInputType,
		Path:             blueprintDir,
		MediaType:        mediatype.NewBuilder(mediatype.BlueprintArtifactsLayerMediaTypeV1).Compression(mediatype.GZipCompression).String(),
		CompressWithGzip: pointer.BoolPtr(true),
	}
	blob, err := blueprintInput.Read(baseFs, "")
	utils.ExpectNoError(err)
	defer blob.Reader.Close()
	file, err := fs.Create("blobs/bp")
	utils.ExpectNoError(err)
	_, err = io.Copy(file, blob.Reader)
	utils.ExpectNoError(err)
	utils.ExpectNoError(file.Close())
	utils.ExpectNoError(blob.Reader.Close())

	cd.Resources = append(cd.Resources, utils.BuildLocalFilesystemResource("my-blueprint", mediatype.BlueprintType, blueprintInput.MediaType, "bp"))

	utils.ExpectNoError(cdv2.DefaultComponent(cd))

	ca := ctf.NewComponentArchive(cd, fs)
	manifest, err := cdoci.NewManifestBuilder(f.OCICache, ca).Build(ctx)
	utils.ExpectNoError(err)

	ref, err := cdoci.OCIRef(repoCtx, cd.Name, cd.Version)
	utils.ExpectNoError(err)
	utils.ExpectNoError(f.OCIClient.PushManifest(ctx, ref, manifest))
	return cd
}

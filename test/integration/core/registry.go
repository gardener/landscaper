// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"

	"github.com/gardener/component-cli/pkg/commands/componentarchive/input"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"github.com/gardener/landscaper/apis/mediatype"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func RegistryTest(f *framework.Framework) {
	if !f.IsRegistryEnabled() {
		f.Log().Logln("No registry configured skipping the registry tests...")
		return
	}

	_ = Describe("RegistryTest", func() {

		var (
			state = f.Register()
			ctx   context.Context
		)

		BeforeEach(func() {
			ctx = context.Background()
		})

		AfterEach(func() {
			ctx.Done()
		})

		Context("Registry", func() {

			It("should upload a component descriptor and blueprint to a private registry and install that blueprint", func() {
				var (
					tutorialResourcesRootDir = filepath.Join(f.RootPath, "/docs/tutorials/resources/local-ingress-nginx")
					targetResource           = filepath.Join(tutorialResourcesRootDir, "my-target.yaml")
					importResource           = filepath.Join(tutorialResourcesRootDir, "configmap.yaml")
					instResource             = filepath.Join(tutorialResourcesRootDir, "installation.yaml")

					componentName    = "example.com/test-ingress"
					componentVersion = "v0.0.1"
				)

				By("upload component descriptor, blueprint and helm chart")
				cd := buildAndUploadNginxComponentDescriptorWithArtifacts(ctx, f, componentName, componentVersion)
				repoCtx := cd.GetEffectiveRepositoryContext()

				By("Create Target for the installation")
				target := &lsv1alpha1.Target{}
				utils.ExpectNoError(utils.ReadResourceFromFile(target, targetResource))
				target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, target.Name, f.RestConfig, false)
				utils.ExpectNoError(err)
				utils.ExpectNoError(state.Create(ctx, target))

				By("Create ConfigMap with imports for the installation")
				cm := &corev1.ConfigMap{}
				cm.SetNamespace(state.Namespace)
				utils.ExpectNoError(utils.ReadResourceFromFile(cm, importResource))
				cm.Data["namespace"] = state.Namespace
				utils.ExpectNoError(state.Create(ctx, cm))

				By("Create Installation")
				inst := &lsv1alpha1.Installation{}
				Expect(utils.ReadResourceFromFile(inst, instResource)).To(Succeed())
				inst.SetNamespace(state.Namespace)
				lsv1alpha1helper.SetOperation(&inst.ObjectMeta, lsv1alpha1.ReconcileOperation)
				inst.Spec.ComponentDescriptor = &lsv1alpha1.ComponentDescriptorDefinition{
					Reference: &lsv1alpha1.ComponentDescriptorReference{
						RepositoryContext: repoCtx,
						ComponentName:     componentName,
						Version:           componentVersion,
					},
				}
				inst.Spec.Blueprint.Reference.ResourceName = "my-blueprint"

				utils.ExpectNoError(state.Create(ctx, inst))

				// wait for installation to finish
				utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

				deployItems, err := lsutils.GetDeployItemsOfInstallation(ctx, f.Client, inst)
				utils.ExpectNoError(err)
				Expect(deployItems).To(HaveLen(1))
				Expect(deployItems[0].Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))

				// expect that the nginx deployment is successfully running
				nginxIngressDeploymentName := "test-ingress-nginx-controller"
				nginxIngressObjectKey := kutil.ObjectKey(nginxIngressDeploymentName, state.Namespace)
				utils.ExpectNoError(utils.WaitForDeploymentToBeReady(ctx, f.TestLog(), f.Client, nginxIngressObjectKey, 2*time.Minute))

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

		Context("ComponentVersionOverwrites", func() {

			It("should apply component version overwrites to directly and indirectly referenced components", func() {
				// define component descriptors
				/*
					Idea:
					'source' component references 'intermediate' and 'referencedSource'.
					'intermediate' references 'referencedSource'.
					The corresponding installations just contain a nested installation for each referenced cd.
					This means the 'source' installation will create a 'intermediate' and a 'referencedSource' installation,
					and the 'intermediate' installation will create another 'referencedSource' installation.
					The 'referenced*' installations just create a mock deployitem which contains their version (either 'source' or 'overwritten') in its state.

					To verify that the componentVersionOverwrites are working as expected, the version of 'referencedSource' will
					be overwritten with the version of 'referencedOverwritten'. If everything works, the cluster should contain
					two deployitems which were created by the 'referencedOverwritten' component, which can be detected via the version in their status (or spec).
				*/
				var (
					testdataDir = filepath.Join(f.RootPath, "test", "integration", "testdata", "componentoverwrites")

					sourceComponentDir = filepath.Join(testdataDir, "source_component")
					sourceBlueprintDir = filepath.Join(sourceComponentDir, "blueprint")
					sourceInstFile     = filepath.Join(sourceComponentDir, "installation.yaml")

					intermediateName         = "intermediate" // has to match reference in subinstallation in the blueprint
					intermediateComponentDir = filepath.Join(testdataDir, "intermediate_component")
					intermediateBlueprintDir = filepath.Join(intermediateComponentDir, "blueprint")

					referencedName                    = "referenced" // has to match reference in subinstallation in the blueprint
					referencedComponentDir            = filepath.Join(testdataDir, "referenced_component")
					referencedSourceComponentDir      = filepath.Join(referencedComponentDir, "source")
					referencedSourceBlueprintDir      = filepath.Join(referencedSourceComponentDir, "blueprint")
					referencedOverwrittenComponentDir = filepath.Join(referencedComponentDir, "overwritten")
					referencedOverwrittenBlueprintDir = filepath.Join(referencedOverwrittenComponentDir, "blueprint")
				)

				// create and upload CDs
				By("build and upload component descriptors with blueprints")
				referencedSourceDescription := cdDescription{
					name:         "example.com/overwrites/referenced",
					version:      "v0.1.0",
					blueprintDir: referencedSourceBlueprintDir,
				}
				referencedOverwrittenDescription := cdDescription{
					name:         referencedSourceDescription.name,
					version:      "v1.2.3",
					blueprintDir: referencedOverwrittenBlueprintDir,
				}
				intermediateDescription := cdDescription{
					name:         "example.com/overwrites/intermediate",
					version:      "v0.1.0",
					blueprintDir: intermediateBlueprintDir,
					cdRefs: []cdv2.ComponentReference{
						{
							Name:          referencedName,
							ComponentName: referencedSourceDescription.name,
							Version:       referencedSourceDescription.version,
						},
					},
				}
				sourceDescription := cdDescription{
					name:         "example.com/overwrites/source",
					version:      "v0.1.0",
					blueprintDir: sourceBlueprintDir,
					cdRefs: []cdv2.ComponentReference{
						{
							Name:          intermediateName,
							ComponentName: intermediateDescription.name,
							Version:       intermediateDescription.version,
						},
						{
							Name:          referencedName,
							ComponentName: referencedSourceDescription.name,
							Version:       referencedSourceDescription.version,
						},
					},
				}

				cds := buildAndUploadComponentDescriptorsWithBlueprints(ctx, f, sourceDescription, intermediateDescription, referencedSourceDescription, referencedOverwrittenDescription)
				repoCtx := cds[0].GetEffectiveRepositoryContext()

				By("create componentVersionOverwrite")
				cvoName := "cvo"
				cvo := &lsv1alpha1.ComponentVersionOverwrites{
					Overwrites: lsv1alpha1.ComponentVersionOverwriteList{
						{
							Source: lsv1alpha1.ComponentVersionOverwriteReference{
								RepositoryContext: repoCtx,
								ComponentName:     referencedSourceDescription.name,
								Version:           referencedSourceDescription.version,
							},
							Substitution: lsv1alpha1.ComponentVersionOverwriteReference{
								Version: referencedOverwrittenDescription.version,
							},
						},
					},
				}
				cvo.SetName(cvoName)
				cvo.SetNamespace(state.Namespace)
				utils.ExpectNoError(state.Create(ctx, cvo))

				By("patch context to reference ComponentVersionOverwrites")
				lsCtx := &lsv1alpha1.Context{}
				utils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKey(lsv1alpha1.DefaultContextName, state.Namespace), lsCtx))
				lsCtxOld := lsCtx.DeepCopy()
				lsCtx.ComponentVersionOverwritesReference = cvoName
				utils.ExpectNoError(state.Client.Patch(ctx, lsCtx, client.MergeFrom(lsCtxOld)))

				By("create installation")
				inst := &lsv1alpha1.Installation{}
				Expect(utils.ReadResourceFromFile(inst, sourceInstFile)).To(Succeed())
				inst.SetNamespace(state.Namespace)
				lsv1alpha1helper.SetOperation(&inst.ObjectMeta, lsv1alpha1.ReconcileOperation)
				inst.Spec.ComponentDescriptor = &lsv1alpha1.ComponentDescriptorDefinition{
					Reference: &lsv1alpha1.ComponentDescriptorReference{
						RepositoryContext: repoCtx,
						ComponentName:     sourceDescription.name,
						Version:           sourceDescription.version,
					},
				}
				inst.Spec.Blueprint.Reference.ResourceName = "blueprint"
				utils.ExpectNoError(state.Create(ctx, inst))

				// wait for installation to finish
				utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, state.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

				By("fetch subinstallations of source installation")
				utils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst)) // refresh installation for updated status
				var sourceIntermediateSubinst *lsv1alpha1.Installation
				var sourceReferencedSubinst *lsv1alpha1.Installation
				Expect(inst.Status.InstallationReferences).To(HaveLen(2))
				for _, subInstRef := range inst.Status.InstallationReferences {
					switch subInstRef.Name {
					case "intermediate":
						sourceIntermediateSubinst = &lsv1alpha1.Installation{}
						utils.ExpectNoError(state.Client.Get(ctx, subInstRef.Reference.NamespacedName(), sourceIntermediateSubinst))
					case "referenced":
						sourceReferencedSubinst = &lsv1alpha1.Installation{}
						utils.ExpectNoError(state.Client.Get(ctx, subInstRef.Reference.NamespacedName(), sourceReferencedSubinst))
					default:
						Fail(fmt.Sprintf("unexpected subinstallation: %s", subInstRef.Name))
					}
				}
				Expect(sourceIntermediateSubinst).ToNot(BeNil())
				Expect(sourceReferencedSubinst).ToNot(BeNil())

				By("fetch subinstallations of intermediate installation")
				Expect(sourceIntermediateSubinst.Status.InstallationReferences).To(HaveLen(1))
				Expect(sourceIntermediateSubinst.Status.InstallationReferences[0].Name).To(BeEquivalentTo("referenced"))
				intermediateReferencedSubinst := &lsv1alpha1.Installation{}
				utils.ExpectNoError(state.Client.Get(ctx, sourceIntermediateSubinst.Status.InstallationReferences[0].Reference.NamespacedName(), intermediateReferencedSubinst))

				By("fetch deployitems of referenced subinstallations")
				deployItems, err := lsutils.GetDeployItemsOfInstallation(ctx, state.Client, sourceReferencedSubinst)
				utils.ExpectNoError(err)
				Expect(deployItems).To(HaveLen(1))
				sourceReferencedDI := deployItems[0]

				deployItems, err = lsutils.GetDeployItemsOfInstallation(ctx, state.Client, intermediateReferencedSubinst)
				utils.ExpectNoError(err)
				Expect(deployItems).To(HaveLen(1))
				intermediateReferencedDI := deployItems[0]

				By("verify status")
				expectedVersion := "overwritten"
				providerStatus := map[string]interface{}{}
				utils.ExpectNoError(json.Unmarshal(sourceReferencedDI.Status.ProviderStatus.Raw, &providerStatus))
				Expect(providerStatus).To(HaveKeyWithValue("version", BeEquivalentTo(expectedVersion)), "componentVersionOverwrites did not overwrite the version of the directly referenced component")

				providerStatus = map[string]interface{}{}
				utils.ExpectNoError(json.Unmarshal(intermediateReferencedDI.Status.ProviderStatus.Raw, &providerStatus))
				Expect(providerStatus).To(HaveKeyWithValue("version", BeEquivalentTo(expectedVersion)), "componentVersionOverwrites did not overwrite the version of the indirectly referenced component")
			})

			It("should apply component version overwrites to references of already replaced components", func() {
				// define component descriptors
				/*
					Idea:
					'source' component references 'intermediate' in version v0.1.0.
					'intermediate' references 'referencedSource'.
					The corresponding installations just contain a nested installation for each referenced cd.

					ComponentVersionOverwrites are created for intermediate (v0.1.0 -> v0.2.0) and for referenced (v0.1.0 -> v1.2.3).
					The overwritten version of intermediate (v0.2.0) references the to-be-overwritten version of referenced (v0.1.0).
				*/
				var (
					testdataDir = filepath.Join(f.RootPath, "test", "integration", "testdata", "componentoverwrites")

					sourceComponentDir = filepath.Join(testdataDir, "source_component")
					sourceBlueprintDir = filepath.Join(sourceComponentDir, "blueprint")
					sourceInstFile     = filepath.Join(sourceComponentDir, "installation.yaml")

					intermediateName         = "intermediate" // has to match reference in subinstallation in the blueprint
					intermediateComponentDir = filepath.Join(testdataDir, "intermediate_component")
					intermediateBlueprintDir = filepath.Join(intermediateComponentDir, "blueprint")

					referencedName                    = "referenced" // has to match reference in subinstallation in the blueprint
					referencedComponentDir            = filepath.Join(testdataDir, "referenced_component")
					referencedSourceComponentDir      = filepath.Join(referencedComponentDir, "source")
					referencedSourceBlueprintDir      = filepath.Join(referencedSourceComponentDir, "blueprint")
					referencedOverwrittenComponentDir = filepath.Join(referencedComponentDir, "overwritten")
					referencedOverwrittenBlueprintDir = filepath.Join(referencedOverwrittenComponentDir, "blueprint")
				)

				// create and upload CDs
				By("build and upload component descriptors with blueprints")
				referencedSourceDescription := cdDescription{
					name:         "example.com/overwrites/tobeoverwritten",
					version:      "v0.1.0",
					blueprintDir: referencedSourceBlueprintDir,
				}
				referencedDummyDescription := cdDescription{
					name:         "example.com/overwrites/dummy",
					version:      "v0.0.1",
					blueprintDir: referencedSourceBlueprintDir,
				}
				referencedOverwrittenDescription := cdDescription{
					name:         "example.com/overwrites/referenced",
					version:      "v1.2.3",
					blueprintDir: referencedOverwrittenBlueprintDir,
				}
				intermediateSourceDescription := cdDescription{
					name:         "example.com/overwrites/intermediate",
					version:      "v0.1.0",
					blueprintDir: intermediateBlueprintDir,
					cdRefs: []cdv2.ComponentReference{
						{
							Name:          referencedName,
							ComponentName: referencedDummyDescription.name,
							Version:       referencedDummyDescription.version,
						},
					},
				}
				intermediateOverwrittenDescription := cdDescription{
					name:         intermediateSourceDescription.name,
					version:      "v0.2.0",
					blueprintDir: intermediateBlueprintDir,
					cdRefs: []cdv2.ComponentReference{
						{
							Name:          referencedName,
							ComponentName: referencedSourceDescription.name,
							Version:       referencedSourceDescription.version,
						},
					},
				}
				sourceDescription := cdDescription{
					name:         "example.com/overwrites/source",
					version:      "v0.1.0",
					blueprintDir: sourceBlueprintDir,
					cdRefs: []cdv2.ComponentReference{
						{
							Name:          intermediateName,
							ComponentName: intermediateSourceDescription.name,
							Version:       intermediateSourceDescription.version,
						},
						{
							Name:          referencedName,
							ComponentName: referencedSourceDescription.name,
							Version:       referencedSourceDescription.version,
						},
					},
				}

				cds := buildAndUploadComponentDescriptorsWithBlueprints(ctx, f, sourceDescription, intermediateSourceDescription, intermediateOverwrittenDescription, referencedSourceDescription, referencedDummyDescription, referencedOverwrittenDescription)
				repoCtx := cds[0].GetEffectiveRepositoryContext()

				By("create componentVersionOverwrite")
				cvoName := "cvo"
				cvo := &lsv1alpha1.ComponentVersionOverwrites{
					Overwrites: lsv1alpha1.ComponentVersionOverwriteList{
						{
							Source: lsv1alpha1.ComponentVersionOverwriteReference{
								RepositoryContext: repoCtx,
								ComponentName:     referencedSourceDescription.name,
								Version:           referencedSourceDescription.version,
							},
							Substitution: lsv1alpha1.ComponentVersionOverwriteReference{
								ComponentName: referencedOverwrittenDescription.name,
								Version:       referencedOverwrittenDescription.version,
							},
						},
						{
							Source: lsv1alpha1.ComponentVersionOverwriteReference{
								RepositoryContext: repoCtx,
								ComponentName:     intermediateSourceDescription.name,
								Version:           intermediateSourceDescription.version,
							},
							Substitution: lsv1alpha1.ComponentVersionOverwriteReference{
								Version: intermediateOverwrittenDescription.version,
							},
						},
					},
				}
				cvo.SetName(cvoName)
				cvo.SetNamespace(state.Namespace)
				utils.ExpectNoError(state.Create(ctx, cvo))

				By("patch context to reference ComponentVersionOverwrites")
				lsCtx := &lsv1alpha1.Context{}
				utils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKey(lsv1alpha1.DefaultContextName, state.Namespace), lsCtx))
				lsCtxOld := lsCtx.DeepCopy()
				lsCtx.ComponentVersionOverwritesReference = cvoName
				utils.ExpectNoError(state.Client.Patch(ctx, lsCtx, client.MergeFrom(lsCtxOld)))

				By("create installation")
				inst := &lsv1alpha1.Installation{}
				Expect(utils.ReadResourceFromFile(inst, sourceInstFile)).To(Succeed())
				inst.SetNamespace(state.Namespace)
				lsv1alpha1helper.SetOperation(&inst.ObjectMeta, lsv1alpha1.ReconcileOperation)
				inst.Spec.ComponentDescriptor = &lsv1alpha1.ComponentDescriptorDefinition{
					Reference: &lsv1alpha1.ComponentDescriptorReference{
						RepositoryContext: repoCtx,
						ComponentName:     sourceDescription.name,
						Version:           sourceDescription.version,
					},
				}
				inst.Spec.Blueprint.Reference.ResourceName = "blueprint"
				utils.ExpectNoError(state.Create(ctx, inst))

				// wait for installation to finish
				utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, state.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

				utils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(inst), inst)) // refresh installation for updated status
				var sourceIntermediateSubinst *lsv1alpha1.Installation
				var sourceReferencedSubinst *lsv1alpha1.Installation
				Expect(inst.Status.InstallationReferences).To(HaveLen(2))
				for _, subInstRef := range inst.Status.InstallationReferences {
					switch subInstRef.Name {
					case "intermediate":
						sourceIntermediateSubinst = &lsv1alpha1.Installation{}
						utils.ExpectNoError(state.Client.Get(ctx, subInstRef.Reference.NamespacedName(), sourceIntermediateSubinst))
					case "referenced":
						sourceReferencedSubinst = &lsv1alpha1.Installation{}
						utils.ExpectNoError(state.Client.Get(ctx, subInstRef.Reference.NamespacedName(), sourceReferencedSubinst))
					default:
						Fail(fmt.Sprintf("unexpected subinstallation: %s", subInstRef.Name))
					}
				}
				Expect(sourceIntermediateSubinst).ToNot(BeNil())
				Expect(sourceReferencedSubinst).ToNot(BeNil())

				By("fetch subinstallations of intermediate installation")
				Expect(sourceIntermediateSubinst.Status.InstallationReferences).To(HaveLen(1))
				Expect(sourceIntermediateSubinst.Status.InstallationReferences[0].Name).To(BeEquivalentTo("referenced"))
				intermediateReferencedSubinst := &lsv1alpha1.Installation{}
				utils.ExpectNoError(state.Client.Get(ctx, sourceIntermediateSubinst.Status.InstallationReferences[0].Reference.NamespacedName(), intermediateReferencedSubinst))

				By("fetch deployitems of referenced subinstallations")
				deployItems, err := lsutils.GetDeployItemsOfInstallation(ctx, state.Client, intermediateReferencedSubinst)
				utils.ExpectNoError(err)
				Expect(deployItems).To(HaveLen(1))
				intermediateReferencedDI := deployItems[0]

				By("verify status")
				expectedVersion := "overwritten"
				providerStatus := map[string]interface{}{}
				utils.ExpectNoError(json.Unmarshal(intermediateReferencedDI.Status.ProviderStatus.Raw, &providerStatus))
				Expect(providerStatus).To(HaveKeyWithValue("version", BeEquivalentTo(expectedVersion)), "componentVersionOverwrites did not overwrite a reference contained in an already overwritten component")
			})

		})

	})
}

func buildAndUploadNginxComponentDescriptorWithArtifacts(ctx context.Context, f *framework.Framework, name, version string) *cdv2.ComponentDescriptor {
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
	utils.ExpectNoError(cdv2.InjectRepositoryContext(cd, &repoCtx))
	utils.ExpectNoError(fs.MkdirAll("blobs", os.ModePerm))

	// gzip and add helm chart
	helmInput := input.BlobInput{
		Type:             input.DirInputType,
		Path:             helmChartDir,
		CompressWithGzip: pointer.BoolPtr(true),
	}
	blob, err := helmInput.Read(ctx, osfs.New(), "")
	utils.ExpectNoError(err)
	file, err := fs.Create("blobs/chart")
	utils.ExpectNoError(err)
	_, err = io.Copy(file, blob.Reader)
	utils.ExpectNoError(err)
	utils.ExpectNoError(file.Close())
	utils.ExpectNoError(blob.Reader.Close())

	cd.Resources = append(cd.Resources, buildLocalFilesystemResource("ingress-nginx-chart", "helm", input.MediaTypeGZip, "chart"))

	blueprintInput := input.BlobInput{
		Type:             input.DirInputType,
		Path:             blueprintDir,
		MediaType:        mediatype.NewBuilder(mediatype.BlueprintArtifactsLayerMediaTypeV1).Compression(mediatype.GZipCompression).String(),
		CompressWithGzip: pointer.BoolPtr(true),
	}
	blob, err = blueprintInput.Read(ctx, osfs.New(), "")
	utils.ExpectNoError(err)
	defer blob.Reader.Close()
	file, err = fs.Create("blobs/bp")
	utils.ExpectNoError(err)
	_, err = io.Copy(file, blob.Reader)
	utils.ExpectNoError(err)
	utils.ExpectNoError(file.Close())
	utils.ExpectNoError(blob.Reader.Close())

	cd.Resources = append(cd.Resources, buildLocalFilesystemResource("my-blueprint", mediatype.BlueprintType, blueprintInput.MediaType, "bp"))

	utils.ExpectNoError(cdv2.DefaultComponent(cd))

	ca := ctf.NewComponentArchive(cd, fs)
	manifest, err := cdoci.NewManifestBuilder(f.OCICache, ca).Build(ctx)
	utils.ExpectNoError(err)

	ref, err := cdoci.OCIRef(repoCtx, cd.Name, cd.Version)
	utils.ExpectNoError(err)

	err = f.OCIClient.PushManifest(ctx, ref, manifest)
	utils.ExpectNoError(err)
	return cd
}

type cdDescription struct {
	name         string                    // name of the component
	version      string                    // version of the component
	blueprintDir string                    // path to the blueprint directory of the component
	cdRefs       []cdv2.ComponentReference // referenced to other components
}

func buildAndUploadComponentDescriptorsWithBlueprints(ctx context.Context, f *framework.Framework, components ...cdDescription) []*cdv2.ComponentDescriptor {
	repoCtx := cdv2.OCIRegistryRepository{
		ObjectType: cdv2.ObjectType{
			Type: cdv2.OCIRegistryType,
		},
		BaseURL:              f.RegistryBasePath,
		ComponentNameMapping: cdv2.OCIRegistryURLPathMapping,
	}

	cds := []*cdv2.ComponentDescriptor{}

	for _, cdd := range components {
		// create component descriptor
		cd := &cdv2.ComponentDescriptor{}
		cd.Name = cdd.name
		cd.Version = cdd.version
		cd.Provider = cdv2.InternalProvider
		utils.ExpectNoError(cdv2.InjectRepositoryContext(cd, &repoCtx))

		// add blueprint
		fs := memoryfs.New()
		utils.ExpectNoError(fs.MkdirAll("blobs", os.ModePerm))
		blueprintInput := input.BlobInput{
			Type:             input.DirInputType,
			Path:             cdd.blueprintDir,
			MediaType:        mediatype.NewBuilder(mediatype.BlueprintArtifactsLayerMediaTypeV1).Compression(mediatype.GZipCompression).String(),
			CompressWithGzip: pointer.BoolPtr(true),
		}
		blob, err := blueprintInput.Read(ctx, osfs.New(), "")
		utils.ExpectNoError(err)
		defer blob.Reader.Close()
		file, err := fs.Create("blobs/blueprint")
		utils.ExpectNoError(err)
		_, err = io.Copy(file, blob.Reader)
		utils.ExpectNoError(err)
		utils.ExpectNoError(file.Close())
		utils.ExpectNoError(blob.Reader.Close())
		cd.Resources = append(cd.Resources, buildLocalFilesystemResource("blueprint", mediatype.BlueprintType, blueprintInput.MediaType, "blueprint"))
		cd.ComponentReferences = cdd.cdRefs

		// upload component descriptor
		utils.ExpectNoError(cdv2.DefaultComponent(cd))

		ca := ctf.NewComponentArchive(cd, fs)
		manifest, err := cdoci.NewManifestBuilder(f.OCICache, ca).Build(ctx)
		utils.ExpectNoError(err)

		ref, err := cdoci.OCIRef(repoCtx, cd.Name, cd.Version)
		utils.ExpectNoError(err)

		err = f.OCIClient.PushManifest(ctx, ref, manifest)
		utils.ExpectNoError(err)

		cds = append(cds, cd)
	}

	return cds
}

func buildLocalFilesystemResource(name, ttype, mediaType, path string) cdv2.Resource {
	res := cdv2.Resource{}
	res.Name = name
	res.Type = ttype
	res.Relation = cdv2.LocalRelation

	localFsAccess := cdv2.NewLocalFilesystemBlobAccess(path, mediaType)
	uAcc, err := cdv2.NewUnstructured(localFsAccess)
	utils.ExpectNoError(err)
	res.Access = &uAcc
	return res
}

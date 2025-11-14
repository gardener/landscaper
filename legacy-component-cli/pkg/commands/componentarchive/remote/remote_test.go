// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package remote_test

import (
	"bytes"
	"context"
	"os"
	"path"

	. "github.com/onsi/gomega"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/landscaper/legacy-component-spec/bindings-go/oci"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	ginkgo "github.com/onsi/ginkgo/v2"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/utils"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient"
	"github.com/gardener/landscaper/legacy-component-cli/ociclient/cache"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/components"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient/options"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive/remote"
)

var _ = ginkgo.Describe("Remote", func() {

	var (
		testdataFs       vfs.FileSystem
		srcRepoCtxURL    string
		targetRepoCtxURL string
	)

	ginkgo.BeforeEach(func() {
		baseFs, err := projectionfs.New(osfs.New(), "../")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)

		r := utils.RandomString(5)
		srcRepoCtxURL = testenv.Addr + "/test-" + r
		targetRepoCtxURL = testenv.Addr + "/target-" + r
	})

	ginkgo.It("should push a component archive", func() {
		baseFs, err := projectionfs.New(osfs.New(), "../")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)
		ctx := context.Background()

		cf, err := testenv.GetConfigFileBytes()
		Expect(err).ToNot(HaveOccurred())
		Expect(vfs.WriteFile(testdataFs, "/auth.json", cf, os.ModePerm))

		pushOpts := &remote.PushOptions{
			OciOptions: options.Options{
				AllowPlainHttp:     false,
				RegistryConfigPath: "/auth.json",
			},
		}
		pushOpts.ComponentArchivePath = "./testdata/00-ca"
		pushOpts.BaseUrl = testenv.Addr + "/test"

		Expect(pushOpts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

		repos, err := client.ListRepositories(ctx, testenv.Addr+"/test")
		Expect(err).ToNot(HaveOccurred())

		expectedRef := testenv.Addr + "/test/component-descriptors/example.com/component"
		Expect(repos).To(ContainElement(Equal(expectedRef)))

		manifest, err := client.GetManifest(ctx, expectedRef+":v0.0.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(manifest.Layers).To(HaveLen(1))
		Expect(manifest.Layers[0].MediaType).To(Equal(cdoci.ComponentDescriptorTarMimeType),
			"Expect that the first layer contains the component descriptor")
	})

	ginkgo.It("should get component archive", func() {
		baseFs, err := projectionfs.New(osfs.New(), "../")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)
		ctx := context.Background()

		cf, err := testenv.GetConfigFileBytes()
		Expect(err).ToNot(HaveOccurred())
		Expect(vfs.WriteFile(testdataFs, "/auth.json", cf, os.ModePerm))

		pushOpts := &remote.PushOptions{
			OciOptions: options.Options{
				AllowPlainHttp:     false,
				RegistryConfigPath: "/auth.json",
			},
		}
		pushOpts.ComponentArchivePath = "./testdata/00-ca"
		pushOpts.BaseUrl = testenv.Addr + "/test"

		Expect(pushOpts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

		showOpts := &remote.ShowOptions{
			OciOptions: options.Options{
				AllowPlainHttp:     false,
				RegistryConfigPath: "/auth.json",
			},
		}
		showOpts.BaseUrl = testenv.Addr + "/test"
		showOpts.ComponentName = "example.com/component"
		showOpts.Version = "v0.0.0"

		Expect(showOpts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())
	})

	ginkgo.It("should fail getting component archive which does not exist", func() {
		baseFs, err := projectionfs.New(osfs.New(), "../")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)
		ctx := context.Background()

		cf, err := testenv.GetConfigFileBytes()
		Expect(err).ToNot(HaveOccurred())
		Expect(vfs.WriteFile(testdataFs, "/auth.json", cf, os.ModePerm))

		showOpts := &remote.ShowOptions{
			OciOptions: options.Options{
				AllowPlainHttp:     false,
				RegistryConfigPath: "/auth.json",
			},
		}
		showOpts.BaseUrl = testenv.Addr + "/test"
		showOpts.ComponentName = "example.com/component"
		showOpts.Version = "v6.6.6"

		Expect(showOpts.Run(ctx, logr.Discard(), testdataFs)).To(HaveOccurred())
	})

	ginkgo.It("should copy a component descriptor and its blobs from the source repository to the target repository.", func() {
		baseFs, err := projectionfs.New(osfs.New(), "../")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)
		ctx := context.Background()

		cf, err := testenv.GetConfigFileBytes()
		Expect(err).ToNot(HaveOccurred())
		Expect(vfs.WriteFile(testdataFs, "/auth.json", cf, os.ModePerm))

		cd := &cdv2.ComponentDescriptor{}
		cd.Name = "example.com/component"
		cd.Version = "v0.0.0"
		cd.Provider = "internal"
		Expect(cdv2.InjectRepositoryContext(cd, cdv2.NewOCIRegistryRepository(srcRepoCtxURL, "")))

		blobContent := "blob test\n"

		pushOpts := &remote.PushOptions{
			OciOptions: options.Options{
				AllowPlainHttp:     false,
				RegistryConfigPath: "/auth.json",
			},
		}
		pushOpts.ComponentArchivePath = "./testdata/01-ca-blob"
		pushOpts.BaseUrl = srcRepoCtxURL
		Expect(pushOpts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

		repos, err := client.ListRepositories(ctx, srcRepoCtxURL)
		Expect(err).ToNot(HaveOccurred())

		srcOCIRef, err := components.OCIRef(cdv2.NewOCIRegistryRepository(srcRepoCtxURL, ""), cd.Name, cd.Version)
		expectedRef := srcRepoCtxURL + "/component-descriptors/" + cd.Name
		Expect(err).ToNot(HaveOccurred())
		Expect(repos).To(ContainElement(Equal(expectedRef)))

		manifest, err := client.GetManifest(ctx, srcOCIRef)
		Expect(err).ToNot(HaveOccurred())
		Expect(manifest.Layers).To(HaveLen(2))
		Expect(manifest.Layers[0].MediaType).To(Equal(cdoci.ComponentDescriptorTarMimeType),
			"Expect that the first layer contains the component descriptor")

		compResolver := cdoci.NewResolver(client)
		sourceComp, err := compResolver.Resolve(ctx, cdv2.NewOCIRegistryRepository(srcRepoCtxURL, ""), cd.Name, cd.Version)
		Expect(err).ToNot(HaveOccurred())

		Expect(sourceComp.Name).To(Equal(cd.Name))
		Expect(sourceComp.Resources[0].Access.Type).To(Equal("localOciBlob"), "Expect that the localFilesystem blob has been correctly converted to a localOciBlob")

		var layerBlob bytes.Buffer
		Expect(client.Fetch(ctx, srcOCIRef, manifest.Layers[1], &layerBlob)).To(Succeed(), "Expect that the second layer contains the local blob")
		Expect(layerBlob.String()).To(Equal(blobContent))

		copyOpts := &remote.CopyOptions{
			OciOptions: options.Options{
				AllowPlainHttp:     false,
				RegistryConfigPath: "/auth.json",
			},
		}
		copyOpts.SourceRepository = srcRepoCtxURL
		copyOpts.ComponentName = cd.Name
		copyOpts.ComponentVersion = cd.Version
		copyOpts.TargetRepository = targetRepoCtxURL

		Expect(copyOpts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

		repos, err = client.ListRepositories(ctx, targetRepoCtxURL)
		Expect(err).ToNot(HaveOccurred())

		targetOCIRef, err := components.OCIRef(cdv2.NewOCIRegistryRepository(targetRepoCtxURL, ""), cd.Name, cd.Version)
		Expect(err).ToNot(HaveOccurred())
		Expect(repos).To(ContainElement(Equal(targetRepoCtxURL+"/component-descriptors/"+cd.Name)), "Expect that the repositories contains target repo")

		manifestTarget, err := client.GetManifest(ctx, targetOCIRef)
		Expect(err).ToNot(HaveOccurred())
		Expect(manifestTarget.Layers).To(HaveLen(2))
		Expect(manifestTarget.Layers[0].MediaType).To(Equal(cdoci.ComponentDescriptorTarMimeType),
			"Expect that the first layer contains the component descriptor")
		Expect(manifestTarget.Layers[1].MediaType).To(Equal("text/plain"),
			"Expect that the second layer contains the local blob")

		targetComp, err := compResolver.Resolve(ctx, cdv2.NewOCIRegistryRepository(targetRepoCtxURL, ""), cd.Name, cd.Version)
		Expect(err).ToNot(HaveOccurred())

		Expect(targetComp.Name).To(Equal(cd.Name))
		Expect(targetComp.Resources[len(targetComp.Resources)-1].Access.Type).To(Equal("localOciBlob"), "Expect that the localFilesystem blob has been correctly converted to a localOciBlob")

		var layerBlobTarget bytes.Buffer
		Expect(client.Fetch(ctx, targetOCIRef, manifestTarget.Layers[1], &layerBlobTarget)).To(Succeed())
		Expect(layerBlobTarget.String()).To(Equal(blobContent), "Expect that the target blob contains the same as source blob")
	})

	ginkgo.Context("Copy", func() {

		var (
			srcRepoCtxURL    string
			targetRepoCtxURL string
		)

		ginkgo.BeforeEach(func() {
			r := utils.RandomString(5)
			srcRepoCtxURL = testenv.Addr + "/test-" + r
			targetRepoCtxURL = testenv.Addr + "/target-" + r
		})

		ginkgo.It("should copy a component descriptor with a docker image and an oci artifact by value", func() {
			ctx := context.Background()
			ociCache, err := cache.NewCache(logr.Discard())
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			cd.Name = "example.com/my-test-component"
			cd.Version = "v0.0.1"
			cd.Provider = "internal"
			Expect(cdv2.InjectRepositoryContext(cd, cdv2.NewOCIRegistryRepository(srcRepoCtxURL, "")))

			remoteOCIImage := cdv2.Resource{}
			remoteOCIImage.Name = "component-cli-image"
			remoteOCIImage.Version = "v0.28.0"
			remoteOCIImage.Type = cdv2.OCIImageType
			remoteOCIImage.Relation = cdv2.ExternalRelation
			remoteOCIImageAcc, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess("eu.gcr.io/gardener-project/component/cli:v0.28.0"))
			Expect(err).ToNot(HaveOccurred())
			remoteOCIImage.Access = &remoteOCIImageAcc

			remoteOCIArtifact := cdv2.Resource{}
			remoteOCIArtifact.Name = "landscaper-chart"
			remoteOCIArtifact.Version = "v0.11.0"
			remoteOCIArtifact.Type = "helm.chart.io"
			remoteOCIArtifact.Relation = cdv2.ExternalRelation
			remoteOCIArtifactAcc, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess("eu.gcr.io/gardener-project/landscaper/charts/landscaper-controller:v0.11.0"))
			Expect(err).ToNot(HaveOccurred())
			remoteOCIArtifact.Access = &remoteOCIArtifactAcc
			cd.Resources = append(cd.Resources, remoteOCIImage, remoteOCIArtifact)

			manifest, err := cdoci.NewManifestBuilder(ociCache, ctf.NewComponentArchive(cd, memoryfs.New())).Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			ref, err := components.OCIRef(cd.GetEffectiveRepositoryContext(), cd.Name, cd.Version)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.PushManifest(ctx, ref, manifest, ociclient.WithStore(ociCache)))

			baseFs, err := projectionfs.New(osfs.New(), "../")
			Expect(err).ToNot(HaveOccurred())
			testdataFs = layerfs.New(memoryfs.New(), baseFs)

			cf, err := testenv.GetConfigFileBytes()
			Expect(err).ToNot(HaveOccurred())
			Expect(vfs.WriteFile(testdataFs, "/auth.json", cf, os.ModePerm))

			copyOpts := &remote.CopyOptions{
				OciOptions: options.Options{
					AllowPlainHttp:     false,
					RegistryConfigPath: "/auth.json",
				},
				ComponentName:            cd.Name,
				ComponentVersion:         cd.Version,
				SourceRepository:         srcRepoCtxURL,
				TargetRepository:         targetRepoCtxURL,
				CopyByValue:              true,
				TargetArtifactRepository: targetRepoCtxURL,
			}
			Expect(copyOpts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

			compResolver := cdoci.NewResolver(client)
			targetComp, err := compResolver.Resolve(ctx, cdv2.NewOCIRegistryRepository(targetRepoCtxURL, ""), cd.Name, cd.Version)
			Expect(err).ToNot(HaveOccurred())

			Expect(targetComp.Resources).To(HaveLen(2))

			acc := &cdv2.OCIRegistryAccess{}
			Expect(targetComp.Resources[0].Access.DecodeInto(acc)).To(Succeed())
			Expect(acc.ImageReference).To(ContainSubstring(targetRepoCtxURL))
			Expect(acc.ImageReference).To(ContainSubstring("gardener-project/component/cli:v0.28.0"))

			acc = &cdv2.OCIRegistryAccess{}
			Expect(targetComp.Resources[1].Access.DecodeInto(acc)).To(Succeed())
			Expect(acc.ImageReference).To(ContainSubstring(targetRepoCtxURL))
			Expect(acc.ImageReference).To(ContainSubstring("gardener-project/landscaper/charts/landscaper-controller:v0.11.0"))
		})

		ginkgo.It("should replace parts of the target ref of copied docker image resource", func() {
			ctx := context.Background()
			ociCache, err := cache.NewCache(logr.Discard())
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			cd.Name = "example.com/my-test-component"
			cd.Version = "v0.0.1"
			cd.Provider = "internal"
			Expect(cdv2.InjectRepositoryContext(cd, cdv2.NewOCIRegistryRepository(srcRepoCtxURL, "")))

			registryAccessImage := cdv2.Resource{}
			registryAccessImage.Name = "component-cli-image"
			registryAccessImage.Version = "v0.28.0"
			registryAccessImage.Type = cdv2.OCIImageType
			registryAccessImage.Relation = cdv2.ExternalRelation
			registryAcc, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess("eu.gcr.io/gardener-project/component/cli:v0.28.0"))
			Expect(err).ToNot(HaveOccurred())
			registryAccessImage.Access = &registryAcc
			cd.Resources = append(cd.Resources, registryAccessImage)

			relativeAccessImage := cdv2.Resource{}
			relativeAccessImage.Name = "component-cli-image-relative-acc"
			relativeAccessImage.Version = "v0.28.0"
			relativeAccessImage.Type = cdv2.OCIImageType
			relativeAccessImage.Relation = cdv2.ExternalRelation
			relativeOCIRefAcc, err := cdv2.NewUnstructured(cdv2.NewRelativeOciAccess("component/cli:v0.28.0"))
			Expect(err).ToNot(HaveOccurred())
			relativeAccessImage.Access = &relativeOCIRefAcc
			cd.Resources = append(cd.Resources, relativeAccessImage)

			manifest, err := cdoci.NewManifestBuilder(ociCache, ctf.NewComponentArchive(cd, memoryfs.New())).Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			ref, err := components.OCIRef(cd.GetEffectiveRepositoryContext(), cd.Name, cd.Version)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.PushManifest(ctx, ref, manifest, ociclient.WithStore(ociCache)))

			baseFs, err := projectionfs.New(osfs.New(), "../")
			Expect(err).ToNot(HaveOccurred())
			testdataFs = layerfs.New(memoryfs.New(), baseFs)

			cf, err := testenv.GetConfigFileBytes()
			Expect(err).ToNot(HaveOccurred())
			Expect(vfs.WriteFile(testdataFs, "/auth.json", cf, os.ModePerm))

			copyOpts := &remote.CopyOptions{
				OciOptions: options.Options{
					AllowPlainHttp:     false,
					RegistryConfigPath: "/auth.json",
				},
				ComponentName:            cd.Name,
				ComponentVersion:         cd.Version,
				SourceRepository:         srcRepoCtxURL,
				TargetRepository:         targetRepoCtxURL,
				CopyByValue:              true,
				SourceArtifactRepository: "eu.gcr.io/gardener-project",
				TargetArtifactRepository: targetRepoCtxURL,
				ReplaceOCIRefs: []string{
					"gardener-project:my-project",
					"component/cli:component-cli",
				},
			}
			Expect(copyOpts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

			compResolver := cdoci.NewResolver(client)
			targetComp, err := compResolver.Resolve(ctx, cdv2.NewOCIRegistryRepository(targetRepoCtxURL, ""), cd.Name, cd.Version)
			Expect(err).ToNot(HaveOccurred())

			Expect(targetComp.Resources).To(HaveLen(2))

			res1Acc := &cdv2.OCIRegistryAccess{}
			Expect(targetComp.Resources[0].Access.DecodeInto(res1Acc)).To(Succeed())
			Expect(res1Acc.ImageReference).To(ContainSubstring(targetRepoCtxURL))
			Expect(res1Acc.ImageReference).To(ContainSubstring("my-project/component-cli:v0.28.0"))

			res2Acc := &cdv2.OCIRegistryAccess{}
			Expect(targetComp.Resources[1].Access.DecodeInto(res2Acc)).To(Succeed())
			Expect(res2Acc.ImageReference).To(ContainSubstring(targetRepoCtxURL))
			Expect(res2Acc.ImageReference).To(ContainSubstring("my-project/component-cli:v0.28.0"))
		})

		ginkgo.It("should copy a component descriptor with a relative oci ref and convert it to a absolute path", func() {
			ctx := context.Background()
			ociCache, err := cache.NewCache(logr.Discard())
			Expect(err).ToNot(HaveOccurred())

			// copy external image to registry
			ociImageTargetRelRef := "component/cli:v0.28.0"
			ociImageSrcRef := path.Join(srcRepoCtxURL, ociImageTargetRelRef)
			Expect(ociclient.Copy(ctx, client, "eu.gcr.io/gardener-project/component/cli:v0.28.0", ociImageSrcRef)).To(Succeed())

			cd := &cdv2.ComponentDescriptor{}
			cd.Name = "example.com/my-test-component"
			cd.Version = "v0.0.1"
			cd.Provider = "internal"

			Expect(cdv2.InjectRepositoryContext(cd, cdv2.NewOCIRegistryRepository(srcRepoCtxURL, "")))

			remoteOCIImage := cdv2.Resource{}
			remoteOCIImage.Name = "component-cli-image"
			remoteOCIImage.Version = "v0.28.0"
			remoteOCIImage.Type = cdv2.OCIImageType
			remoteOCIImage.Relation = cdv2.ExternalRelation
			remoteOCIImageAcc, err := cdv2.NewUnstructured(cdv2.NewRelativeOciAccess(ociImageTargetRelRef))
			Expect(err).ToNot(HaveOccurred())
			remoteOCIImage.Access = &remoteOCIImageAcc
			cd.Resources = append(cd.Resources, remoteOCIImage)

			manifest, err := cdoci.NewManifestBuilder(ociCache, ctf.NewComponentArchive(cd, memoryfs.New())).Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			ref, err := components.OCIRef(cd.GetEffectiveRepositoryContext(), cd.Name, cd.Version)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.PushManifest(ctx, ref, manifest, ociclient.WithStore(ociCache)))

			baseFs, err := projectionfs.New(osfs.New(), "../")
			Expect(err).ToNot(HaveOccurred())
			testdataFs = layerfs.New(memoryfs.New(), baseFs)

			cf, err := testenv.GetConfigFileBytes()
			Expect(err).ToNot(HaveOccurred())
			Expect(vfs.WriteFile(testdataFs, "/auth.json", cf, os.ModePerm))

			copyOpts := &remote.CopyOptions{
				OciOptions: options.Options{
					AllowPlainHttp:     false,
					SkipTLSVerify:      true,
					RegistryConfigPath: "/auth.json",
				},
				ComponentName:            cd.Name,
				ComponentVersion:         cd.Version,
				SourceRepository:         srcRepoCtxURL,
				TargetRepository:         targetRepoCtxURL,
				CopyByValue:              true,
				TargetArtifactRepository: targetRepoCtxURL,
				SourceArtifactRepository: srcRepoCtxURL,
			}
			Expect(copyOpts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

			compResolver := cdoci.NewResolver(client)
			targetComp, err := compResolver.Resolve(ctx, cdv2.NewOCIRegistryRepository(targetRepoCtxURL, ""), cd.Name, cd.Version)
			Expect(err).ToNot(HaveOccurred())

			Expect(targetComp.Resources).To(HaveLen(1))

			acc := &cdv2.OCIRegistryAccess{}
			Expect(targetComp.Resources[0].Access.DecodeInto(acc)).To(Succeed())
			Expect(acc.ImageReference).To(ContainSubstring(targetRepoCtxURL))
			Expect(acc.ImageReference).To(ContainSubstring(ociImageTargetRelRef))
		})

		ginkgo.It("should copy a component descriptor with a absolute oci ref and convert it to a relative path", func() {
			ctx := context.Background()
			ociCache, err := cache.NewCache(logr.Discard())
			Expect(err).ToNot(HaveOccurred())

			// copy external image to registry
			ociImageTargetRelRef := "component/cli:v0.28.0"
			ociImageSrcRef := path.Join(srcRepoCtxURL, ociImageTargetRelRef)
			Expect(ociclient.Copy(ctx, client, "eu.gcr.io/gardener-project/component/cli:v0.28.0", ociImageSrcRef)).To(Succeed())

			cd := &cdv2.ComponentDescriptor{}
			cd.Name = "example.com/my-test-component"
			cd.Version = "v0.0.1"
			cd.Provider = "internal"

			Expect(cdv2.InjectRepositoryContext(cd, cdv2.NewOCIRegistryRepository(srcRepoCtxURL, "")))

			remoteOCIImage := cdv2.Resource{}
			remoteOCIImage.Name = "component-cli-image"
			remoteOCIImage.Version = "v0.28.0"
			remoteOCIImage.Type = cdv2.OCIImageType
			remoteOCIImage.Relation = cdv2.ExternalRelation
			remoteOCIImageAcc, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess(ociImageSrcRef))
			Expect(err).ToNot(HaveOccurred())
			remoteOCIImage.Access = &remoteOCIImageAcc
			cd.Resources = append(cd.Resources, remoteOCIImage)

			manifest, err := cdoci.NewManifestBuilder(ociCache, ctf.NewComponentArchive(cd, memoryfs.New())).Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			ref, err := components.OCIRef(cd.GetEffectiveRepositoryContext(), cd.Name, cd.Version)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.PushManifest(ctx, ref, manifest, ociclient.WithStore(ociCache)))

			baseFs, err := projectionfs.New(osfs.New(), "../")
			Expect(err).ToNot(HaveOccurred())
			testdataFs = layerfs.New(memoryfs.New(), baseFs)

			cf, err := testenv.GetConfigFileBytes()
			Expect(err).ToNot(HaveOccurred())
			Expect(vfs.WriteFile(testdataFs, "/auth.json", cf, os.ModePerm))

			copyOpts := &remote.CopyOptions{
				OciOptions: options.Options{
					AllowPlainHttp:     false,
					SkipTLSVerify:      true,
					RegistryConfigPath: "/auth.json",
				},
				ComponentName:                  cd.Name,
				ComponentVersion:               cd.Version,
				SourceRepository:               srcRepoCtxURL,
				TargetRepository:               targetRepoCtxURL,
				CopyByValue:                    true,
				TargetArtifactRepository:       targetRepoCtxURL,
				ConvertToRelativeOCIReferences: true,
			}
			Expect(copyOpts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

			compResolver := cdoci.NewResolver(client)
			targetComp, err := compResolver.Resolve(ctx, cdv2.NewOCIRegistryRepository(targetRepoCtxURL, ""), cd.Name, cd.Version)
			Expect(err).ToNot(HaveOccurred())

			Expect(targetComp.Resources).To(HaveLen(1))

			acc := &cdv2.RelativeOciAccess{}
			Expect(targetComp.Resources[0].Access.DecodeInto(acc)).To(Succeed())
			Expect(acc.Reference).To(HaveSuffix(ociImageTargetRelRef))
		})

	})
})

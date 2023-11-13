// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"context"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-spec/bindings-go/ctf"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

type Factory interface {
	// NewRegistryAccess provides an instance of a RegistryAccess, which is an interface for dealing with ocm
	// components.Technically, it is a facade either backed by the [component-cli] or by the [ocmlib].
	//
	// fs allows to pass a file system that is considered for resolving local components or artifacts as well as other
	// local resources such as dockerconfig files specified in the ociRegistryConfig. If nil is passed, the hosts
	// file system is used.
	//
	// secrets allows to pass in credentials of specific types (such as dockerconfigjson or
	// credentials.config.ocm.software although the latter only works with the ocmlib backed implementation and are
	// ignored otherwise) that will be considered when accessing registries.
	//
	// sharedCache is an oci cache. It is only used by the component-cli backed implementation and is ignored otherwise,
	// as the ocmlib backed implementations uses an ocmlib internal cache for oci artifacts.
	//
	// localRegistryConfig allows to pass in a root path. This root path may already point to a local ocm repository (in
	// which case it is sufficient to specify that the repository context is of type "local" in the component reference
	// when trying to get a component version) or it may point to a directory above (in which case the repository
	// context has to be further specified).
	//
	// ociRegistryConfig allows to provide configuration for the oci client used to access artifacts in oci registries.
	// The OCICacheConfiguration only influences the component-cli backed implementation and is ignored otherwise, since
	// the ocmlib backed implementation uses an ocmlib internal cache for oci artifacts.
	//
	// inlineCd allows to pass a component descriptor into the RegistryAccess, so the described component can later be
	// resolved through it. This is primarily used to pass in the inline component descriptors specified in
	// installations. Local artifacts described in inline component descriptors can be resolved based on the fs and the
	// localRegistryConfig. Referenced components in remote repositories can be resolved based on the repository context
	// of the inline component descriptor itself.
	//
	// additionalBlobResolvers allows to pass in additional blob resolvers. These are only used by the component-cli
	// backed implementation and are ignored otherwise.
	//
	// [component-cli]: https://github.com/gardener/component-cli
	// [ocmlib]: https://github.com/open-component-model/ocm
	NewRegistryAccess(ctx context.Context,
		fs vfs.FileSystem,
		secrets []corev1.Secret,
		sharedCache cache.Cache,
		localRegistryConfig *config.LocalRegistryConfiguration,
		ociRegistryConfig *config.OCIConfiguration,
		inlineCd *types.ComponentDescriptor,
		additionalBlobResolvers ...ctf.TypedBlobResolver) (RegistryAccess, error)

	// NewHelmRepoResource returns a helm chart resource that is stored in a helm chart repository.
	NewHelmRepoResource(ctx context.Context,
		helmChartRepo *helmv1alpha1.HelmChartRepo,
		lsClient client.Client,
		contextObj *lsv1alpha1.Context) (TypedResourceProvider, error)

	// NewHelmOCIResource returns a helm chart resource that is stored in an OCI registry.
	NewHelmOCIResource(ctx context.Context,
		fs vfs.FileSystem,
		ociImageRef string,
		registryPullSecrets []corev1.Secret,
		ociConfig *config.OCIConfiguration,
		sharedCache cache.Cache) (TypedResourceProvider, error)
}

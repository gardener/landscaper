// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/dockerconfig"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/genericocireg"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/ocireg"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
)

type RegistryAccess struct {
	octx ocm.Context
}

var _ model.RegistryAccess = &RegistryAccess{}

func NewOCMRegistry(ctx context.Context, secrets []corev1.Secret,
	localRegistryConfig *config.LocalRegistryConfiguration, ociRegistryConfig *config.OCIConfiguration,
	inlineCd *cdv2.ComponentDescriptor) (model.RegistryAccess, error) {

	octx := ocm.DefaultContext()

	ociConfigFiles := make([]string, 0)
	if ociRegistryConfig != nil {
		ociConfigFiles = ociRegistryConfig.ConfigFiles
	}

	var spec *dockerconfig.RepositorySpec
	for _, path := range ociConfigFiles {
		spec = dockerconfig.NewRepositorySpec(path, true)
		_, err := octx.CredentialsContext().RepositoryForSpec(spec)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot access %v", path)
		}
	}

	for _, secret := range secrets {
		if secret.Type != corev1.SecretTypeDockerConfigJson {
			continue
		}
		dockerConfigBytes, ok := secret.Data[corev1.DockerConfigJsonKey]
		if !ok {
			continue
		}
		spec := dockerconfig.NewRepositorySpecForConfig(dockerConfigBytes)
		_, err := octx.CredentialsContext().RepositoryForSpec(spec)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot create credentials from secret")
		}
	}

	return &RegistryAccess{
		octx: octx,
	}, nil
}

func (r *RegistryAccess) GetComponentVersion(ctx context.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (model.ComponentVersion, error) {
	// Muss noch ersetzt werden, macht langfristig keinen Sinn, auf Datentypen aus dem Legacy Code aufzubauen
	var cnudieRepoSpec cdv2.OCIRegistryRepository
	if err := cdRef.RepositoryContext.DecodeInto(&cnudieRepoSpec); err != nil {
		return nil, err
	}
	ocmRepoSpec := ocireg.NewRepositorySpec(cnudieRepoSpec.BaseURL,
		&genericocireg.ComponentRepositoryMeta{ComponentNameMapping: genericocireg.ComponentNameMapping(string(cnudieRepoSpec.ComponentNameMapping))})
	repo, err := r.octx.RepositoryForSpec(ocmRepoSpec)
	if err != nil {
		return nil, err
	}

	compvers, err := repo.LookupComponentVersion(cdRef.ComponentName, cdRef.Version)
	if err != nil {
		return nil, err
	}

	return newComponentVersion(compvers), nil
}

//func (r *RegistryAccess) GetStandaloneResource(ctx context.Context, ref string) (model.Resource, error) {
//	spec := ociartifact.New(ref)
//
//	fakeComponentVersionAccess := &FakeComponentVersionAccess{
//		context: r.octx,
//	}
//
//	return &StandaloneResource{
//		accessSpec: spec,
//		compvers:   fakeComponentVersionAccess,
//	}, nil
//
//}

// temporary
func (r *RegistryAccess) GetComponentResolver() ctf.ComponentResolver {
	panic("to be removed")
}

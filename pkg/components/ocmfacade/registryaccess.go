package ocmfacade

import (
	"bytes"
	"context"
	"fmt"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository/attrs/localrootfs"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository/internal"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localfsblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localociblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/generics"
	"github.com/open-component-model/ocm/pkg/runtime"
	"sync"
)

var localaccesstypes = generics.NewSet(localblob.Type, localblob.TypeV1, localfsblob.Type, localfsblob.TypeV1, localociblob.Type, localociblob.TypeV1)

const PREDEFINED_CTF_PATH = "predefined-ctf"

type RegistryAccess struct {
	lock                           sync.RWMutex
	octx                           ocm.Context
	session                        ocm.Session
	predefinedComponentDescriptors []*compdesc.ComponentDescriptor
	predefinedRepository           ocm.Repository
}

func (r *RegistryAccess) GetComponentVersion(ctx context.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (_ model.ComponentVersion, rerr error) {
	var ocmrepo ocm.Repository

	// Predefined checks whether the referenced Component Descriptor (cdRef) is among the Inline Component Descriptors
	// Inline Component Descriptors are directly specified in the Installation
	predefined, err := r.isPredefined(cdRef)
	if err != nil {
		return nil, err
	}

	if predefined {
		// initialize in-memory ctf with all component versions defined through the inline component descriptor
		ocmrepo, err = r.getRepositoryForPredefined()
		if err != nil {
			return nil, err
		}
	} else {
		spec, err := r.octx.RepositorySpecForConfig(cdRef.RepositoryContext.Raw, runtime.DefaultYAMLEncoding)
		if err != nil {
			return nil, err
		}
		ocmrepo, err = r.session.LookupRepository(r.octx, spec)
		if err != nil {
			return nil, err
		}
	}

	compvers, err := r.session.LookupComponentVersion(ocmrepo, cdRef.ComponentName, cdRef.Version)
	if err != nil {
		return nil, err
	}
	return &ComponentVersion{
		componentVersionAccess: compvers,
	}, err
}

func (r *RegistryAccess) Close() error {
	err := r.predefinedRepository.Close()
	if err != nil {
		return err
	}
	err = r.session.Close()
	if err != nil {
		return err
	}
	return nil
}

// isPredefined returns (true, nil) if the referenced component descriptor is an Inline Component Descriptor
func (r *RegistryAccess) isPredefined(cdRef *lsv1alpha1.ComponentDescriptorReference) (bool, error) {
	if len(r.predefinedComponentDescriptors) == 0 {
		return false, nil
	}

	for _, cd := range r.predefinedComponentDescriptors {
		ocmRaw, err := cd.GetEffectiveRepositoryContext().GetRaw()
		if err != nil {
			return false, err
		}
		cnudieRaw, err := cdRef.RepositoryContext.GetRaw()
		if err != nil {
			return false, err
		}
		if bytes.Equal(ocmRaw, cnudieRaw) && cdRef.ComponentName == cd.Name && cdRef.Version == cd.Version {
			return true, nil
		}
	}
	return false, nil
}

func (r *RegistryAccess) getRepositoryForPredefined() (_ ocm.Repository, rerr error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.predefinedRepository != nil {
		return r.predefinedRepository, nil
	}

	memfs := memoryfs.New()

	for i, cd := range r.predefinedComponentDescriptors {
		data, err := compdesc.Encode(cd)
		if err != nil {
			return nil, err
		}
		file, err := memfs.Create(fmt.Sprintf("component-descriptor%d.yaml", i))
		if err != nil {
			return nil, err
		}
		if _, err := file.Write(data); err != nil {
			return nil, err
		}
		if err := file.Close(); err != nil {
			return nil, err
		}
	}
	return internal.NewRepository(r.octx, memfs, "/", localrootfs.Get(r.octx), "blobs")
}

func IsLocalAccessType(accessType string) bool {
	return localaccesstypes.Contains(accessType)
}

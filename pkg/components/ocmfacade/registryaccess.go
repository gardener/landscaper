package ocmfacade

import (
	"bytes"
	"context"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localfsblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localociblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/ctf"
	"github.com/open-component-model/ocm/pkg/finalizer"
	"github.com/open-component-model/ocm/pkg/generics"
	"github.com/open-component-model/ocm/pkg/runtime"
	"os"
	"sync"
)

var localaccesstypes = generics.NewSet(localblob.Type, localblob.TypeV1, localfsblob.Type, localfsblob.TypeV1, localociblob.Type, localociblob.TypeV1)

const PREDEFINED_CTF_PATH = "predefined-ctf"

type RegistryAccess struct {
	lock                           sync.RWMutex
	octx                           ocm.Context
	session                        ocm.Session
	predefinedComponentDescriptors []*compdesc.ComponentDescriptor
	memoryfs                       vfs.FileSystem
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
	err := vfs.Cleanup(r.memoryfs)
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

	var finalize finalizer.Finalizer

	if r.memoryfs != nil {
		exists, err := vfs.DirExists(r.memoryfs, PREDEFINED_CTF_PATH)
		if err != nil {
			return nil, err
		}
		if exists {
			return ctf.Open(r.octx, accessobj.ACC_WRITABLE, PREDEFINED_CTF_PATH, os.ModePerm, accessio.PathFileSystem(r.memoryfs))
		}
	}

	if r.memoryfs == nil {
		memfs, err := osfs.NewTempFileSystem()
		if err != nil {
			return nil, err
		}
		r.memoryfs = memfs
	}

	ocmrepo, err := ctf.Open(r.octx, accessobj.ACC_WRITABLE|accessobj.ACC_CREATE, PREDEFINED_CTF_PATH, os.ModePerm, accessio.PathFileSystem(r.memoryfs))
	if err != nil {
		return nil, err
	}

	finalize.Close(ocmrepo)

	for _, cd := range r.predefinedComponentDescriptors {
		loop := finalize.Nested()

		comp, err := ocmrepo.LookupComponent(cd.Name)
		if err != nil {
			return nil, err
		}
		loop.Close(comp)

		vers, err := comp.NewVersion(cd.Version)
		if err != nil {
			return nil, err
		}
		loop.Close(vers)

		for _, resource := range cd.Resources {
			if IsLocalAccessType(resource.Access.GetType()) {
				continue
			}
			err := vers.SetResource(&resource.ResourceMeta, resource.Access)
			if err != nil {
				return nil, err
			}
		}
		for _, source := range cd.Sources {
			if IsLocalAccessType(source.Access.GetType()) {
				continue
			}
			err := vers.SetSource(&source.SourceMeta, source.Access)
			if err != nil {
				return nil, err
			}
		}
		for _, reference := range cd.References {
			err := vers.SetReference(&reference)
			if err != nil {
				return nil, err
			}
		}

		err = comp.AddVersion(vers)
		if err != nil {
			return nil, err
		}

		err = loop.Finalize()
		if err != nil {
			return nil, err
		}
	}
	finalize.FinalizeWithErrorPropagation(&rerr)

	return ctf.Open(r.octx, accessobj.ACC_WRITABLE, PREDEFINED_CTF_PATH, os.ModePerm, accessio.PathFileSystem(r.memoryfs))
}

func IsLocalAccessType(accessType string) bool {
	return localaccesstypes.Contains(accessType)
}

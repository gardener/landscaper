// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package virtual

import (
	"sync"

	"github.com/open-component-model/ocm/pkg/common"
	ocicpi "github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

type IndexEntry[I interface{}] struct {
	cd   *compdesc.ComponentDescriptor
	info I
}

func (i *IndexEntry[I]) CD() *compdesc.ComponentDescriptor {
	if i == nil {
		return nil
	}
	return i.cd
}

func (i *IndexEntry[I]) Info() I {
	var zero I
	if i == nil {
		return zero
	}
	return i.info
}

type Index[I interface{}] struct {
	lock        sync.Mutex
	descriptors map[string]map[string]*IndexEntry[I]
}

func NewIndex[I interface{}]() *Index[I] {
	return &Index[I]{descriptors: map[string]map[string]*IndexEntry[I]{}}
}

func (i *Index[I]) NumComponents(prefix string) (int, error) {
	i.lock.Lock()
	defer i.lock.Unlock()

	list := ocicpi.FilterByNamespacePrefix(prefix, utils.StringMapKeys(i.descriptors))
	return len(list), nil
}

func (i *Index[I]) GetComponents(prefix string, closure bool) ([]string, error) {
	i.lock.Lock()
	defer i.lock.Unlock()

	return ocicpi.FilterChildren(closure, prefix, utils.StringMapKeys(i.descriptors)), nil
}

func (i *Index[I]) GetVersions(comp string) []string {
	i.lock.Lock()
	defer i.lock.Unlock()

	vers := i.descriptors[comp]
	if len(vers) == 0 {
		return []string{}
	}
	return utils.StringMapKeys(vers)
}

func (i *Index[I]) Get(comp, vers string) *IndexEntry[I] {
	i.lock.Lock()
	defer i.lock.Unlock()

	var e *IndexEntry[I]
	set := i.descriptors[comp]
	if len(vers) != 0 {
		e = set[vers]
	}
	return e
}

func (i *Index[I]) Add(cd *compdesc.ComponentDescriptor, info I) error {
	i.lock.Lock()
	defer i.lock.Unlock()

	set := i.descriptors[cd.Name]
	if set == nil {
		set = map[string]*IndexEntry[I]{}
		i.descriptors[cd.Name] = set
	}
	if set[cd.Version] != nil {
		return errors.ErrAlreadyExists(cpi.KIND_COMPONENTVERSION, common.VersionedElementKey(cd).String())
	}
	set[cd.Version] = &IndexEntry[I]{cd, info}
	return nil
}

func (i *Index[I]) Set(cd *compdesc.ComponentDescriptor, info I) {
	i.lock.Lock()
	defer i.lock.Unlock()

	set := i.descriptors[cd.Name]
	if set == nil {
		set = map[string]*IndexEntry[I]{}
		i.descriptors[cd.Name] = set
	}
	set[cd.Version] = &IndexEntry[I]{cd, info}
}

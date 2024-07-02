// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/landscaper/pkg/components/model/componentoverwrites"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

var sem chan bool

func init() {
	sem = make(chan bool, 30)
}

type componentIdentifier struct {
	Name    string
	Version string
}

func (i *componentIdentifier) String() string {
	return fmt.Sprintf("%s/%s", i.Name, i.Version)
}

func newComponentIdentifier(componentVersion ComponentVersion) *componentIdentifier {
	return &componentIdentifier{
		Name:    componentVersion.GetName(),
		Version: componentVersion.GetVersion(),
	}
}

type Protocol struct {
	entries []string
	mutex   sync.Mutex
}

func NewProtocol() *Protocol {
	return &Protocol{
		entries: make([]string, 0),
	}
}

func (p *Protocol) AddEntry(entry string) {
	p.mutex.Lock()
	p.entries = append(p.entries, entry)
	p.mutex.Unlock()
}

func (p *Protocol) GetEntries() string {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	var result string
	for _, entry := range p.entries {
		result += entry + "\n"
	}

	return result
}

type errorSet struct {
	errSet []error
	mutex  sync.Mutex
}

func (s *errorSet) add(err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.errSet = append(s.errSet, err)
}

func (s *errorSet) getFirstError() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if len(s.errSet) > 0 {
		return s.errSet[0]
	}

	return nil
}

func newErrorSet() *errorSet {
	return &errorSet{
		errSet: []error{},
	}
}

type componentVersionsMap struct {
	compMap map[componentIdentifier]ComponentVersion
	mutex   sync.Mutex
}

func newComponentVersionsMap() *componentVersionsMap {
	return &componentVersionsMap{
		compMap: make(map[componentIdentifier]ComponentVersion),
	}
}

func (c *componentVersionsMap) add(componentVersion ComponentVersion) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	id := newComponentIdentifier(componentVersion)

	if _, ok := c.compMap[*id]; ok {
		// we have already handled this component before, no need to do it again
		return false
	}
	c.compMap[*id] = componentVersion
	return true
}

func (c *componentVersionsMap) length() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return len(c.compMap)
}

// GetTransitiveComponentReferences returns a list of ComponentVersions that consists of the current one
// and all which are transitively referenced by it.
func GetTransitiveComponentReferences(ctx context.Context,
	componentVersion ComponentVersion,
	repositoryContext *types.UnstructuredTypedObject,
	overwriter componentoverwrites.Overwriter, protocol *Protocol) (*ComponentVersionList, error) {

	logger, ctx := logging.FromContextOrNew(ctx, nil)
	pm := utils.StartPerformanceMeasurement(&logger, "GetTransitiveComponentReferences")
	defer pm.StopDebug()

	cds := newComponentVersionsMap()
	errs := newErrorSet()

	var wg sync.WaitGroup
	getTransitiveComponentReferencesRecursively(ctx, componentVersion, repositoryContext, overwriter, cds, errs, &wg,
		protocol)

	wg.Wait()

	if err := errs.getFirstError(); err != nil {
		return nil, err
	}

	cdList := make([]ComponentVersion, cds.length())

	i := 0
	for _, cd := range cds.compMap {
		cdList[i] = cd
		i++
	}

	componentDescriptor := componentVersion.GetComponentDescriptor()

	return &ComponentVersionList{
		Metadata:   componentDescriptor.Metadata,
		Components: cdList,
	}, nil
}

// getTransitiveComponentReferencesRecursively is a helper function which fetches all referenced component descriptor,
// including the referencing one the fetched CDs are stored in the given 'cds' map to avoid duplicates
func getTransitiveComponentReferencesRecursively(ctx context.Context,
	cd ComponentVersion,
	repositoryContext *types.UnstructuredTypedObject,
	overwriter componentoverwrites.Overwriter,
	cds *componentVersionsMap,
	errs *errorSet,
	wg *sync.WaitGroup, protocol *Protocol) {

	logger, ctx := logging.FromContextOrNew(ctx, nil)
	pm := utils.StartPerformanceMeasurement(&logger, "getTransitiveComponentReferencesRecursively")
	defer pm.StopDebug()

	writeCalledForToProtocol(protocol, cd, "getTransitiveComponentReferencesRecursively")

	if cds.add(cd) {
		cdRepositoryContext := cd.GetRepositoryContext()
		if cdRepositoryContext == nil {
			errs.add(errors.New("component descriptor must at least contain one repository context with a base url"))
			return
		}

		cdComponentReferences := cd.GetComponentReferences()

		for i := range cdComponentReferences {
			compRef := &cdComponentReferences[i]

			if errs.getFirstError() != nil {
				return
			}

			writeCallingForToProtocol(protocol, cd, "getTransitiveComponentReferencesRecursively")
			wg.Add(1)
			go fetchComponentVersion(ctx, cd, compRef, repositoryContext, overwriter, cds, errs, wg, protocol)
		}
	}
}

func fetchComponentVersion(
	ctx context.Context,
	cd ComponentVersion,
	compRef *types.ComponentReference,
	repositoryContext *types.UnstructuredTypedObject,
	overwriter componentoverwrites.Overwriter,
	cds *componentVersionsMap,
	errs *errorSet,
	wg *sync.WaitGroup,
	protocol *Protocol,
) {
	defer wg.Done()

	sem <- true
	defer func() { <-sem }()

	writeCalledForToProtocol(protocol, cd, "fetchComponentVersion")

	referencedComponentVersion, err := cd.GetReferencedComponentVersion(ctx, compRef, repositoryContext, overwriter)
	if err != nil {
		errs.add(fmt.Errorf("unable to resolve component reference %s with component name %s and version %s: %w",
			compRef.Name, compRef.ComponentName, compRef.Version, err))
		return
	}

	writeCallingForToProtocol(protocol, cd, "fetchComponentVersion")
	getTransitiveComponentReferencesRecursively(ctx, referencedComponentVersion, repositoryContext, overwriter, cds,
		errs, wg, protocol)
}

func writeCalledForToProtocol(protocol *Protocol, cd ComponentVersion, method string) {
	if protocol != nil {
		entry := fmt.Sprintf("called for %s - method: %s", newComponentIdentifier(cd).String(), method)
		protocol.AddEntry(entry)
	}
}

func writeCallingForToProtocol(protocol *Protocol, cd ComponentVersion, method string) {
	if protocol != nil {
		entry := fmt.Sprintf("calling for %s - method: %s", newComponentIdentifier(cd).String(), method)
		protocol.AddEntry(entry)
	}
}

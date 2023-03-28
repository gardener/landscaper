package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"
)

// GetTransitiveComponentReferences returns a list of ComponentVersions that consists of the current one
// and all which are transitively referenced by it.
func GetTransitiveComponentReferences(ctx context.Context,
	componentVersion ComponentVersion,
	repositoryContext *types.UnstructuredTypedObject,
	overwriter componentoverwrites.Overwriter) (*ComponentVersionList, error) {

	cds := map[componentIdentifier]ComponentVersion{}
	if err := getTransitiveComponentReferencesRecursively(ctx, componentVersion, repositoryContext, overwriter, cds); err != nil {
		return nil, err
	}

	cdList := make([]ComponentVersion, len(cds))

	i := 0
	for _, cd := range cds {
		cdList[i] = cd
		i++
	}

	return &ComponentVersionList{
		Metadata:   componentVersion.GetComponentDescriptor().Metadata,
		Components: cdList,
	}, nil
}

type componentIdentifier struct {
	Name    string
	Version string
}

// getTransitiveComponentReferencesRecursively is a helper function which fetches all referenced component descriptor,
// including the referencing one the fetched CDs are stored in the given 'cds' map to avoid duplicates
func getTransitiveComponentReferencesRecursively(ctx context.Context,
	cd ComponentVersion,
	repositoryContext *types.UnstructuredTypedObject,
	overwriter componentoverwrites.Overwriter,
	cds map[componentIdentifier]ComponentVersion) error {

	cid := componentIdentifier{
		Name:    cd.GetName(),
		Version: cd.GetVersion(),
	}
	if _, ok := cds[cid]; ok {
		// we have already handled this component before, no need to do it again
		return nil
	}
	cds[cid] = cd

	if cd.GetRepositoryContext() == nil {
		return errors.New("component descriptor must at least contain one repository context with a base url")
	}

	for _, compRef := range cd.GetComponentReferences() {
		referencedComponentVersion, err := cd.GetReferencedComponentVersion(ctx, &compRef, repositoryContext, overwriter)
		if err != nil {
			return fmt.Errorf("unable to resolve component reference %s with component name %s and version %s: %w",
				compRef.Name, compRef.ComponentName, compRef.Version, err)
		}

		err = getTransitiveComponentReferencesRecursively(ctx, referencedComponentVersion, repositoryContext, overwriter, cds)
		if err != nil {
			return fmt.Errorf("unable to resolve component references for component descriptor %s with version %s: %w",
				compRef.Name, compRef.Version, err)
		}
	}

	return nil
}

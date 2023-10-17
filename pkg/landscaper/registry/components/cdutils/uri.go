// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/componentoverwrites"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

const Scheme = "cd"

const PathDelimiter = "/"

// path keywords
const (
	ComponentReferences = "componentReferences"
	Resources           = "resources"
)

// MaxURICharLength is teh maximum length that is supported for a uri.
const MaxURICharLength = 2083

type Path struct {
	Keyword string
	Value   string
}

// URI defines a component descriptor resource selector URI
type URI struct {
	Raw      string
	Path     []Path
	Fragment string
}

// ParseURI parses a component descriptor access uri of the format:
// cd://<keyword>/<value>/<keyword>/<value>/...
func ParseURI(cdURI string) (*URI, error) {
	if len(cdURI) > MaxURICharLength {
		return nil, fmt.Errorf("too long uri, got %d but expected max %d", len(cdURI), MaxURICharLength)
	}
	u, err := url.Parse(cdURI)
	if err != nil {
		return nil, err
	}
	uriPath := path.Join(u.Host, u.Path)

	if u.Scheme != Scheme {
		return nil, fmt.Errorf("scheme must be %s but given %s", Scheme, u.Scheme)
	}

	// parse key value pairs from path
	splitPath := strings.Split(uriPath, PathDelimiter)
	if len(splitPath) == 0 {
		return nil, errors.New("a path must be defined")
	}
	if len(splitPath)%2 != 0 {
		return nil, errors.New("even number of path arguments expected")
	}

	cdPath := make([]Path, len(splitPath)/2)
	for i := 0; i < len(splitPath)/2; i++ {
		cdPath[i] = Path{
			Keyword: splitPath[i*2],
			Value:   splitPath[i*2+1],
		}
	}

	return &URI{
		Raw:      cdURI,
		Path:     cdPath,
		Fragment: u.Fragment,
	}, nil
}

// Get resolves to a resource (model.Resource) or component (model.ComponentVersion) specified by the URI.
// It also returns the resource kind.
func (u *URI) Get(cd model.ComponentVersion, repositoryContext *types.UnstructuredTypedObject) (lsv1alpha1.ComponentDescriptorKind, interface{}, error) {
	var (
		ctx       = context.Background()
		component = cd
	)
	defer ctx.Done()
	for i, elem := range u.Path {
		isLast := len(u.Path) == i+1
		switch elem.Keyword {
		case ComponentReferences:
			ref := component.GetComponentReference(elem.Value)
			if ref == nil {
				return "", nil, fmt.Errorf("component reference %s cannot be found", elem.Value)
			}

			var err error
			component, err = component.GetReferencedComponentVersion(ctx, ref, repositoryContext, nil)
			if err != nil {
				return "", nil, fmt.Errorf("component %s cannot be found", elem.Value)
			}
			if isLast {
				return lsv1alpha1.ComponentResourceKind, component, nil
			}
		case Resources:
			res, err := component.GetResource(elem.Value, nil)
			if err != nil {
				return "", nil, fmt.Errorf("local resource %s cannot be found", elem.Value)
			}
			if !isLast {
				return "", nil, fmt.Errorf("the selector seems to contain more path segements after a external resource")
			}
			return lsv1alpha1.ResourceKind, res, nil
		default:
			return "", nil, fmt.Errorf("unknown keyword %s", elem.Keyword)
		}
	}
	return lsv1alpha1.ComponentResourceKind, component, nil
}

// GetComponent resolves to the component descriptor specified by the URI.
// If a resource is specified, the component descriptor of the resource is returned, in combination with the reference from which it was resolved.
// ComponentVersionOverwrites are taken into account, but unlike the returned component, the reference is not overwritten.
func (u *URI) GetComponent(cd model.ComponentVersion, repositoryContext *types.UnstructuredTypedObject,
	overwriter componentoverwrites.Overwriter) (model.ComponentVersion, *lsv1alpha1.ComponentDescriptorReference, error) {

	cdRepositoryContext := cd.GetRepositoryContext()

	var (
		ctx       = context.Background()
		component = cd
		cdRef     = &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: cdRepositoryContext,
			ComponentName:     cd.GetName(),
			Version:           cd.GetVersion(),
		}
	)

	defer ctx.Done()
	for i, elem := range u.Path {
		isLast := len(u.Path) == i+1
		switch elem.Keyword {
		case ComponentReferences:
			ref := component.GetComponentReference(elem.Value)
			if ref == nil {
				return nil, nil, fmt.Errorf("component reference %s cannot be found", elem.Value)
			}

			cdRef = &lsv1alpha1.ComponentDescriptorReference{
				RepositoryContext: repositoryContext,
				ComponentName:     ref.ComponentName,
				Version:           ref.Version,
			}

			var err error
			component, err = component.GetReferencedComponentVersion(ctx, ref, repositoryContext, overwriter)
			if err != nil {
				return nil, nil, fmt.Errorf("component %s cannot be found", elem.Value)
			}
			if isLast {
				return component, cdRef, nil
			}
		case Resources:
			if !isLast {
				return nil, nil, fmt.Errorf("the selector seems to contain more path segements after a resource")
			}
			return component, cdRef, nil
		default:
			return nil, nil, fmt.Errorf("unknown keyword %s", elem.Keyword)
		}
	}
	return component, cdRef, nil
}

// GetResource resolves to a resource specified by the URI.
// It also returns the resource kind.
func (u *URI) GetResource(cd model.ComponentVersion, repositoryContext *types.UnstructuredTypedObject) (model.ComponentVersion, model.Resource, error) {
	var (
		ctx       = context.Background()
		component = cd
	)
	defer ctx.Done()
	for i, elem := range u.Path {
		isLast := len(u.Path) == i+1
		switch elem.Keyword {
		case ComponentReferences:
			ref := component.GetComponentReference(elem.Value)
			if ref == nil {
				return nil, nil, fmt.Errorf("component reference %s cannot be found", elem.Value)
			}

			var err error
			component, err = component.GetReferencedComponentVersion(ctx, ref, repositoryContext, nil)
			if err != nil {
				return nil, nil, fmt.Errorf("component %s cannot be found", elem.Value)
			}
			if isLast {
				return nil, nil, fmt.Errorf("the selector seems to target a component desscriptor but a resource is requested")
			}
		case Resources:
			res, err := component.GetResource(elem.Value, nil)
			if err != nil {
				return nil, nil, fmt.Errorf("local resource %s cannot be found", elem.Value)
			}
			if !isLast {
				return nil, nil, fmt.Errorf("the selector seems to contain more path segements after a external resource")
			}
			return component, res, nil
		default:
			return nil, nil, fmt.Errorf("unknown keyword %s", elem.Keyword)
		}
	}
	return nil, nil, fmt.Errorf("unable to find resource %q", u.Raw)
}

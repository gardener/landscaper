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

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
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

// Get resolves to a resource or component descriptor specified by the URI.
// It also returns the resource kind.
func (u *URI) Get(cd *cdv2.ComponentDescriptor, compResolver ctf.ComponentResolver, repositoryContext *cdv2.UnstructuredTypedObject) (lsv1alpha1.ComponentDescriptorKind, interface{}, error) {
	var (
		ctx       = context.Background()
		component = cd
	)
	defer ctx.Done()
	for i, elem := range u.Path {
		isLast := len(u.Path) == i+1
		switch elem.Keyword {
		case ComponentReferences:
			refs, err := cd.GetComponentReferencesByName(elem.Value)
			if err != nil || len(refs) == 0 {
				return "", nil, fmt.Errorf("component %s cannot be found", elem.Value)
			}

			ref := refs[0]
			component, err = compResolver.Resolve(ctx, repositoryContext, ref.ComponentName, ref.Version)
			if err != nil {
				return "", nil, fmt.Errorf("component %s cannot be found", elem.Value)
			}
			if isLast {
				return lsv1alpha1.ComponentResourceKind, component, nil
			}
		case Resources:
			res, err := component.GetResourcesByName(elem.Value)
			if err != nil {
				return "", nil, fmt.Errorf("local resource %s cannot be found", elem.Value)
			}
			if !isLast {
				return "", nil, fmt.Errorf("the selector seems to contain more path segements after a external resource")
			}
			return lsv1alpha1.ResourceKind, res[0], nil
		default:
			return "", nil, fmt.Errorf("unknown keyword %s", elem.Keyword)
		}
	}
	return lsv1alpha1.ComponentResourceKind, component, nil
}

// GetComponent resolves to the component descriptor specified by the URI.
// If a resource is specified, the component descriptor of the resource is returned.
func (u *URI) GetComponent(cd *cdv2.ComponentDescriptor, compResolver ctf.ComponentResolver, repositoryContext *cdv2.UnstructuredTypedObject) (*cdv2.ComponentDescriptor, error) {
	var (
		ctx       = context.Background()
		component = cd
	)
	defer ctx.Done()
	for i, elem := range u.Path {
		isLast := len(u.Path) == i+1
		switch elem.Keyword {
		case ComponentReferences:
			refs, err := cd.GetComponentReferencesByName(elem.Value)
			if err != nil || len(refs) == 0 {
				return nil, fmt.Errorf("component %s cannot be found", elem.Value)
			}
			ref := refs[0]
			component, err = compResolver.Resolve(ctx, repositoryContext, ref.ComponentName, ref.Version)
			if err != nil {
				return nil, fmt.Errorf("component %s cannot be found", elem.Value)
			}
			if isLast {
				return component, nil
			}
		case Resources:
			if !isLast {
				return nil, fmt.Errorf("the selector seems to contain more path segements after a resource")
			}
			return component, nil
		default:
			return nil, fmt.Errorf("unknown keyword %s", elem.Keyword)
		}
	}
	return component, nil
}

// GetResource resolves to a resource specified by the URI.
// It also returns the resource kind.
func (u *URI) GetResource(cd *cdv2.ComponentDescriptor, compResolver ctf.ComponentResolver, repositoryContext *cdv2.UnstructuredTypedObject) (*cdv2.ComponentDescriptor, cdv2.Resource, error) {
	var (
		ctx       = context.Background()
		component = cd
	)
	defer ctx.Done()
	for i, elem := range u.Path {
		isLast := len(u.Path) == i+1
		switch elem.Keyword {
		case ComponentReferences:
			refs, err := cd.GetComponentReferencesByName(elem.Value)
			if err != nil || len(refs) == 0 {
				return nil, cdv2.Resource{}, fmt.Errorf("component %s cannot be found", elem.Value)
			}

			ref := refs[0]
			component, err = compResolver.Resolve(ctx, repositoryContext, ref.ComponentName, ref.Version)
			if err != nil {
				return nil, cdv2.Resource{}, fmt.Errorf("component %s cannot be found", elem.Value)
			}
			if isLast {
				return nil, cdv2.Resource{}, fmt.Errorf("the selector seems to target a component desscriptor but a resource is requested")
			}
		case Resources:
			res, err := component.GetResourcesByName(elem.Value)
			if err != nil {
				return nil, cdv2.Resource{}, fmt.Errorf("local resource %s cannot be found", elem.Value)
			}
			if !isLast {
				return nil, cdv2.Resource{}, fmt.Errorf("the selector seems to contain more path segements after a external resource")
			}
			return component, res[0], nil
		default:
			return nil, cdv2.Resource{}, fmt.Errorf("unknown keyword %s", elem.Keyword)
		}
	}
	return nil, cdv2.Resource{}, fmt.Errorf("unable to find resource %q", u.Raw)
}

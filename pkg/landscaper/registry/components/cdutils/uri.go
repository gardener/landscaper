// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cdutils

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

const Scheme = "cd"

const PathDelimiter = "/"

// path keywords
const (
	ComponentReferences = "componentReferences"
	LocalResources      = "localResources"
	ExternalResources   = "externalResources"
)

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
func (u *URI) Get(cd ResolvedComponentDescriptor) (lsv1alpha1.ResourceKind, interface{}, error) {
	component := cd
	for i, elem := range u.Path {
		isLast := len(u.Path) == i+1
		switch elem.Keyword {
		case ComponentReferences:
			var ok bool
			component, ok = component.ComponentReferences[elem.Value]
			if !ok {
				return "", nil, fmt.Errorf("component %s cannot be found", elem.Value)
			}
			if isLast {
				return lsv1alpha1.ComponentResourceKind, component, nil
			}
		case LocalResources:
			res, ok := component.LocalResources[elem.Value]
			if !ok {
				return "", nil, fmt.Errorf("local resource %s cannot be found", elem.Value)
			}
			if !isLast {
				return "", nil, fmt.Errorf("the selector seems to contain more path segements after a local resource")
			}
			return lsv1alpha1.LocalResourceKind, res, nil
		case ExternalResources:
			res, ok := component.LocalResources[elem.Value]
			if !ok {
				return "", nil, fmt.Errorf("local resource %s cannot be found", elem.Value)
			}
			if !isLast {
				return "", nil, fmt.Errorf("the selector seems to contain more path segements after a external resource")
			}
			return lsv1alpha1.ExternalResourceKind, res, nil
		default:
			return "", nil, fmt.Errorf("unknown keyword %s", elem.Keyword)
		}
	}
	return lsv1alpha1.ComponentResourceKind, component, nil
}

// Get resolves to the component descriptor specified by the URI.
// If a resource is specified, the component descriptor of the resource is returned.
func (u *URI) GetComponent(cd ResolvedComponentDescriptor) (ResolvedComponentDescriptor, error) {
	component := cd
	for i, elem := range u.Path {
		isLast := len(u.Path) == i+1
		switch elem.Keyword {
		case ComponentReferences:
			var ok bool
			component, ok = component.ComponentReferences[elem.Value]
			if !ok {
				return ResolvedComponentDescriptor{}, fmt.Errorf("component %s cannot be found", elem.Value)
			}
			if isLast {
				return component, nil
			}
		case LocalResources:
			if !isLast {
				return ResolvedComponentDescriptor{}, fmt.Errorf("the selector seems to contain more path segements after a local resource")
			}
			return component, nil
		case ExternalResources:
			if !isLast {
				return ResolvedComponentDescriptor{}, fmt.Errorf("the selector seems to contain more path segements after a external resource")
			}
			return component, nil
		default:
			return ResolvedComponentDescriptor{}, fmt.Errorf("unknown keyword %s", elem.Keyword)
		}
	}
	return component, nil
}

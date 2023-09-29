// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociuploadattr

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	ocicpi "github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	ATTR_KEY   = "github.com/mandelsoft/ocm/ociuploadrepo"
	ATTR_SHORT = "ociuploadrepo"
)

func init() {
	datacontext.RegisterAttributeType(ATTR_KEY, AttributeType{}, ATTR_SHORT)
}

type AttributeType struct{}

func (a AttributeType) Name() string {
	return ATTR_KEY
}

func (a AttributeType) Description() string {
	return `
*oci base repository ref*
Upload local OCI artifact blobs to a dedicated repository.
`
}

func (a AttributeType) Encode(v interface{}, marshaller runtime.Marshaler) ([]byte, error) {
	if _, ok := v.(*Attribute); !ok {
		return nil, fmt.Errorf("OCI Upload Attribute structure required")
	}
	return marshaller.Marshal(v)
}

func (a AttributeType) Decode(data []byte, unmarshaller runtime.Unmarshaler) (interface{}, error) {
	var value Attribute
	err := unmarshaller.Unmarshal(data, &value)
	if err == nil {
		if value.Repository != nil {
			if value.Repository.GetType() == "" {
				return nil, errors.ErrInvalidWrap(errors.Newf("missing repository type"), oci.KIND_OCI_REFERENCE, string(data))
			}
			return &value, nil
		}
		if value.Ref == "" {
			return nil, errors.ErrInvalidWrap(errors.Newf("missing repository or ref"), oci.KIND_OCI_REFERENCE, string(data))
		}
		data = []byte(value.Ref)
	}
	ref, err := oci.ParseRef(string(data))
	if err != nil {
		return nil, errors.ErrInvalidWrap(err, oci.KIND_OCI_REFERENCE, string(data))
	}
	if ref.Tag != nil || ref.Digest != nil {
		return nil, errors.ErrInvalidWrap(err, oci.KIND_OCI_REFERENCE, string(data))
	}
	return &Attribute{Ref: strings.Trim(string(data), "\"")}, nil
}

////////////////////////////////////////////////////////////////////////////////

type Attribute struct {
	Ref             string                        `json:"ociRef,omitempty"`
	Repository      *ocicpi.GenericRepositorySpec `json:"repository,omitempty"`
	NamespacePrefix string                        `json:"namespacePrefix,omitempty"`

	lock sync.Mutex
	ref  *oci.RefSpec
	spec []byte

	repo   oci.Repository
	prefix string
}

func AttributeDescription() map[string]string {
	return map[string]string{
		"ociRef":          "an OCI repository reference",
		"repository":      "an OCI repository specification for the target OCI registry",
		"namespacePrefix": "a namespace prefix used for the uploaded artifacts",
	}
}

func New(ref string) *Attribute {
	return &Attribute{Ref: ref}
}

func (a *Attribute) reset() {
	a.repo = nil
	a.prefix = ""
	a.ref = nil
	a.spec = nil
}

func (a *Attribute) Close() error {
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.repo != nil {
		defer a.reset()
		return a.repo.Close()
	}
	return nil
}

func (a *Attribute) GetInfo(ctx cpi.Context) (oci.Repository, *oci.UniformRepositorySpec, string, error) {
	if a.Ref != "" {
		return a.getByRef(ctx)
	}
	if a.Repository != nil {
		return a.getBySpec(ctx)
	}
	return nil, nil, "", errors.ErrInvalid("ociuploadspec")
}

func (a *Attribute) getBySpec(ctx cpi.Context) (oci.Repository, *oci.UniformRepositorySpec, string, error) {
	data, _ := a.Repository.MarshalJSON()

	spec, err := a.Repository.Evaluate(ctx.OCIContext())
	if err != nil {
		return nil, nil, "", errors.ErrInvalidWrap(err, oci.KIND_OCI_REFERENCE, string(data))
	}

	a.lock.Lock()
	defer a.lock.Unlock()

	if a.spec == nil || bytes.Equal(a.spec, data) {
		if a.repo != nil {
			a.repo.Close()
			a.reset()
		}

		a.repo, err = ctx.OCIContext().RepositoryForSpec(spec)
		if err != nil {
			return nil, nil, "", err
		}

		a.prefix = a.NamespacePrefix
		a.spec = data
		a.ref = &oci.RefSpec{UniformRepositorySpec: *spec.UniformRepositorySpec()}
		ctx.Finalizer().Close(a)
	}
	return a.repo, &a.ref.UniformRepositorySpec, a.prefix, nil
}

func (a *Attribute) getByRef(ctx cpi.Context) (oci.Repository, *oci.UniformRepositorySpec, string, error) {
	ref, err := oci.ParseRef(a.Ref)
	if err != nil {
		return nil, nil, "", errors.ErrInvalidWrap(err, oci.KIND_OCI_REFERENCE, a.Ref)
	}
	if ref.Tag != nil || ref.Digest != nil {
		return nil, nil, "", errors.ErrInvalidWrap(err, oci.KIND_OCI_REFERENCE, a.Ref)
	}

	a.lock.Lock()
	defer a.lock.Unlock()
	if a.ref == nil || ref != *a.ref {
		if a.repo != nil {
			a.repo.Close()
			a.reset()
		}

		spec, err := ctx.OCIContext().MapUniformRepositorySpec(&ref.UniformRepositorySpec)
		if err != nil {
			return nil, nil, "", err
		}
		a.repo, err = ctx.OCIContext().RepositoryForSpec(spec)
		if err != nil {
			return nil, nil, "", err
		}
		a.prefix = ref.Repository
		a.ref = &ref
		ctx.Finalizer().Close(a)
	}
	return a.repo, &a.ref.UniformRepositorySpec, a.prefix, nil
}

////////////////////////////////////////////////////////////////////////////////

func Get(ctx datacontext.Context) *Attribute {
	a := ctx.GetAttributes().GetAttribute(ATTR_KEY)
	if a == nil {
		return nil
	}
	return a.(*Attribute)
}

func Set(ctx datacontext.Context, attr *Attribute) error {
	return ctx.GetAttributes().SetAttribute(ATTR_KEY, attr)
}

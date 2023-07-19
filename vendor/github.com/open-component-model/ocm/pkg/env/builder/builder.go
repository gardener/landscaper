// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/env"
	"github.com/open-component-model/ocm/pkg/utils"
)

type element interface {
	SetBuilder(b *Builder)
	Type() string
	Close() error
	Set()

	Result() interface{}
}

type State struct{}

type base struct {
	*Builder
	result interface{}
}

func (e *base) SetBuilder(b *Builder) {
	e.Builder = b
}

func (e *base) Result() interface{} {
	return e.result
}

type Builder struct {
	*env.Environment
	stack []element

	ocm_repo ocm.Repository
	ocm_comp ocm.ComponentAccess
	ocm_vers ocm.ComponentVersionAccess
	ocm_rsc  *compdesc.ResourceMeta
	ocm_src  *compdesc.SourceMeta
	ocm_meta *compdesc.ElementMeta
	ocm_acc  *compdesc.AccessSpec

	blob *accessio.BlobAccess
	hint *string

	oci_repo          oci.Repository
	oci_nsacc         oci.NamespaceAccess
	oci_artacc        oci.ArtifactAccess
	oci_cleanuplayers bool
	oci_tags          *[]string
	oci_artfunc       func(oci.ArtifactAccess) error
	oci_annofunc      func(name, value string)
}

func NewBuilder(t *env.Environment) *Builder {
	if t == nil {
		t = env.NewEnvironment()
	}
	return &Builder{Environment: t}
}

var _ accessio.Option = (*Builder)(nil)

func (b *Builder) set() {
	b.ocm_repo = nil
	b.ocm_comp = nil
	b.ocm_vers = nil
	b.ocm_rsc = nil
	b.ocm_src = nil
	b.ocm_meta = nil
	b.ocm_acc = nil

	b.blob = nil
	b.hint = nil

	b.oci_repo = nil
	b.oci_nsacc = nil
	b.oci_artacc = nil
	b.oci_tags = nil
	b.oci_artfunc = nil
	b.oci_annofunc = nil

	if len(b.stack) > 0 {
		b.peek().Set()
	}
}

func (b *Builder) expect(p interface{}, msg string, tests ...func() bool) {
	if p == nil {
		Fail(msg+" required", 2)
	}
	for _, f := range tests {
		if !f() {
			Fail(msg+" required", 2)
		}
	}
}

func (b *Builder) failOn(err error, callerSkip ...int) {
	if err != nil {
		Fail(err.Error(), utils.Optional(callerSkip...)+2)
	}
}

func (b *Builder) peek() element {
	Expect(len(b.stack) > 0).To(BeTrue())
	return b.stack[len(b.stack)-1]
}

func (b *Builder) pop() element {
	Expect(len(b.stack) > 0).To(BeTrue())
	e := b.stack[len(b.stack)-1]
	b.stack = b.stack[:len(b.stack)-1]
	b.set()
	return e
}

func (b *Builder) push(e element) {
	b.stack = append(b.stack, e)
	b.set()
}

func (b *Builder) configure(e element, funcs []func(), skip ...int) interface{} {
	e.SetBuilder(b)
	b.push(e)
	for _, f := range funcs {
		if f != nil {
			f()
		}
	}
	err := b.pop().Close()
	if err != nil {
		Fail(err.Error(), utils.Optional(skip...)+2)
	}
	return e.Result()
}

////////////////////////////////////////////////////////////////////////////////

const T_BLOBACCESS = "blob access"

func (b *Builder) BlobStringData(mime string, data string) {
	b.expect(b.blob, T_BLOBACCESS)
	if b.ocm_acc != nil && *b.ocm_acc != nil {
		Fail("access already set", 1)
	}
	*(b.blob) = accessio.BlobAccessForData(mime, []byte(data))
}

func (b *Builder) BlobData(mime string, data []byte) {
	b.expect(b.blob, T_BLOBACCESS)
	if b.ocm_acc != nil && *b.ocm_acc != nil {
		Fail("access already set", 1)
	}
	*(b.blob) = accessio.BlobAccessForData(mime, data)
}

func (b *Builder) BlobFromFile(mime string, path string) {
	b.expect(b.blob, T_BLOBACCESS)
	if b.ocm_acc != nil && *b.ocm_acc != nil {
		Fail("access already set", 1)
	}
	*(b.blob) = accessio.BlobAccessForFile(mime, path, b.FileSystem())
}

func (b *Builder) Hint(hint string) {
	b.expect(b.hint, T_OCMACCESS)
	if b.ocm_acc != nil && *b.ocm_acc != nil {
		Fail("access already set", 1)
	}
	*(b.hint) = hint
}

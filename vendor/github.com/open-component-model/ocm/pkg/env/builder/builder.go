// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/modern-go/reflect2"
	"github.com/onsi/ginkgo/v2"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/env"
	"github.com/open-component-model/ocm/pkg/exception"
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

type static struct {
	def_modopts ocm.ModificationOptions
}

type state struct {
	*static
	ocm_repo    ocm.Repository
	ocm_comp    ocm.ComponentAccess
	ocm_vers    ocm.ComponentVersionAccess
	ocm_rsc     *compdesc.ResourceMeta
	ocm_src     *compdesc.SourceMeta
	ocm_meta    *compdesc.ElementMeta
	ocm_labels  *metav1.Labels
	ocm_acc     *compdesc.AccessSpec
	ocm_modopts *ocm.ModificationOptions

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

type Builder struct {
	*env.Environment
	stack []element
	state
}

// New creates a new composition environment
// including an own OCM context and a private
// filesystem, which can be used to compose
// OCM/OCI repositories and their content.
// It can be configured to work with dedicated
// settings, also.
func New(opts ...env.Option) *Builder {
	return &Builder{Environment: env.NewEnvironment(append([]env.Option{env.FileSystem(osfs.OsFs, "/"), env.FailHandler(env.ExceptionFailHandler)}, opts...)...), state: state{static: &static{}}}
}

// NewBuilder creates a new composition environment
// including an own OCM context and a private
// filesystem, which can be used to compose
// OCM/OCI repositories and their content.
// By default, a private environment is created based on
// a ginko fail handling intended to be used for test cases.
// But it can be configured to work as library with dedicated
// settings, also.
func NewBuilder(opts ...env.Option) *Builder {
	return &Builder{Environment: env.NewEnvironment(append([]env.Option{env.FailHandler(ginkgo.Fail)}, opts...)...), state: state{static: &static{}}}
}

var _ accessio.Option = (*Builder)(nil)

// Build executes the given functions and returns a potential configuration
// error, instead of using the builder's env.FailHandler.
// Additionally, a build can always throw an exception using
// the exception.Throw function.
func (b *Builder) Build(funcs ...func(*Builder)) (err error) {
	old := b.GetFailHandler()
	defer func() {
		b.SetFailHandler(old)
	}()
	b.SetFailHandler(env.ExceptionFailHandler)

	defer exception.PropagateException(&err)
	for _, f := range funcs {
		f(b)
	}
	return nil
}

func (b *Builder) SetFailhandler(h ...env.FailHandler) *Builder {
	b.Environment.SetFailHandler(h...)
	return b
}

// PropagateError can be used in defer to convert an composition error
// into an error return.
func (b *Builder) PropagateError(errp *error, matchers ...exception.Matcher) {
	if r := recover(); r != nil {
		*errp = exception.FilterException(r, matchers...)
	}
}

func (b *Builder) set() {
	b.state = state{static: b.state.static}

	if len(b.stack) > 0 {
		b.peek().Set()
	}
}

func (b *Builder) expect(p interface{}, msg string, tests ...func() bool) {
	if reflect2.IsNil(p) {
		b.fail(msg+" required", 1)
	}
	for _, f := range tests {
		if !f() {
			b.fail(msg+" required", 1)
		}
	}
}

func (b *Builder) fail(msg string, callerSkip ...int) {
	b.Fail(msg, utils.Optional(callerSkip...)+2)
}

func (b *Builder) failOn(err error, callerSkip ...int) {
	b.FailOnErr(err, "", utils.Optional(callerSkip...)+2)
}

func (b *Builder) peek() element {
	if len(b.stack) == 0 {
		b.fail("no open frame", 2)
	}
	return b.stack[len(b.stack)-1]
}

func (b *Builder) pop() element {
	if len(b.stack) == 0 {
		b.fail("no open frame", 2)
	}
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
	b.Configure(funcs...)
	err := b.pop().Close()
	if err != nil {
		b.fail(err.Error(), utils.Optional(skip...)+1)
	}
	return e.Result()
}

func (b *Builder) Configure(funcs ...func()) {
	for _, f := range funcs {
		if f != nil {
			f()
		}
	}
}

////////////////////////////////////////////////////////////////////////////////

const T_BLOBACCESS = "blob access"

func (b *Builder) BlobStringData(mime string, data string) {
	b.expect(b.blob, T_BLOBACCESS)
	if b.ocm_acc != nil && *b.ocm_acc != nil {
		b.fail("access already set")
	}
	*(b.blob) = accessio.BlobAccessForData(mime, []byte(data))
}

func (b *Builder) BlobData(mime string, data []byte) {
	b.expect(b.blob, T_BLOBACCESS)
	if b.ocm_acc != nil && *b.ocm_acc != nil {
		b.fail("access already set")
	}
	*(b.blob) = accessio.BlobAccessForData(mime, data)
}

func (b *Builder) BlobFromFile(mime string, path string) {
	b.expect(b.blob, T_BLOBACCESS)
	if b.ocm_acc != nil && *b.ocm_acc != nil {
		b.fail("access already set")
	}
	*(b.blob) = accessio.BlobAccessForFile(mime, path, b.FileSystem())
}

func (b *Builder) Hint(hint string) {
	b.expect(b.hint, T_OCMACCESS)
	if b.ocm_acc != nil && *b.ocm_acc != nil {
		b.fail("access already set")
	}
	*(b.hint) = hint
}

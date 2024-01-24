// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessobj

import (
	"fmt"
	"sync"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/modern-go/reflect2"
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/errors"
)

// These objects deal with descriptor based state descriptions
// of an access object

type AccessMode byte

const (
	ACC_WRITABLE = AccessMode(0)
	ACC_READONLY = AccessMode(1)
	ACC_CREATE   = AccessMode(2)
)

func (m AccessMode) IsReadonly() bool {
	return (m & ACC_READONLY) != 0
}

func (m AccessMode) IsCreate() bool {
	return (m & ACC_CREATE) != 0
}

var ErrReadOnly = accessio.ErrReadOnly

// StateHandler is responsible to handle the technical representation of state
// carrying object as byte array.
type StateHandler interface {
	Initial() interface{}
	Encode(d interface{}) ([]byte, error)
	Decode([]byte) (interface{}, error)
	Equivalent(a, b interface{}) bool
}

// StateAccess is responsible to handle the persistence
// of a state object.
type StateAccess interface {
	// Get returns the technical representation of a state object from its persistence
	// It MUST return an errors.IsErrNotFound compatible error
	// if the persistence not yet exists.
	Get() (blobaccess.BlobAccess, error)
	// Digest() digest.Digest
	Put(data []byte) error
}

// BlobStateAccess provides state handling for data given by a blob access.
type BlobStateAccess struct {
	lock sync.RWMutex
	blob blobaccess.BlobAccess
}

var _ StateAccess = (*BlobStateAccess)(nil)

func NewBlobStateAccess(blob blobaccess.BlobAccess) StateAccess {
	return &BlobStateAccess{
		blob: blob,
	}
}

func NewBlobStateAccessForData(mimeType string, data []byte) StateAccess {
	return &BlobStateAccess{
		blob: blobaccess.ForData(mimeType, data),
	}
}

func (b *BlobStateAccess) Get() (blobaccess.BlobAccess, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.blob, nil
}

func (b *BlobStateAccess) Put(data []byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.blob = blobaccess.ForData(b.blob.MimeType(), data)
	return nil
}

func (b *BlobStateAccess) Digest() digest.Digest {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.blob.Digest()
}

// State manages the modification and access of state
// with a technical representation as byte array
// It tries to keep the byte representation unchanged as long as
// possible.
type State interface {
	IsReadOnly() bool
	IsCreate() bool

	GetOriginalBlob() blobaccess.BlobAccess
	GetBlob() (blobaccess.BlobAccess, error)

	HasChanged() bool
	GetOriginalState() interface{}
	GetState() interface{}

	// Update updates the technical representation in its persistence
	Update() (bool, error)
}

type state struct {
	mode         AccessMode
	access       StateAccess
	handler      StateHandler
	originalBlob blobaccess.BlobAccess
	original     interface{}
	current      interface{}
}

var _ State = (*state)(nil)

// NewState creates a new State based on its persistence handling
// and the management of its technical representation as byte array.
func NewState(mode AccessMode, a StateAccess, p StateHandler) (State, error) {
	state, err := newState(mode, a, p)
	// avoid nil pinter problem: go is great
	if err != nil {
		return nil, err
	}
	return state, nil
}

func newState(mode AccessMode, a StateAccess, p StateHandler) (*state, error) {
	blob, err := a.Get()
	if err != nil {
		if (mode&ACC_CREATE) == 0 || !errors.IsErrNotFound(err) {
			return nil, err
		}
	}

	var current, original interface{}

	if blob != nil {
		data, err := blob.Get()
		if err != nil {
			return nil, fmt.Errorf("failed to get blob data: %w", err)
		}

		blob = blobaccess.ForData(blob.MimeType(), data) // cache original data
		current, err = p.Decode(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode blob data: %w", err)
		}

		// we don't need a copy operation, because we can just deserialize it twice.
		original, _ = p.Decode(data)
	} else {
		current = p.Initial()
	}

	return &state{
		mode:         mode,
		access:       a,
		handler:      p,
		originalBlob: blob,
		original:     original,
		current:      current,
	}, nil
}

// NewBlobStateForBlob provides state handling for an object persisted as a blob.
// It tries to keep the blob representation unchanged as long as possible
// consulting the state handler responsible for analysing the binary blob data
// and the object.
func NewBlobStateForBlob(mode AccessMode, blob blobaccess.BlobAccess, p StateHandler) (State, error) {
	if blob == nil {
		data, err := p.Encode(p.Initial())
		if err != nil {
			return nil, err
		}
		blob = blobaccess.ForData("", data)
	}
	return NewState(mode, NewBlobStateAccess(blob), p)
}

// NewBlobStateForObject returns a representation state handling for a given object.
func NewBlobStateForObject(mode AccessMode, obj interface{}, p StateHandler) (State, error) {
	if reflect2.IsNil(obj) {
		obj = p.Initial()
	}
	data, err := p.Encode(obj)
	if err != nil {
		return nil, err
	}
	return NewBlobStateForBlob(mode, blobaccess.ForData("", data), p)
}

func (s *state) IsReadOnly() bool {
	return s.mode.IsReadonly()
}

func (s *state) IsCreate() bool {
	return s.mode.IsCreate()
}

func (s *state) Refresh() error {
	n, err := newState(s.mode, s.access, s.handler)
	if err != nil {
		return fmt.Errorf("unable to create new state: %w", err)
	}

	*s = *n
	return nil
}

func (s *state) GetOriginalState() interface{} {
	if s.originalBlob == nil {
		return nil
	}
	// always provide a private copy to not corrupt the internal state
	var original interface{}
	data, err := s.originalBlob.Get()
	if err == nil {
		original, err = s.handler.Decode(data)
	}
	if err != nil {
		panic("use of invalid state: " + err.Error())
	}
	return original
}

func (s *state) GetState() interface{} {
	return s.current
}

func (s *state) GetOriginalBlob() blobaccess.BlobAccess {
	b, _ := s.originalBlob.Dup()
	return b
}

func (s *state) HasChanged() bool {
	if s.original == nil {
		return true
	}
	return !s.handler.Equivalent(s.original, s.current)
}

func (s *state) GetBlob() (blobaccess.BlobAccess, error) {
	if !s.HasChanged() {
		return s.originalBlob.Dup()
	}
	data, err := s.handler.Encode(s.current)
	if err != nil {
		return nil, err
	}
	if s.originalBlob != nil {
		return blobaccess.ForData(s.originalBlob.MimeType(), data), nil
	}
	return blobaccess.ForData("", data), nil
}

func (s *state) Update() (bool, error) {
	if !s.HasChanged() {
		return false, nil
	}

	if s.IsReadOnly() {
		return true, ErrReadOnly
	}
	data, err := s.handler.Encode(s.current)
	if err != nil {
		return false, err
	}
	original, err := s.handler.Decode(data)
	if err != nil {
		return false, err
	}
	err = s.access.Put(data)
	if err != nil {
		return false, err
	}
	mimeType := ""
	if s.originalBlob != nil {
		mimeType = s.originalBlob.MimeType()
	}
	s.originalBlob = blobaccess.ForData(mimeType, data)
	s.original = original
	return true, nil
}

////////////////////////////////////////////////////////////////////////////////

type fileBasedAccess struct {
	filesystem vfs.FileSystem
	path       string
	mimeType   string
	mode       vfs.FileMode
}

func (f *fileBasedAccess) Get() (blobaccess.BlobAccess, error) {
	ok, err := vfs.FileExists(f.filesystem, f.path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.ErrNotFoundWrap(vfs.ErrNotExist, "file", f.path, f.filesystem.Name())
	}
	return blobaccess.ForFile(f.mimeType, f.path, f.filesystem), nil
}

func (f *fileBasedAccess) Put(data []byte) error {
	if err := vfs.WriteFile(f.filesystem, f.path, data, f.mode); err != nil {
		return fmt.Errorf("unable to write file %q: %w", f.path, err)
	}
	return nil
}

func (f *fileBasedAccess) Digest() digest.Digest {
	data, err := f.filesystem.Open(f.path)
	if err == nil {
		defer data.Close()
		d, err := digest.FromReader(data)
		if err == nil {
			return d
		}
	}
	return ""
}

////////////////////////////////////////////////////////////////////////////////

// NewFileBasedState create a new State object based on a file based persistence
// of the state carrying object.
func NewFileBasedState(acc AccessMode, fs vfs.FileSystem, path string, mimeType string, h StateHandler, mode vfs.FileMode) (State, error) {
	return NewState(acc, &fileBasedAccess{
		filesystem: fs,
		path:       path,
		mode:       mode,
		mimeType:   mimeType,
	}, h)
}

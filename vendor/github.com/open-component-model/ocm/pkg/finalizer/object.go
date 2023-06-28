// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package finalizer

import (
	"fmt"
	"runtime"
	"sync"
)

// NumberRange can be used as source for successive id numbers to tag
// elements, since debuggers not always sow object addresses.
type NumberRange struct {
	id   uint64
	lock sync.Mutex
}

func (n *NumberRange) NextId() uint64 {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.id++
	return n.id
}

var (
	lock         sync.Mutex
	objectranges = map[string]*NumberRange{}
)

type ObjectIdentity string

func (i ObjectIdentity) String() string {
	return string(i)
}

func NewObjectIdentity(kind string) ObjectIdentity {
	lock.Lock()
	defer lock.Unlock()
	nr := objectranges[kind]
	if nr == nil {
		nr = &NumberRange{}
		objectranges[kind] = nr
	}
	return ObjectIdentity(fmt.Sprintf("%s/%d", kind, nr.NextId()))
}

type RuntimeFinalizationRecoder struct {
	lock sync.Mutex
	ids  []ObjectIdentity
}

func (r *RuntimeFinalizationRecoder) Get() []ObjectIdentity {
	r.lock.Lock()
	defer r.lock.Unlock()

	return append(r.ids[:0:0], r.ids...)
}

func (r *RuntimeFinalizationRecoder) Record(id ObjectIdentity) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.ids = append(r.ids, id)
}

func (r *RuntimeFinalizationRecoder) IsFinalized(objs ...ObjectIdentity) bool {
	for _, o := range objs {
		found := false
		for _, f := range r.ids {
			if f == o {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return len(r.ids) != 0
}

type RuntimeFinalizer struct {
	id       ObjectIdentity
	recorder *RuntimeFinalizationRecoder
}

func fi(o *RuntimeFinalizer) {
	o.finalize()
}

func NewRuntimeFinalizer(id ObjectIdentity, r *RuntimeFinalizationRecoder) *RuntimeFinalizer {
	f := &RuntimeFinalizer{
		id:       id,
		recorder: r,
	}

	runtime.SetFinalizer(f, fi)
	return f
}

func (f *RuntimeFinalizer) finalize() {
	if f.recorder != nil {
		f.recorder.Record(f.id)
		f.recorder = nil
	}
}

type RecorderProvider interface {
	GetRecorder() *RuntimeFinalizationRecoder
}

func GetRuntimeFinalizationRecorder(o any) *RuntimeFinalizationRecoder {
	if r, ok := o.(RecorderProvider); ok {
		return r.GetRecorder()
	}
	return nil
}

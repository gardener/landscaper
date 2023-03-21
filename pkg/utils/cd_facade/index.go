package cd_facade

import (
	"time"

	"github.com/gardener/component-cli/ociclient/cache"
)

type Index struct {
	index *cache.Index
}

func NewIndex() Index {
	internal := cache.NewIndex()

	return Index{
		index: &internal,
	}
}

func (r *Index) Reset() {
	r.index.Reset()
}

func (r *Index) Add(name string, size int64, createdAt time.Time) {
	r.index.Add(name, size, createdAt)
}

func (r *Index) Hit(name string) {
	r.index.Hit(name)
}

func (r *Index) DeepCopy() *Index {
	cp := r.index.DeepCopy()

	return &Index{
		index: cp,
	}
}

func (r *Index) PriorityList() []cache.IndexEntry {
	return r.index.PriorityList()
}

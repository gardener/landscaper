// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/api/resource"
)

// GCHighThreshold defines the default percent of disk usage which triggers files garbage collection.
const GCHighThreshold float64 = 0.85

// GCLowThreshold defines the default percent of disk usage to which files garbage collection attempts to free.
const GCLowThreshold float64 = 0.80

// ResetInterval defines the default interval when the hit reset should run.
const ResetInterval time.Duration = 1 * time.Hour

// PreservedHitsProportion defines the default percent of hits that should be preserved.
const PreservedHitsProportion = 0.5

// GarbageCollectionConfiguration contains all options for the cache garbage collection.
type GarbageCollectionConfiguration struct {
	// Size is the size of the filesystem.
	// If the value is 0 there is no limit and no garbage collection will happen.
	// See the kubernetes quantity docs for detailed description of the format
	// https://github.com/kubernetes/apimachinery/blob/master/pkg/api/resource/quantity.go
	Size string
	// GCHighThreshold defines the percent of disk usage which triggers files garbage collection.
	GCHighThreshold float64
	// GCLowThreshold defines the percent of disk usage to which files garbage collection attempts to free.
	GCLowThreshold float64
	// ResetInterval defines the interval when the hit reset should run.
	ResetInterval time.Duration
	// PreservedHitsProportion defines the percent of hits that should be preserved.
	PreservedHitsProportion float64
}

// FileSystem is a internal representation of FileSystem with a optional max size
// and a garbage collection.
type FileSystem struct {
	log logr.Logger
	mux sync.Mutex

	// Configuration
	vfs.FileSystem
	// Size is the size of the filesystem in bytes.
	// If the value is 0 there is no limit and no garbage collection will happen.
	Size int64
	// GCHighThreshold defines the percent of disk usage which triggers files garbage collection.
	GCHighThreshold float64
	// GCLowThreshold defines the percent of disk usage to which files garbage collection attempts to free.
	GCLowThreshold float64
	// ResetInterval defines the interval when the hit reset should run.
	ResetInterval time.Duration
	// PreservedHitsProportion defines the percent of hits that should be preserved.
	PreservedHitsProportion float64

	index Index
	// currentSize is the current size of the filesystem.
	currentSize   int64
	resetStopChan chan struct{}

	// optional metrics
	itemsCountMetric prometheus.Gauge
	diskUsageMetric  prometheus.Gauge
	hitsCountMetric  prometheus.Counter
}

// ApplyOptions parses and applies the options to the filesystem.
// It also applies defaults
func (o GarbageCollectionConfiguration) ApplyOptions(fs *FileSystem) error {
	if len(o.Size) == 0 {
		// no garbage collection configured ignore all other values
		return nil
	}
	quantity, err := resource.ParseQuantity(o.Size)
	if err != nil {
		return fmt.Errorf("unable to parse size %q: %w", o.Size, err)
	}
	sizeInBytes, ok := quantity.AsInt64()
	if ok {
		fs.Size = sizeInBytes
	}

	if o.GCHighThreshold == 0 {
		o.GCHighThreshold = GCHighThreshold
	}
	fs.GCHighThreshold = o.GCHighThreshold

	if o.GCLowThreshold == 0 {
		o.GCLowThreshold = GCLowThreshold
	}
	fs.GCLowThreshold = o.GCLowThreshold

	if o.ResetInterval == 0 {
		o.ResetInterval = ResetInterval
	}
	fs.ResetInterval = o.ResetInterval

	if o.PreservedHitsProportion == 0 {
		o.PreservedHitsProportion = PreservedHitsProportion
	}
	fs.PreservedHitsProportion = o.PreservedHitsProportion

	return nil
}

// Merge merges 2 gc configurations whereas the defined values are overwritten.
func (o GarbageCollectionConfiguration) Merge(cfg *GarbageCollectionConfiguration) {
	if len(o.Size) != 0 {
		cfg.Size = o.Size
	}
	if o.GCHighThreshold != 0 {
		cfg.GCHighThreshold = o.GCHighThreshold
	}
	if o.GCLowThreshold != 0 {
		cfg.GCLowThreshold = o.GCLowThreshold
	}
	if o.ResetInterval != 0 {
		cfg.ResetInterval = o.ResetInterval
	}
	if o.PreservedHitsProportion != 0 {
		cfg.PreservedHitsProportion = o.PreservedHitsProportion
	}
}

// NewCacheFilesystem creates a new FileSystem cache.
// It acts as a replacement for a vfs filesystem that restricts the size of the filesystem.
// The garbage collection is triggered when a file is created.
// When the filesystem is not used anymore fs.Close() should be called to gracefully free resources.
func NewCacheFilesystem(log logr.Logger, fs vfs.FileSystem, gcOpts GarbageCollectionConfiguration) (*FileSystem, error) {
	cFs := &FileSystem{
		log:        log,
		FileSystem: fs,
		index:      NewIndex(),
	}
	if err := gcOpts.ApplyOptions(cFs); err != nil {
		return nil, err
	}

	// load all cached files from the filesystem
	files, err := vfs.ReadDir(fs, "/")
	if err != nil {
		return nil, fmt.Errorf("unable to read current cached files: %w", err)
	}
	for _, file := range files {
		cFs.currentSize = cFs.currentSize + file.Size()
		cFs.index.Add(file.Name(), file.Size(), file.ModTime())
	}

	if cFs.Size != 0 {
		// start hit reset counter
		cFs.StartResetInterval()
	}

	return cFs, nil
}

// WithMetrics adds prometheus metrics to the filesystem
// that are set by the filesystem.
func (fs *FileSystem) WithMetrics(itemsCount, usage prometheus.Gauge, hits prometheus.Counter) {
	fs.diskUsageMetric = usage
	fs.hitsCountMetric = hits
	fs.itemsCountMetric = itemsCount

	if fs.diskUsageMetric != nil {
		fs.diskUsageMetric.Set(float64(fs.CurrentSize()))
	}
	if fs.itemsCountMetric != nil {
		fs.itemsCountMetric.Set(float64(fs.index.Len()))
	}
}

// Close implements the io.Closer interface.
// It should be called when the cache is not used anymore.
func (fs *FileSystem) Close() error {
	if fs.resetStopChan == nil {
		return nil
	}
	fs.resetStopChan <- struct{}{}
	return nil
}

var _ io.Closer = &FileSystem{}

// StartResetInterval starts the reset counter for the cache hits.
func (fs *FileSystem) StartResetInterval() {
	interval := time.NewTicker(ResetInterval)
	fs.resetStopChan = make(chan struct{})
	go func() {
		for {
			select {
			case <-interval.C:
				fs.index.Reset()
			case <-fs.resetStopChan:
				interval.Stop()
				return
			}
		}
	}()
}

func (fs *FileSystem) Create(path string, size int64) (vfs.File, error) {
	fs.mux.Lock()
	defer fs.mux.Unlock()
	file, err := fs.FileSystem.Create(path)
	if err != nil {
		return nil, err
	}
	fs.setCurrentSize(fs.currentSize + size)
	fs.index.Add(path, size, time.Now())
	if fs.itemsCountMetric != nil {
		fs.itemsCountMetric.Inc()
	}
	go fs.RunGarbageCollection()
	return file, err
}

func (fs *FileSystem) OpenFile(name string, flags int, perm os.FileMode) (vfs.File, error) {
	fs.index.Hit(name)
	if fs.hitsCountMetric != nil {
		fs.hitsCountMetric.Inc()
	}
	return fs.FileSystem.OpenFile(name, flags, perm)
}

func (fs *FileSystem) Remove(name string) error {
	if err := fs.FileSystem.Remove(name); err != nil {
		return err
	}
	entry := fs.index.Get(name)
	fs.setCurrentSize(fs.currentSize - entry.Size)
	fs.index.Remove(name)
	if fs.itemsCountMetric != nil {
		fs.itemsCountMetric.Dec()
	}
	return nil
}

// DeleteAll removes all files in the filesystem
func (fs *FileSystem) DeleteAll() error {
	fs.mux.Lock()
	defer fs.mux.Unlock()
	files, err := vfs.ReadDir(fs.FileSystem, "/")
	if err != nil {
		return fmt.Errorf("unable to read current cached files: %w", err)
	}
	for _, file := range files {
		if err := fs.Remove(file.Name()); err != nil {
			return err
		}
	}
	return nil
}

// CurrentSize returns the current size of the filesystem
func (fs *FileSystem) CurrentSize() int64 {
	return fs.currentSize
}

// setCurrentSize sets the current size of the filesystem
func (fs *FileSystem) setCurrentSize(size int64) {
	fs.currentSize = size
	if fs.diskUsageMetric != nil {
		fs.diskUsageMetric.Set(float64(size))
	}
}

// usage returns the current disk usage of the filesystem
func (fs *FileSystem) usage() float64 {
	return float64(fs.currentSize) / float64(fs.Size)
}

// RunGarbageCollection runs the garbage collection of the filesystem.
// The gc only deletes items when the max size reached a certain threshold.
// If that threshold is reached the files are deleted with the following priority:
// - least hits
// - oldest
// - random
func (fs *FileSystem) RunGarbageCollection() {
	// do not run gc if the size is infinite
	if fs.Size == 0 {
		return
	}
	// first check if we reached the threshold to start garbage collection
	if usage := fs.usage(); usage < GCHighThreshold {
		fs.log.V(10).Info(fmt.Sprintf("run gc with %v%% usage", usage))
		return
	}

	// while the index is read and copied no write should happen
	fs.mux.Lock()
	index := fs.index.DeepCopy()
	fs.mux.Unlock()

	// while the garbage collection is running read operations are blocked
	// todo: improve to only block read operations on really deleted objects
	fs.mux.Lock()
	defer fs.mux.Unlock()

	// sort all files according to their deletion priority
	items := index.PriorityList()
	for fs.usage() > GCLowThreshold {
		if len(items) == 0 {
			return
		}
		item := items[0]
		if err := fs.Remove(item.Name); err != nil {
			fs.log.Error(err, "unable to delete file", "file", item.Name)
		}
		// remove currently garbage collected item
		items = items[1:]
	}
}

type Index struct {
	mut     sync.RWMutex
	entries map[string]IndexEntry
}

// NewIndex creates a new index structure
func NewIndex() Index {
	return Index{
		mut:     sync.RWMutex{},
		entries: map[string]IndexEntry{},
	}
}

type IndexEntry struct {
	// Name is the name if the file.
	Name string
	// Size is the size of the file in bytes.
	Size int64
	// Is the number of hits since the last gc
	Hits int64
	// CreatedAt is the time when the file ha been created.
	CreatedAt time.Time
	// HitsSinceLastReset is the number hits since the last reset interval
	HitsSinceLastReset int64
}

// Add adds a entry to the index.
func (i *Index) Add(name string, size int64, createdAt time.Time) {
	i.entries[name] = IndexEntry{
		Name:               name,
		Size:               size,
		Hits:               0,
		CreatedAt:          createdAt,
		HitsSinceLastReset: 0,
	}
}

// Len returns the number of items that are currently in the index.
func (i *Index) Len() int {
	return len(i.entries)
}

// Get return the index entry with the given name.
func (i *Index) Get(name string) IndexEntry {
	i.mut.RLock()
	defer i.mut.RUnlock()
	return i.entries[name]
}

// Remove removes the entry from the index.
func (i *Index) Remove(name string) {
	i.mut.Lock()
	defer i.mut.Unlock()
	delete(i.entries, name)
}

// Hit increases the hit count for the file.
func (i *Index) Hit(name string) {
	i.mut.Lock()
	defer i.mut.Unlock()
	entry, ok := i.entries[name]
	if !ok {
		return
	}
	entry.Hits++
	entry.HitsSinceLastReset++
	i.entries[name] = entry
}

// Reset resets the hit counter for all entries.
// The reset preserves 20% if the old hits.
func (i *Index) Reset() {
	for name := range i.entries {
		entry := i.Get(name)

		oldHits := entry.Hits - entry.HitsSinceLastReset
		preservedHits := int64(float64(oldHits) * PreservedHitsProportion)
		entry.Hits = preservedHits + entry.HitsSinceLastReset

		entry.HitsSinceLastReset = 0
		i.entries[name] = entry
	}
}

// DeepCopy creates a deep copy of the current index.
func (i *Index) DeepCopy() *Index {
	index := &Index{
		entries: make(map[string]IndexEntry, len(i.entries)),
	}
	for key, value := range i.entries {
		index.entries[key] = value
	}
	return index
}

// PriorityList returns a entries of all entries
// sorted by their gc priority.
// The entry with the lowest priority is the first item.
func (i *Index) PriorityList() []IndexEntry {
	p := priorityList{
		entries: make([]IndexEntry, 0),
	}
	for _, i := range i.entries {
		entry := i
		if entry.Hits > p.maxHits {
			p.maxHits = entry.Hits
		}
		if p.minHits == 0 || entry.Hits < p.minHits {
			p.minHits = entry.Hits
		}
		if p.newest.Before(entry.CreatedAt) {
			p.newest = entry.CreatedAt
		}
		if p.oldest.IsZero() || entry.CreatedAt.Before(p.oldest) {
			p.oldest = entry.CreatedAt
		}
		p.entries = append(p.entries, entry)
	}
	sort.Sort(p)
	return p.entries
}

// priorityList is a helper type that implements the sort.Sort function.
// the entries are sorted by their priority.
// The priority is calculated using the entries hits and creation date.
type priorityList struct {
	minHits, maxHits int64
	oldest, newest   time.Time
	entries          []IndexEntry
}

var _ sort.Interface = priorityList{}

func (i priorityList) Len() int { return len(i.entries) }

func (i priorityList) Less(a, b int) bool {
	eA, eB := i.entries[a], i.entries[b]

	// calculate the entries hits value
	// based on the min and max values
	priorityA := CalculatePriority(eA, i.minHits, i.maxHits, i.oldest, i.newest)
	priorityB := CalculatePriority(eB, i.minHits, i.maxHits, i.oldest, i.newest)
	return priorityA < priorityB
}

// CalculatePriority calculates the gc priority of a index entry.
// A lower priority means that is more likely to be deleted.
func CalculatePriority(entry IndexEntry, minHits, maxHits int64, oldest, newest time.Time) float64 {
	hitsVal := float64(entry.Hits-minHits) / float64(maxHits-minHits)
	if math.IsNaN(hitsVal) {
		hitsVal = 0
	}
	dateVal := float64(entry.CreatedAt.UnixNano()-oldest.UnixNano()) / float64(newest.UnixNano()-oldest.UnixNano())
	if math.IsNaN(dateVal) {
		dateVal = 0
	}

	return (hitsVal * 0.6) + (dateVal * 0.4)
}

func (i priorityList) Swap(a, b int) { i.entries[a], i.entries[b] = i.entries[b], i.entries[a] }

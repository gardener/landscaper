// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gardener/component-cli/ociclient/cache"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/readonlyfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/gardener/landscaper/apis/config/v1alpha1"

	"github.com/gardener/landscaper/pkg/utils/tar"

	"github.com/gardener/landscaper/apis/config"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/utils"
)

var storeSingleton *Store

// GetStore returns the currently active store.
func GetStore() *Store {
	return storeSingleton
}

func init() {
	// init default blueprint store
	store, err := DefaultStore(memoryfs.New())
	if err != nil {
		panic(err)
	}
	SetStore(store)
}

// SetStore returns the currently active store.
func SetStore(store *Store) {
	if storeSingleton != nil {
		_ = storeSingleton.Close()
	}
	storeSingleton = store
}

var NotFoundError = errors.New("NOTFOUND")
var StoreClosedError = errors.New("STORE_CLOSED")

// Store describes a blueprint cache using a base filesystem.
// The blueprints are stored decompressed and untarred in the root of the filesystem using a hash value.
// root
// - <some hash>
//   - blueprint.yaml
//   - ... some other data
//
// The hash is calculated using the component descriptor and the name of the blueprint.
type Store struct {
	log      logr.Logger
	disabled bool
	mux      sync.RWMutex
	index    cache.Index
	fs       vfs.FileSystem

	size        int64
	currentSize int64
	// usage describes the actual usage of the filesystem.
	// It calculated by using the max size and the current size.
	usage float64

	gcConfig      config.GarbageCollectionConfiguration
	resetStopChan chan struct{}
	closed        bool
}

// NewStore creates a new blueprint cache using a base filesystem.
//
// The caller should always close the cache for a graceful termination.
//
// The store should be initialized once as this is a global singleton.
func NewStore(log logr.Logger, baseFs vfs.FileSystem, config config.BlueprintStore) (*Store, error) {
	if log == nil {
		log = logr.Discard()
	}

	if len(config.Path) == 0 {
		var err error
		config.Path, err = vfs.TempDir(baseFs, baseFs.FSTempDir(), "bsStore")
		if err != nil {
			return nil, fmt.Errorf("unable to setup temporary directory for blueprint store: %w", err)
		}
	}

	fs, err := projectionfs.New(baseFs, config.Path)
	if err != nil {
		return nil, fmt.Errorf("unable to create store filesystem: %w", err)
	}

	store := &Store{
		log:      log,
		disabled: config.DisableCache,
		index:    cache.NewIndex(),
		fs:       fs,
		gcConfig: config.GarbageCollectionConfiguration,
	}

	if config.Size != "0" {
		quantity, err := resource.ParseQuantity(config.Size)
		if err != nil {
			return nil, fmt.Errorf("unable to parse size %q: %w", config.Size, err)
		}
		sizeInBytes, ok := quantity.AsInt64()
		if !ok {
			return nil, fmt.Errorf("unable to parse size %q as int", config.Size)
		}
		store.size = sizeInBytes

		store.StartResetInterval()
	}
	return store, nil
}

// DefaultStore creates a default blueprint store.
func DefaultStore(fs vfs.FileSystem) (*Store, error) {
	defaultStoreConfig := config.BlueprintStore{}
	cs := v1alpha1.BlueprintStore{}
	v1alpha1.SetDefaults_BlueprintStore(&cs)
	if err := v1alpha1.Convert_v1alpha1_BlueprintStore_To_config_BlueprintStore(&cs, &defaultStoreConfig, nil); err != nil {
		return nil, err
	}
	return NewStore(logr.Discard(), fs, defaultStoreConfig)
}

func (s *Store) Close() error {
	s.closed = true
	close(s.resetStopChan)
	return nil
}

// Store stores a blueprint on the given filesystem.
// It is expected that the bpReader contains a tar archive.
func (s *Store) Store(ctx context.Context, cd *cdv2.ComponentDescriptor, resource cdv2.Resource, bpReader io.Reader) (*Blueprint, error) {
	if s.closed {
		return nil, StoreClosedError
	}
	bpID := blueprintID(cd, resource)
	bpPath := blueprintPath(bpID)
	if bp, err := s.Get(ctx, cd, resource); err == nil {
		return bp, nil
	}

	s.mux.Lock()
	defer s.mux.Unlock()
	if _, err := s.fs.Stat(bpPath); err == nil {
		if err := s.fs.RemoveAll(bpPath); err != nil {
			s.log.Error(err, "unable to cleanup directory")
			return nil, errors.New("unable to cleanup store")
		}
	}

	if err := s.fs.Mkdir(bpPath, os.ModePerm); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("unable to create bluprint directory: %w", err)
	}
	if err := tar.ExtractTar(ctx, bpReader, s.fs, tar.ToPath(bpPath), tar.Overwrite(true)); err != nil {
		return nil, fmt.Errorf("unable to extract blueprint from blob: %w", err)
	}

	size, err := utils.GetSizeOfDirectory(s.fs, bpPath)
	if err != nil {
		return nil, fmt.Errorf("unable to get size of blueprint directory: %w", err)
	}
	s.index.Add(bpID, size, time.Now())
	s.updateUsage(size)
	StoredItems.Inc()
	defer func() {
		go s.RunGarbageCollection()
	}()

	return s.buildBlueprint(bpID)
}

// CurrentSize returns the current used storage.
func (s *Store) CurrentSize() int64 {
	return s.currentSize
}

func (s *Store) updateUsage(size int64) {
	s.currentSize = s.currentSize + size
	s.usage = float64(s.currentSize) / float64(s.size)
	DiskUsage.Set(float64(s.currentSize))
}

// Get reads the blueprint from the filesystem.
func (s *Store) Get(_ context.Context, cd *cdv2.ComponentDescriptor, resource cdv2.Resource) (*Blueprint, error) {
	if s.closed {
		return nil, StoreClosedError
	}
	if s.disabled {
		return nil, NotFoundError
	}
	bpID := blueprintID(cd, resource)
	bpPath := blueprintPath(bpID)
	s.mux.RLock()
	defer s.mux.RUnlock()
	if _, err := s.fs.Stat(bpPath); err != nil {
		if os.IsNotExist(err) {
			return nil, NotFoundError
		}
		return nil, err
	}
	s.index.Hit(bpID)
	return s.buildBlueprint(bpPath)
}

func (s *Store) buildBlueprint(bpID string) (*Blueprint, error) {
	blueprintBytes, err := vfs.ReadFile(s.fs, filepath.Join(bpID, lsv1alpha1.BlueprintFileName))
	if err != nil {
		return nil, fmt.Errorf("unable to read blueprint definition: %w", err)
	}
	blueprint := &lsv1alpha1.Blueprint{}
	if _, _, err := api.Decoder.Decode(blueprintBytes, nil, blueprint); err != nil {
		return nil, fmt.Errorf("unable to decode blueprint definition: %w", err)
	}
	bpFs, err := projectionfs.New(readonlyfs.New(s.fs), bpID)
	if err != nil {
		return nil, fmt.Errorf("unable to create blueprint filesystem: %w", err)
	}
	return New(blueprint, readonlyfs.New(bpFs)), nil
}

////////////////////////
// Garbage Collection //
///////////////////////

// StartResetInterval starts the reset counter for the cache hits.
func (s *Store) StartResetInterval() {
	interval := time.NewTicker(s.gcConfig.ResetInterval.Duration)
	s.resetStopChan = make(chan struct{})
	go func() {
		for {
			select {
			case <-interval.C:
				s.index.Reset()
			case <-s.resetStopChan:
				interval.Stop()
				return
			}
		}
	}()
}

// RunGarbageCollection runs the garbage collection of the filesystem.
// The gc only deletes items when the max size reached a certain threshold.
// If that threshold is reached the files are deleted with the following priority:
// - least hits
// - oldest
// - random
func (s *Store) RunGarbageCollection() {
	// do not run gc if the size is infinite
	if s.size == 0 {
		return
	}
	// while the garbage collection is running read operations are blocked
	// todo: improve to only block read operations on really deleted objects
	s.mux.Lock()
	defer s.mux.Unlock()

	// first check if we reached the threshold to start garbage collection
	if s.usage < s.gcConfig.GCHighThreshold {
		s.log.V(10).Info(fmt.Sprintf("run gc with %v%% usage", s.usage))
		return
	}

	// while the index is read and copied no write should happen
	index := s.index.DeepCopy()

	// sort all files according to their deletion priority
	items := index.PriorityList()
	for s.usage > s.gcConfig.GCLowThreshold {
		if len(items) == 0 {
			return
		}
		item := items[0]
		if err := s.fs.RemoveAll(blueprintPath(item.Name)); err != nil {
			s.log.Error(err, "unable to delete blueprint directory", "file", item.Name)
		}
		s.log.V(7).Info("garbage collected", "item", item.Name)
		s.updateUsage(-item.Size)
		StoredItems.Dec()
		// remove currently garbage collected item
		items = items[1:]
	}
}

// blueprintID generates a unique blueprint id that can be used a a file/directory name.
// The ID is calculated by hashing (sha256) the component descriptor and the blueprint resource.
func blueprintID(cd *cdv2.ComponentDescriptor, resource cdv2.Resource) string {
	h := sha256.New()
	_, _ = h.Write([]byte(fmt.Sprintf("%s-%s-%s-%s", cd.Name, cd.Version, resource.Name, resource.Version)))
	return hex.EncodeToString(h.Sum(nil))
}

func blueprintPath(bpID string) string {
	return filepath.Join("/", bpID)
}

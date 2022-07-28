// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints

import (
	"compress/gzip"
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
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/readonlyfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/api/resource"
	errorsutil "k8s.io/apimachinery/pkg/util/errors"

	"github.com/gardener/landscaper/apis/mediatype"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"

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
	log         logging.Logger
	disabled    bool
	mux         sync.RWMutex
	indexMethod config.IndexMethod
	index       cache.Index
	fs          vfs.FileSystem

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
func NewStore(log logging.Logger, baseFs vfs.FileSystem, config config.BlueprintStore) (*Store, error) {
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
		log:         log,
		disabled:    config.DisableCache,
		indexMethod: config.IndexMethod,
		index:       cache.NewIndex(),
		fs:          fs,
		gcConfig:    config.GarbageCollectionConfiguration,
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
	return NewStore(logging.Discard(), fs, defaultStoreConfig)
}

func (s *Store) Close() error {
	s.closed = true
	close(s.resetStopChan)
	return nil
}

// Fetch fetches the blueprint from the store or the remote.
// The blueprint is automatically cached once downloaded from the remote endpoint.
func (s *Store) Fetch(ctx context.Context,
	cd *cdv2.ComponentDescriptor,
	blobResolver ctf.BlobResolver,
	blueprintName string) (*Blueprint, error) {

	// get blueprint resource from component descriptor
	resource, err := GetBlueprintResourceFromComponentDescriptor(cd, blueprintName)
	if err != nil {
		return nil, err
	}

	var (
		blueprintID string
		blobInfo    *ctf.BlobInfo
	)
	switch s.indexMethod {
	case config.ComponentDescriptorIdentityMethod:
		blueprintID = blueprintIDFromComponentDescriptor(cd, resource)
	case config.BlueprintDigestIndex:
		blobInfo, err = blobResolver.Info(ctx, resource)
		if err != nil {
			return nil, fmt.Errorf("unable to get blob info: %w", err)
		}
		blueprintID = blobInfo.Digest
	default:
		return nil, fmt.Errorf("unknown blueprint index method %q", s.indexMethod)
	}

	if s.indexMethod == config.BlueprintDigestIndex {
		// read the digest directly if the digest index is used
		blobInfo, err = blobResolver.Info(ctx, resource)
		if err != nil {
			return nil, fmt.Errorf("unable to get blob info: %w", err)
		}
	}

	if blueprint, err := s.Get(ctx, blueprintID); err == nil {
		return blueprint, nil
	}

	return s.Store(ctx, blobResolver, resource, blueprintID, blobInfo)
}

// Store stores a blueprint on the given filesystem.
// It is expected that the bpReader contains a tar archive.
// The blobInfo is optional and will be fetched from the BlobResolver if not defined.
func (s *Store) Store(ctx context.Context, blobResolver ctf.BlobResolver, resource cdv2.Resource, blueprintID string, blobInfo *ctf.BlobInfo) (*Blueprint, error) {
	if s.closed {
		return nil, StoreClosedError
	}

	bpPath := blueprintPath(blueprintID)
	if bp, err := s.Get(ctx, blueprintID); err == nil {
		// this should never happen when used with the Fetch method.
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

	if blobInfo == nil {
		var err error
		blobInfo, err = blobResolver.Info(ctx, resource)
		if err != nil {
			return nil, fmt.Errorf("unable to get blob info: %w", err)
		}
	}
	if err := FetchAndExtractBlueprint(ctx, s.fs, bpPath, blobResolver, resource, blobInfo); err != nil {
		return nil, err
	}

	size, err := utils.GetSizeOfDirectory(s.fs, bpPath)
	if err != nil {
		return nil, fmt.Errorf("unable to get size of blueprint directory: %w", err)
	}
	s.index.Add(blueprintID, size, time.Now())
	s.updateUsage(size)
	StoredItems.Inc()
	defer func() {
		go s.RunGarbageCollection()
	}()

	return buildBlueprintFromPath(s.fs, bpPath)
}

// FetchAndExtractBlueprint fetches a blueprint from a remote blob resolver and extracts the tar to the given path.
func FetchAndExtractBlueprint(
	ctx context.Context,
	fs vfs.FileSystem,
	bpPath string,
	blobResolver ctf.BlobResolver,
	resource cdv2.Resource,
	blobInfo *ctf.BlobInfo) error {

	mediaType, err := mediatype.Parse(blobInfo.MediaType)
	if err != nil {
		return fmt.Errorf("unable to parse media type: %w", err)
	}

	pr, pw := io.Pipe()
	defer pw.Close()
	downloadCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eg, downloadCtx := errgroup.WithContext(downloadCtx)
	eg.Go(func() error {
		_, err := blobResolver.Resolve(downloadCtx, resource, utils.NewContextAwareWriter(downloadCtx, pw))
		if err != nil {
			if err2 := pw.Close(); err2 != nil {
				return errorsutil.NewAggregate([]error{err, err2})
			}
			return fmt.Errorf("unable to resolve blueprint blob: %w", err)
		}
		return pw.Close()
	})

	var blobReader io.Reader = pr
	if mediaType.String() == mediatype.MediaTypeGZip || mediaType.IsCompressed(mediatype.GZipCompression) {
		gr, err := gzip.NewReader(pr)
		if err != nil {
			if err == gzip.ErrHeader {
				return errors.New("expected a gzip compressed tar")
			}
			return err
		}
		blobReader = gr
		defer gr.Close()
	}

	if err := fs.Mkdir(bpPath, os.ModePerm); err != nil && !os.IsExist(err) {
		return fmt.Errorf("unable to create bluprint directory: %w", err)
	}
	if err := tar.ExtractTar(ctx, blobReader, fs, tar.ToPath(bpPath), tar.Overwrite(true)); err != nil {
		return fmt.Errorf("unable to extract blueprint from blob: %w", err)
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
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
func (s *Store) Get(_ context.Context, blueprintID string) (*Blueprint, error) {
	if s.closed {
		return nil, StoreClosedError
	}
	if s.disabled {
		return nil, NotFoundError
	}
	bpPath := blueprintPath(blueprintID)
	s.mux.RLock()
	defer s.mux.RUnlock()
	if _, err := s.fs.Stat(bpPath); err != nil {
		if os.IsNotExist(err) {
			return nil, NotFoundError
		}
		return nil, err
	}
	s.index.Hit(blueprintID)
	return buildBlueprintFromPath(s.fs, bpPath)
}

// buildBlueprintFromPath creates a read-only blueprint from an extracted blueprint.
func buildBlueprintFromPath(fs vfs.FileSystem, bpPath string) (*Blueprint, error) {
	blueprintBytes, err := vfs.ReadFile(fs, filepath.Join(bpPath, lsv1alpha1.BlueprintFileName))
	if err != nil {
		return nil, fmt.Errorf("unable to read blueprint definition: %w", err)
	}
	blueprint := &lsv1alpha1.Blueprint{}
	if _, _, err := api.Decoder.Decode(blueprintBytes, nil, blueprint); err != nil {
		return nil, fmt.Errorf("unable to decode blueprint definition: %w", err)
	}
	bpFs, err := projectionfs.New(readonlyfs.New(fs), bpPath)
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
		s.log.Debug(fmt.Sprintf("run gc with %v%% usage", s.usage))
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
		s.log.Debug("garbage collected", "item", item.Name)
		s.updateUsage(-item.Size)
		StoredItems.Dec()
		// remove currently garbage collected item
		items = items[1:]
	}
}

// blueprintID generates a unique blueprint id that can be used a a file/directory name.
// The ID is calculated by hashing (sha256) the component descriptor and the blueprint resource.
func blueprintIDFromComponentDescriptor(cd *cdv2.ComponentDescriptor, resource cdv2.Resource) string {
	h := sha256.New()
	if cd.GetEffectiveRepositoryContext() != nil {
		_, _ = h.Write(cd.GetEffectiveRepositoryContext().Raw)
	}
	_, _ = h.Write([]byte(fmt.Sprintf("%s-%s-%s-%s",
		cd.Name,
		cd.Version,
		resource.Name,
		resource.Version)))
	return hex.EncodeToString(h.Sum(nil))
}

func blueprintPath(bpID string) string {
	return filepath.Join("/", bpID)
}

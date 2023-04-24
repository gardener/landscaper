package flow

import (
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/dynaml"
	"github.com/mandelsoft/spiff/features"
	"github.com/mandelsoft/spiff/yaml"
)

const MODE_FILE_ACCESS = 1 // support file system access
const MODE_OS_ACCESS = 2   // support os commands like pipe and exec

type execCache struct {
	cache map[string][]byte
	lock  sync.Mutex
}

func (c *execCache) Lock() {
	c.lock.Lock()
}

func (c *execCache) Unlock() {
	c.lock.Unlock()
}

func (c *execCache) Clear() {
	c.cache = make(map[string][]byte)
}

func (c *execCache) Get(key string) []byte {
	return c.cache[key]
}

func (c *execCache) Set(key string, content []byte) {
	c.cache[key] = content
}

var _ dynaml.ExecCache = &execCache{}

type State struct {
	files      map[string]string // content hash to temp file name
	fileCache  map[string][]byte // file content cache
	key        string            // default encryption key
	mode       int
	exec_cache dynaml.ExecCache // execution cache
	fileSystem vfs.VFS          // virtual filesystem to use for filesystem based operations
	registry   dynaml.Registry
	features   features.FeatureFlags
	tags       map[string]*dynaml.TagInfo
	docno      int // document number
}

var _ dynaml.State = &State{}

func NewState(key string, mode int, optfs ...vfs.FileSystem) *State {
	var fs vfs.FileSystem
	if len(optfs) > 0 {
		fs = optfs[0]
	}
	if fs == nil {
		fs = osfs.New()
	} else {
		mode = mode & ^MODE_OS_ACCESS
	}
	return &State{
		tags:       map[string]*dynaml.TagInfo{},
		files:      map[string]string{},
		fileCache:  map[string][]byte{},
		key:        key,
		mode:       mode,
		exec_cache: &execCache{cache: make(map[string][]byte)},
		fileSystem: vfs.New(fs),
		docno:      1,
		features:   features.Features(),
		registry:   dynaml.DefaultRegistry(),
	}
}

func NewDefaultState() *State {
	return NewState(features.EncryptionKey(), MODE_OS_ACCESS|MODE_FILE_ACCESS)
}

func (s *State) SetRegistry(r dynaml.Registry) *State {
	if r == nil {
		r = dynaml.DefaultRegistry()
	}
	s.registry = r
	return s
}
func (s *State) SetFeatures(f features.FeatureFlags) *State {
	s.features = f
	return s
}

func (s *State) SetTags(tags ...*dynaml.Tag) *State {
	s.tags = map[string]*dynaml.TagInfo{}
	for _, v := range tags {
		s.tags[v.Name()] = dynaml.NewTagInfo(v)
	}
	return s
}

func (s *State) SetInterpolation(b bool) *State {
	s.features.SetInterpolation(b)
	return s
}

func (s *State) InterpolationEnabled() bool {
	return s.features.InterpolationEnabled()
}

func (s *State) SetControl(b bool) *State {
	s.features.SetControl(b)
	return s
}

func (s *State) ControlEnabled() bool {
	return s.features.ControlEnabled()
}

func (s *State) OSAccessAllowed() bool {
	return s.mode&MODE_OS_ACCESS != 0
}

func (s *State) FileAccessAllowed() bool {
	return s.mode&MODE_FILE_ACCESS != 0
}

func (s *State) FileSystem() vfs.VFS {
	return s.fileSystem
}

func (s *State) GetRegistry() dynaml.Registry {
	if s == nil {
		return dynaml.DefaultRegistry()
	}
	return s.registry
}

func (s *State) GetFeatures() features.FeatureFlags {
	if s == nil {
		return nil
	}
	return s.features
}

func (s *State) GetEncryptionKey() string {
	return s.key
}

func (s *State) GetExecCache() dynaml.ExecCache {
	return s.exec_cache
}

func (s *State) GetTempName(data []byte) (string, error) {
	if !s.FileAccessAllowed() {
		return "", fmt.Errorf("tempname: no OS operations supported in this execution environment")
	}
	sum := sha512.Sum512(data)
	hash := base64.StdEncoding.EncodeToString(sum[:])

	name, ok := s.files[hash]
	if !ok {
		file, err := s.fileSystem.TempFile("", "spiff-")
		if err != nil {
			return "", err
		}
		name = file.Name()
		s.files[hash] = name
	}
	return name, nil
}

func (s *State) SetTag(name string, node yaml.Node, path []string, scope dynaml.TagScope) error {
	name = strings.Replace(name, ":", ".", -1)
	debug.Debug("setting tag: %v\n", path)
	old := s.tags[name]
	if old != nil {
		if !old.IsLocal() {
			return fmt.Errorf("duplicate tag %q: %s in foreign document", name, strings.Join(path, "."))
		}
		if !reflect.DeepEqual(path, old.Path()) {
			return fmt.Errorf("duplicate tag %q: %s <-> %s", name, strings.Join(path, "."), strings.Join(old.Path(), "."))
		}
	}
	s.tags[name] = dynaml.NewTagInfo(dynaml.NewTag(name, Cleanup(node, discardTags), path, scope))
	return nil
}

func (s *State) GetTag(name string) *dynaml.Tag {
	name = strings.Replace(name, ":", ".", -1)
	if strings.HasPrefix(name, "doc.") {
		i, err := strconv.Atoi(name[4:])
		if err != nil {
			return nil
		}
		if i <= 0 {
			i += s.docno
			if i <= 0 {
				return nil
			}
			name = fmt.Sprintf("doc.%d", i)
		}
	}
	tag := s.tags[name]
	if tag == nil {
		return nil
	}
	return tag.Tag()
}

func (s *State) GetTags(name string) []*dynaml.TagInfo {
	name = strings.Replace(name, ":", ".", -1)
	if strings.HasPrefix(name, "doc.") {
		i, err := strconv.Atoi(name[4:])
		if err != nil {
			return nil
		}
		if i <= 0 {
			i += s.docno
			if i <= 0 {
				return nil
			}
			name = fmt.Sprintf("doc.%d", i)
		}
		tag := s.tags[name]
		if tag == nil {
			return nil
		}
		return []*dynaml.TagInfo{tag}
	}

	var list []*dynaml.TagInfo
	prefix := name + "."
	for _, t := range s.tags {
		if t.Name() == name || strings.HasPrefix(t.Name(), prefix) {
			list = append(list, t)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Level() != list[j].Level() {
			return list[i].Level() < list[j].Level()
		}
		return strings.Compare(list[i].Name(), list[j].Name()) < 0
	})
	return list
}

func (s *State) ResetTags() {
	s.tags = map[string]*dynaml.TagInfo{}
	s.docno = 1
}

func (s *State) ResetStream() {
	n := map[string]*dynaml.TagInfo{}
	for _, v := range s.tags {
		if !v.IsStream() {
			n[v.Name()] = v
		}
	}
	s.docno = 1
	s.tags = n
}

func (s *State) PushDocument(node yaml.Node) {
	if node != nil {
		s.SetTag(fmt.Sprintf("doc.%d", s.docno), node, nil, dynaml.TAG_SCOPE_GLOBAL)
	}
	for _, t := range s.tags {
		t.ResetLocal()
	}
	s.docno++
}

func (s *State) Cleanup() {
	for _, n := range s.files {
		s.fileSystem.Remove(n)
	}
	s.files = map[string]string{}
}

func (s *State) GetFileContent(file string, cached bool) ([]byte, error) {
	var err error

	file = dynaml.FilePath(file)
	data := s.fileCache[file]
	if !cached || data == nil {
		debug.Debug("reading file %s\n", file)
		if strings.HasPrefix(file, "http:") || strings.HasPrefix(file, "https:") {
			response, err := http.Get(file)
			if err != nil {
				return nil, fmt.Errorf("error getting [%s]: %s", file, err)
			} else {
				defer response.Body.Close()
				contents, err := ioutil.ReadAll(response.Body)
				if err != nil {
					return nil, fmt.Errorf("error getting body [%s]: %s", file, err)
				}
				data = contents
			}
		} else {
			data, err = s.fileSystem.ReadFile(file)
			if err != nil {
				return nil, fmt.Errorf("error reading [%s]: %s", path.Clean(file), err)
			}
		}
		s.fileCache[file] = data
	}
	return data, nil
}

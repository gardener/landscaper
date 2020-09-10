package flow

import (
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/dynaml"
)

const MODE_FILE_ACCESS = 1 // support file system access
const MODE_OS_ACCESS = 2   // support os commands like pipe and exec

type State struct {
	files      map[string]string // content hash to temp file name
	fileCache  map[string][]byte // file content cache
	key        string            // default encryption key
	mode       int
	fileSystem vfs.VFS // virtual filesystem to use for filesystem based operations
	functions  dynaml.Registry
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
		files:      map[string]string{},
		fileCache:  map[string][]byte{},
		key:        key,
		mode:       mode,
		fileSystem: vfs.New(fs),
	}
}

func (s *State) SetFunctions(f dynaml.Registry) *State {
	s.functions = f
	return s
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

func (s *State) GetFunctions() dynaml.Registry {
	return s.functions
}

func (s *State) GetEncryptionKey() string {
	return s.key
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

func (s *State) Cleanup() {
	for _, n := range s.files {
		s.fileSystem.Remove(n)
	}
	s.files = map[string]string{}
}

func (s *State) GetFileContent(file string, cached bool) ([]byte, error) {
	var err error

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

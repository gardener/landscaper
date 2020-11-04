// Code generated for package jsonscheme by go-bindata DO NOT EDIT. (@generated)
// sources:
// ../../../../component-descriptor-v2-schema.yaml
package jsonscheme

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _ComponentDescriptorV2SchemaYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xe4\x58\x4f\x8f\xe3\x34\x14\xbf\xe7\x53\x3c\x69\x46\x32\x48\x9b\x2d\xda\x0b\x52\x2f\x68\xd9\x95\x10\x97\x45\x1a\x0d\x5c\xd0\x1c\x5c\xe7\x25\xf5\x2a\xb1\x83\xed\x94\x29\x88\xef\x8e\x6c\xc7\x89\xdd\x36\x69\xba\x2d\xc3\x20\x4e\x95\xd3\xf7\x7e\xef\xff\xcf\x2f\xb9\xe7\xc5\x1a\xc8\xd6\x98\x56\xaf\x57\xab\x8a\xaa\x02\x05\xaa\xb7\xac\x96\x5d\xb1\xd2\x6c\x8b\x0d\xd5\x2b\x26\x9b\x56\x0a\x14\x26\x2f\x50\x33\xc5\x5b\x23\x55\xbe\x7b\x47\xb2\x7b\x2f\x11\x21\x7c\xd6\x52\xe4\xfe\xe9\x5b\xa9\xaa\x55\xa1\x68\x69\xf2\x6f\xbe\xed\xb1\xee\x48\x16\x20\xb8\x14\x6b\x20\x3f\xf4\x16\xe1\x43\xb0\x01\x1f\x07\x1b\xb0\x7b\x07\x5e\xcf\xaa\x95\x5c\x70\xab\xa5\xd7\x19\x40\x83\x86\xda\x5f\x00\xb3\x6f\x71\x0d\x44\x6e\x3e\x23\x33\xc4\x3d\x4a\x4d\x0c\xde\xc3\xe8\xbd\xd3\x2f\xa8\xa1\x5e\x41\xe1\x6f\x1d\x57\x58\x78\x44\x80\x1c\x88\xb7\xfb\x0b\x2a\xcd\xa5\xf0\x52\xad\x92\x2d\x2a\xc3\x51\x07\xb9\x44\x28\x3c\x1c\x5c\xd2\x46\x71\x51\x91\x2c\x03\xa8\xe9\x06\xeb\x49\x7f\x4f\x98\x17\xb4\x41\x32\x1e\x77\xb4\xee\xd0\x21\x29\x6c\xa5\xe6\x46\xaa\xfd\x07\x29\x0c\x3e\x9b\x4b\x50\x37\x54\xe3\xcf\xaa\x8e\x80\xad\xde\x54\x74\xbd\xf4\x64\x5c\xd1\xc3\x79\x11\x00\x14\x5d\xb3\x86\x5f\x89\x64\xfc\x01\x2b\xae\x8d\xda\x93\x27\x1b\x0e\x65\x0c\xb5\x5e\x58\x49\xeb\x90\x93\x82\x52\xaa\x5e\x15\x35\x7c\x65\x4f\xf8\x6c\x50\xd8\x32\xe8\xaf\x27\xc3\xf7\xc1\x66\x00\x15\x37\xdb\x6e\xf3\x7e\xde\xf6\x24\xc0\x70\xb4\xb5\x48\xd3\xa9\xb0\x9c\xca\xe6\x45\x79\xf2\x0e\x92\xa7\xfe\x8f\xde\xd0\x19\x75\x85\xe5\x5c\x0f\x0a\x29\xf0\x9a\x90\xaf\x0c\xe9\x93\x14\xe8\x6b\xae\x65\xa7\x18\x7e\x1c\x06\xfa\x02\x77\xec\x58\x0c\x07\xab\x31\x1c\x7c\x37\x4c\x38\x6a\xd5\x6e\xd9\xc6\x15\x37\x43\x6d\xdc\x68\xeb\x23\x55\xaa\x14\xdd\x8f\x9a\xdc\x60\x13\x09\x01\xdc\xdb\x6a\x01\xb9\x5b\x45\xc4\xb6\x72\x58\x41\x29\x1e\x0d\x77\x16\xfb\x9f\xca\x18\x22\x3f\x0d\xe2\xf5\xc8\x79\xc1\x78\x0a\x16\x88\x5b\x8a\x0f\xc2\x19\xc0\x40\xab\x0f\x58\xa2\x42\xc1\x70\xe1\x14\x53\xdb\xa8\x5e\x03\x8c\x04\x3a\x22\x2d\x65\xc3\x41\xe1\xd3\x01\x4b\xce\xb3\xf5\x5c\x17\xc0\x1d\x50\x66\x3a\x5a\xd7\xfb\xf5\xe8\x50\xee\xc8\xe6\xf7\x15\xe8\x16\x19\xa7\x35\x28\xb4\xf2\xcc\x25\xa4\x47\xda\xcd\xb3\x7f\x02\xac\xb0\xa6\xcf\x58\x80\xc6\x66\x87\xea\xbb\x7f\xae\x83\xdc\x45\xe1\xe7\xec\x71\x68\xec\x0b\xf9\x35\x00\xe8\xc5\x97\x54\x9f\x7e\xb8\x73\xfa\xb5\x64\x3e\x63\x1e\xe5\x0d\x98\x2d\xd7\xd0\x74\xda\x40\x43\x0d\xdb\x46\x75\xd7\x21\x8b\x33\x54\x5b\x53\x33\xd4\xd6\x3d\x8a\xfb\xfc\x8b\x46\xfe\x4c\xe5\x96\x13\x43\x70\x6e\x31\x7f\xb8\xd4\x90\x37\x40\xec\xad\xa5\x04\xad\x5f\x9e\x4d\x66\xa9\x23\x03\x90\x8c\xff\xd8\xd0\xea\xaa\x1b\xc3\x1d\xb9\x45\x19\x38\xe2\x26\x57\x49\xba\x45\xf4\x29\x49\xcc\xcc\xdd\x83\x92\xf1\xef\x6b\x79\xd9\xed\x9f\x84\x95\x03\xa9\xe9\x1e\xd5\x2d\x62\x01\xd2\xbb\x43\xe0\xe9\xd4\x35\x9e\x0e\xe8\x7b\xeb\x7c\x4a\x9f\x66\x8b\xd0\x50\xc1\x4b\xd4\x86\xcc\x1b\x6d\xb0\xe0\xf4\x31\x71\x2e\x85\x7f\xb4\x58\x56\xc8\x93\x80\x2c\x1d\xba\xcf\x8a\x9f\x5e\xdf\x20\xde\x03\x0d\x46\x9e\xb1\x58\xf0\x0a\xb5\x99\x33\xe7\x25\x82\x29\x43\x55\x85\x06\x0b\x60\x76\xb7\x15\xe7\x02\xd2\xfc\x8f\xd9\x58\xec\xff\xc0\x05\x6c\xf6\x06\x75\xb0\xb1\xb1\xc9\x3e\xc4\x15\x5d\xb3\xb1\x05\x8d\x1a\xff\xa1\x27\xae\x6b\xb6\xf6\xf8\x3e\x3a\x35\x1b\xaf\x87\xc2\x92\xe1\x72\xf1\xbf\x12\x4a\x4a\x79\xc8\x55\x68\x5c\x44\xae\xa0\xa5\x2e\xec\xed\x57\x72\x91\x75\x66\x48\x55\x37\xb3\xa3\xdb\xd7\x0e\xfb\xa6\xcb\xd9\xbf\xb8\x84\xf7\x1e\xf8\x3d\xbc\x3f\xfc\x5f\x1b\x7d\xcc\xc5\x6b\xe8\xf3\xa4\x37\xd2\x05\x7b\xf1\x5e\x7d\xf1\x22\x7d\x5c\xb7\xa3\xcf\x0b\x3a\xfa\xb3\x55\x72\xc7\x8b\x70\xf1\xf9\xcf\x24\xf1\x8a\x98\x2e\xe7\xc3\x65\xac\x13\xfc\x44\xe3\xbf\xb3\xa4\x1f\x27\xe6\x36\x9d\x72\x84\x1b\x00\x42\xb2\x17\xf7\x33\x17\xfd\x2e\xf9\xa2\x7b\x65\x5f\xce\xdb\x20\x1f\x7e\x19\x08\xfa\x27\x7a\xea\x36\x06\x8f\x81\xc7\x9d\xfe\x4b\x23\x3b\x7a\x4f\x9f\x7c\xa3\x8e\x5f\xd1\xc8\x12\x85\xc3\xdd\x64\x91\xd2\x01\xcd\x3b\x6e\x39\x9d\x52\xf8\xf3\xaf\x2c\xcb\x0e\x88\x26\x66\x91\x1c\x48\x83\xfe\xa3\x69\x3c\xe9\x24\x4b\xe7\x78\xfc\x38\x7b\xd2\xa1\x00\x71\x40\x70\xf3\x05\x22\xd9\xdf\x01\x00\x00\xff\xff\xb0\x1f\x38\x71\xab\x16\x00\x00")

func ComponentDescriptorV2SchemaYamlBytes() ([]byte, error) {
	return bindataRead(
		_ComponentDescriptorV2SchemaYaml,
		"../../../../component-descriptor-v2-schema.yaml",
	)
}

func ComponentDescriptorV2SchemaYaml() (*asset, error) {
	bytes, err := ComponentDescriptorV2SchemaYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "../../../../component-descriptor-v2-schema.yaml", size: 5803, mode: os.FileMode(420), modTime: time.Unix(1604417413, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"../../../../component-descriptor-v2-schema.yaml": ComponentDescriptorV2SchemaYaml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"..": &bintree{nil, map[string]*bintree{
		"..": &bintree{nil, map[string]*bintree{
			"..": &bintree{nil, map[string]*bintree{
				"..": &bintree{nil, map[string]*bintree{
					"component-descriptor-v2-schema.yaml": &bintree{ComponentDescriptorV2SchemaYaml, map[string]*bintree{}},
				}},
			}},
		}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}

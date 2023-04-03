package dynaml

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"github.com/mandelsoft/spiff/yaml"
	"io"
	"strings"
	"time"
)

type FileEntry struct {
	path string
	mode int64
	data []byte
}

func func_archive(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) < 1 || len(arguments) > 2 {
		return info.Error("archive takes one or two arguments")
	}

	mode := "tar"

	if len(arguments) == 2 {
		str, ok := arguments[1].(string)
		if !ok {
			return info.Error("second argument for archive must be a string")
		}
		mode = str
	}

	files := []*FileEntry{}

	var e *FileEntry
	var err error

	if arguments[0] != nil {
		switch data := arguments[0].(type) {
		case map[string]yaml.Node:
			for _, file := range getSortedKeys(data) {
				val := data[file]
				if val == nil || val.Value() == nil {
					continue
				}
				switch v := val.Value().(type) {
				case map[string]yaml.Node:
					e, err = getFileEntry(&file, v)
					if err != nil {
						return info.Error("%s", err)
					}
				default:
					file, _, content, ok := getData(file, WriteOpts{}, file, val.Value(), true)
					if !ok {
						return info.Error("invalid file content type %s", ExpressionType(val))
					}
					e = &FileEntry{file, 0644, content}
				}
				files = append(files, e)
			}
		case []yaml.Node:
			for _, val := range data {
				switch v := val.Value().(type) {
				case map[string]yaml.Node:
					e, err = getFileEntry(nil, v)
					if err != nil {
						return info.Error("%s", err)
					}
				default:
					return info.Error("invalid file content type %s", ExpressionType(val))
				}
				files = append(files, e)
			}
		default:
			return info.Error("first argument for hash must be a file map or list, found %s", ExpressionType(arguments[0]))
		}
	}

	var buf bytes.Buffer
	switch mode {
	case "targz":
		zipper := gzip.NewWriter(&buf)
		err = tar_archive(zipper, files)
		zipper.Close()
		if err != nil {
			return info.Error("archiving %s failed: %s", mode, err)
		}
	case "tar":
		err = tar_archive(&buf, files)
		if err != nil {
			return info.Error("archiving %s failed: %s", mode, err)
		}
	default:
		return info.Error("invalid archive type '%s'", mode)
	}

	return Base64Encode(buf.Bytes(), 60), info, true
}

func getFileEntry(file *string, info map[string]yaml.Node) (*FileEntry, error) {
	var e FileEntry
	var err error

	wopt := WriteOpts{
		Permissions: 0644,
	}

	field, ok := info["path"]
	if ok {
		if field == nil || field.Value() == nil {
			return nil, fmt.Errorf("path field must not be nil")
		}
		v, ok := field.Value().(string)
		if !ok {
			return nil, fmt.Errorf("path field must be string, found %s", ExpressionType(field))
		}
		e.path = v
	} else {
		if file == nil {
			return nil, fmt.Errorf("path field required")
		}
		e.path = removeTags(*file)
	}

	field, ok = info["mode"]
	if ok {
		if field == nil || field.Value() == nil {
			return nil, fmt.Errorf("mode field must not be nil")
		}
		wopt, err = getWriteOptions(field.Value(), wopt, false)
		if err != nil {
			return nil, err
		}
	}
	e.mode = wopt.Permissions

	field, ok = info["base64"]
	if ok {
		if field == nil || field.Value() == nil {
			return nil, fmt.Errorf("base64 field must not be nil")
		}
		v, ok := field.Value().(string)
		if !ok {
			return nil, fmt.Errorf("base64 field must be string, found %s", ExpressionType(field))
		}
		e.data, err = base64.StdEncoding.DecodeString(v)
		if err != nil {
			return nil, err
		}
	} else {
		field, ok = info["data"]
		if ok {
			if field == nil || field.Value() == nil {
				return nil, fmt.Errorf("data field must not be nil")
			}
			_, _, content, ok := getData("", wopt, e.path, field.Value(), true)
			if !ok {
				return nil, fmt.Errorf("invalid data field")
			}
			e.data = content
		}
	}

	if file != nil && hasTag(*file, "*") {
		e.mode |= 0100
	}
	return &e, nil
}

func hasTag(file, tag string) bool {
	for strings.HasPrefix(file, "*") || strings.HasPrefix(file, "#") || strings.HasPrefix(file, "-") {
		if strings.HasPrefix(file, tag) {
			return true
		}
		stop := strings.HasPrefix(file, "-")
		file = file[1:]
		if stop {
			break
		}
	}
	return false
}

func removeTags(file string) string {
	for strings.HasPrefix(file, "*") || strings.HasPrefix(file, "#") || strings.HasPrefix(file, ":") {
		stop := strings.HasPrefix(file, ":")
		file = file[1:]
		if stop {
			break
		}
	}
	return file
}

func tar_archive(w io.Writer, files []*FileEntry) error {
	tw := tar.NewWriter(w)
	defer tw.Close()
	now := time.Now()
	for _, file := range files {
		header := &tar.Header{
			Name:       file.path,
			Mode:       file.mode,
			Size:       int64(len(file.data)),
			ModTime:    now,
			ChangeTime: now,
			AccessTime: now,
		}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tw.Write(file.data); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return nil
}

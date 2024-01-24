/*
 * Copyright 2022 Mandelsoft. All rights reserved.
 *  This file is licensed under the Apache Software License, v. 2 except as noted
 *  otherwise in the LICENSE file
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package utils

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"sort"
	"time"

	"github.com/mandelsoft/vfs/pkg/vfs"
)

type FileData interface {
	FileDataDirAccess
	Data() []byte
	Files() []os.FileInfo
	SetData([]byte)
	Mode() os.FileMode
	SetMode(mode os.FileMode)
	ModTime() time.Time
	SetModTime(time.Time)
	Add(name string, f FileData) error
	Del(name string) error
}

type File struct {
	// atomic requires 64-bit alignment for struct field access
	offset       int64
	readDirCount int64
	closed       bool
	readOnly     bool
	fileData     FileData
	name         string
}

var _ vfs.File = &File{}

func newFileHandle(name string, data FileData) *File {
	return &File{name: name, fileData: data}
}

func (f *File) Open() error {
	f.fileData.Lock()
	f.offset = 0
	f.readDirCount = 0
	f.closed = false
	f.fileData.Unlock()
	return nil
}

func (f *File) Close() error {
	f.fileData.Lock()
	f.closed = true
	f.fileData.Unlock()
	return nil
}

func (f *File) Name() string {
	return f.name
}

func (f *File) Stat() (os.FileInfo, error) {
	return NewFileInfo(f.name, f.fileData), nil
}

func (f *File) Sync() error {
	return nil
}

func (f *File) ReadDir(count int) (files []fs.DirEntry, err error) {
	o, err := f.Readdir(count)

	if err != nil {
		return nil, err
	}
	r := make([]fs.DirEntry, len(o), len(o))
	for i, v := range o {
		r[i] = fs.FileInfoToDirEntry(v)
	}
	return r, nil
}

func (f *File) Readdir(count int) (files []os.FileInfo, err error) {
	if !f.fileData.IsDir() {
		return nil, &os.PathError{Op: "readdir", Path: f.name, Err: ErrNotDir}
	}
	var outLength int64

	f.fileData.Lock()
	defer f.fileData.Unlock()

	files = f.fileData.Files()
	sort.Sort(FilesSorter(files))
	if f.readDirCount >= int64(len(files)) {
		files = []os.FileInfo{}
	} else {
		files = files[f.readDirCount:]
	}
	if count > 0 {
		if len(files) < count {
			outLength = int64(len(files))
		} else {
			outLength = int64(count)
		}
		if len(files) == 0 {
			err = io.EOF
		}
	} else {
		outLength = int64(len(files))
	}
	f.readDirCount += outLength

	return files, err
}

func (f *File) Readdirnames(n int) (names []string, err error) {
	fi, err := f.Readdir(n)
	names = make([]string, len(fi))
	for i, f := range fi {
		names[i] = f.Name()
	}
	return names, err
}

func (f *File) Read(buf []byte) (int, error) {
	f.fileData.Lock()
	defer f.fileData.Unlock()
	n, err := f.read(buf, f.offset, io.ErrUnexpectedEOF)
	f.offset += int64(n)
	return n, err
}

func (f *File) read(b []byte, offset int64, err error) (int, error) {
	if f.closed == true {
		return 0, ErrFileClosed
	}
	data := f.fileData.Data()
	if len(b) > 0 && int(offset) == len(data) {
		return 0, io.EOF
	}
	if int(f.offset) > len(data) {
		return 0, err
	}
	n := len(b)
	if len(data)-int(offset) < len(b) {
		n = len(data) - int(offset)
	}
	copy(b, data[offset:offset+int64(n)])
	return n, nil
}

func (f *File) ReadAt(b []byte, off int64) (n int, err error) {
	f.fileData.Lock()
	defer f.fileData.Unlock()
	return f.read(b, off, io.EOF)
}

func (f *File) Truncate(size int64) error {
	if f.readOnly {
		return ErrReadOnly
	}
	if f.closed == true {
		return ErrFileClosed
	}
	if size < 0 {
		return ErrOutOfRange
	}
	f.fileData.Lock()
	defer f.fileData.Unlock()
	data := f.fileData.Data()
	if size > int64(len(data)) {
		diff := size - int64(len(data))
		f.fileData.SetData(append(data, bytes.Repeat([]byte{00}, int(diff))...))
	} else {
		f.fileData.SetData(data[0:size])
	}
	f.fileData.SetModTime(time.Now())
	return nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	if f.closed == true {
		return 0, ErrFileClosed
	}
	f.fileData.Lock()
	defer f.fileData.Unlock()
	data := f.fileData.Data()
	switch whence {
	case 0:
	case 1:
		offset += f.offset
	case 2:
		offset = int64(len(data)) + offset
	}
	if offset < 0 || offset >= int64(len(data)) {
		return 0, ErrOutOfRange
	}
	f.offset = offset
	return f.offset, nil
}

func (f *File) Write(buf []byte) (int, error) {
	f.fileData.Lock()
	defer f.fileData.Unlock()
	n, err := f.write(buf, f.offset)
	if err != nil {
		return 0, err
	}
	f.offset += int64(n)
	return int(n), nil
}

func (f *File) write(buf []byte, offset int64) (int, error) {
	if f.readOnly {
		return 0, ErrReadOnly
	}
	if f.closed == true {
		return 0, ErrFileClosed
	}
	data := f.fileData.Data()
	n := int64(len(buf))
	add := offset + n - int64(len(data))
	copy(data[offset:], buf)
	if add > 0 {
		f.fileData.SetData(append(data, buf[n-add:]...))
	}
	f.fileData.SetModTime(time.Now())
	return int(n), nil
}

func (f *File) WriteAt(buf []byte, off int64) (n int, err error) {
	f.fileData.Lock()
	defer f.fileData.Unlock()
	return f.write(buf, off)
}

func (f *File) WriteString(s string) (ret int, err error) {
	return f.Write([]byte(s))
}

var (
	ErrNotDir   = vfs.ErrNotDir
	ErrReadOnly = vfs.ErrReadOnly

	ErrFileClosed = errors.New("file is closed")
	ErrOutOfRange = errors.New("out of range")
	ErrTooLarge   = errors.New("too large")
	ErrNotEmpty   = vfs.ErrNotEmpty
)

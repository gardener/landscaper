/*
 * Copyright 2020 Mandelsoft. All rights reserved.
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

package vfs

import (
	"io"
	"os"
	"time"
)

const PathSeparatorChar = '/'
const PathSeparatorString = "/"

type FileSystem interface {

	// VolumeName returns leading volume name.
	// Given "C:\foo\bar" it returns "C:" on Windows.
	// Given "\\host\share\foo" it returns "\\host\share".
	// On other platforms it returns "".
	VolumeName(name string) string

	// FSTempDir (similar to os.TempDir) provides
	// the dir to use fortemporary files for this filesystem
	FSTempDir() string

	// Normalize returns a path in the normalized vfs path syntax
	Normalize(name string) string

	// Create creates a file in the filesystem, returning the file and an
	// error, if any happens.
	Create(name string) (File, error)

	// Mkdir creates a directory in the filesystem, return an error if any
	// happens.
	Mkdir(name string, perm os.FileMode) error

	// MkdirAll creates a directory path and all parents that does not exist
	// yet.
	MkdirAll(path string, perm os.FileMode) error

	// Open opens a file, returning it or an error, if any happens.
	Open(name string) (File, error)

	// OpenFile opens a file using the given flags and the given mode.
	OpenFile(name string, flags int, perm os.FileMode) (File, error)

	// Remove removes a file identified by name, returning an error, if any
	// happens.
	Remove(name string) error

	// RemoveAll removes a directory path and any children it contains. It
	// does not fail if the path does not exist (return nil).
	RemoveAll(path string) error

	// Rename renames a file.
	Rename(oldname, newname string) error

	// Stat returns a FileInfo describing the named file, or an error, if any
	// happens.
	Stat(name string) (os.FileInfo, error)

	// Lstat returns a FileInfo describing the named file, or an error, if any
	// happens.
	// If the file is a symbolic link, the returned FileInfo
	// describes the symbolic link. Lstat makes no attempt to follow the link.
	Lstat(name string) (os.FileInfo, error)

	// Create a symlink if supported
	Symlink(oldname, newname string) error

	// Read a symlink if supported
	Readlink(name string) (string, error)

	// Name returns the spec of this FileSystem
	Name() string

	// Chmod changes the mode of the named file to mode.
	Chmod(name string, mode os.FileMode) error

	// Chtimes changes the access and modification times of the named file
	Chtimes(name string, atime time.Time, mtime time.Time) error

	// Getwd return the absolute path of the working directory of the
	// file system
	Getwd() (string, error)
}

type FileSystemWithWorkingDirectory interface {
	FileSystem
	Chdir(path string) error
}

type FileSystemCleanup interface {
	FileSystem

	// Cleanup should remove all temporary resources allocated
	// for this file system
	Cleanup() error
}

type File interface {
	io.Closer
	io.Reader
	io.ReaderAt
	io.Seeker
	io.Writer
	io.WriterAt

	Name() string
	Readdir(count int) ([]os.FileInfo, error)
	Readdirnames(n int) ([]string, error)
	Stat() (os.FileInfo, error)
	Sync() error
	Truncate(size int64) error
	WriteString(s string) (ret int, err error)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type VFS interface {
	FileSystem

	Join(elems ...string) string
	Split(path string) (string, string)
	Base(path string) string
	Dir(path string) string
	Clean(path string) string
	Trim(path string) string
	IsAbs(path string) bool
	IsRoot(path string) bool
	SplitVolume(path string) (string, string)
	SplitPath(path string) (vol string, elems []string, rooted bool)

	Canonical(path string, exist bool) (string, error)
	Abs(path string) (string, error)
	EvalSymlinks(path string) (string, error)
	Walk(path string, fn WalkFunc) error

	Exists(path string) (bool, error)
	DirExists(path string) (bool, error)
	IsDir(path string) (bool, error)
	IsFile(path string) (bool, error)

	ReadDir(path string) ([]os.FileInfo, error)
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, mode os.FileMode) error
	TempFile(dir, prefix string) (File, error)
	TempDir(dir, prefix string) (string, error)
}

func Cleanup(fs FileSystem) error {
	if fs != nil {
		if c, ok := fs.(FileSystemCleanup); ok {
			return c.Cleanup()
		}
	}
	return nil
}

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
	"bytes"
	"errors"
	"io"
	"os"
	"path"
	"sort"
	"strings"
)

func filesys(fs ...FileSystem) FileSystem {
	if len(fs) == 0 {
		return nil
	}
	return fs[0]
}

// IsPathSeparator reports whether c is a directory separator character.
func IsPathSeparator(c uint8) bool {
	return PathSeparatorChar == c
}

// Join joins any number of path elements into a single path, adding
// a Separator if necessary. Join never calls Clean on the result to
// assure the result denotes the same file as the input.
// Empty entries will be ignored.
// If a FileSystem is given, the file systems volume.
// handling is applied, otherwise the path argument
// is handled as a regular plain path
func Join(fs FileSystem, elems ...string) string {
	for i := 0; i < len(elems); i++ {
		if elems[i] == "" {
			elems = append(elems[:i], elems[i+1:]...)
		}
	}
	return Trim(fs, strings.Join(elems, PathSeparatorString))
}

// Clean returns the shortest path name equivalent to path
// by purely lexical processing. It applies the following rules
// iteratively until no further processing can be done:
//
//	1. Replace multiple path separators with a single one.
//	2. Eliminate each . path name element (the current directory).
//	3. Eliminate each inner .. path name element (the parent directory)
//	   along with the non-.. element that precedes it.
//	4. Eliminate .. elements that begin a rooted path:
//	   that is, replace "/.." by "/" at the beginning of a path.
//
// The returned path ends in a slash only if it is the root "/".
//
// If the result of this process is an empty string, Clean
// returns the string ".".
// If a FileSystem is given, the file systems volume.
// handling is applied, otherwise the path argument
// is handled as a regular plain path
func Clean(fs FileSystem, p string) string {
	vol := ""
	if fs != nil {
		p = fs.Normalize(p)
		vol = fs.VolumeName(p)
	}
	return vol + path.Clean(p[len(vol):])
}

// Dir returns the path's directory dropping the final element
// after removing trailing Separators, Dir does not call Clean on the path.
// If the path is empty, Dir returns "." or "/" for a rooted path.
// If the path consists entirely of Separators, Dir2 returns a single Separator.
// The returned path does not end in a Separator unless it is the root directory.
// This function is the counterpart of Base
// Base("a/b/")="b" and Dir("a/b/") = "a".
// If a FileSystem is given, the file systems volume.
// handling is applied, otherwise the path argument
// is handled as a regular plain path
func Dir(fs FileSystem, path string) string {
	def := "."
	vol := ""
	if fs != nil {
		vol, path = SplitVolume(fs, path)
	}
	i := len(path) - 1
	for i > 0 && IsPathSeparator(path[i]) {
		i--
	}
	for i >= 0 && !IsPathSeparator(path[i]) {
		i--
	}
	for i > 0 && IsPathSeparator(path[i]) {
		def = PathSeparatorString
		i--
	}
	path = path[0 : i+1]
	if path == "" {
		path = def
	}
	return vol + path
}

// Base extracts the last path component.
// For the root path it returns the root name,
// For an empty path . is returned
// If a FileSystem is given, the file systems volume.
// handling is applied, otherwise the path argument
// is handled as a regular plain path
func Base(fs FileSystem, path string) string {
	if fs != nil {
		_, path = SplitVolume(fs, path)
	}
	i := len(path) - 1
	for i > 0 && IsPathSeparator(path[i]) {
		i--
	}
	j := i
	for j >= 0 && !IsPathSeparator(path[j]) {
		j--
	}
	path = path[j+1 : i+1]
	if path == "" {
		if j == 0 {
			return PathSeparatorString
		}
		return "."
	}
	return path
}

// Trim eliminates trailing slashes from a path name.
// An empty path is unchanged.
// If a FileSystem is given, the file systems volume
// handling is applied, otherwise the path argument
// is handled as a regular plain path
func Trim(fs FileSystem, path string) string {
	vol := ""
	if fs != nil {
		path = fs.Normalize(path)
		vol = fs.VolumeName(path)
	}
	i := len(path) - 1
	for i > len(vol) && IsPathSeparator(path[i]) {
		i--
	}
	k := i + 1
	path = path[:k]
	for i >= len(vol) {
		j := i
		for j >= len(vol) && IsPathSeparator(path[j]) {
			j--
		}
		if i != j {
			if path[i+1:k] == "." {
				if j < len(vol) && k == len(path) {
					j++ // keep starting separator instead of trailing one, because this does not exist
				}
				i = k
			}
			path = path[:j+1] + path[i:]
			i = j
			k = i + 1
		}
		i--
	}
	if k < len(path) && path[len(vol):k] == "." {
		path = path[:len(vol)] + path[k+1:]
	}

	return path
}

// IsAbs return true if the given path is an absolute one
// starting with a Separator or is quailified by a volume name.
func IsAbs(fs FileSystem, path string) bool {
	_, path = SplitVolume(fs, path)
	return strings.HasPrefix(path, PathSeparatorString)
}

// IsRoot determines whether a given path is a root path.
// This might be the separator or the separator preceded by
// a volume name.
func IsRoot(fs FileSystem, path string) bool {
	_, path = SplitVolume(fs, path)
	return path == PathSeparatorString
}

func SplitVolume(fs FileSystem, path string) (string, string) {
	path = fs.Normalize(path)
	vol := fs.VolumeName(path)
	return vol, path[len(vol):]
}

// Canonical returns the canonical absolute path of a file.
// If exist=false the denoted file must not exist, but
// then the part of the initial path referring to a not existing
// directory structure is lexically resolved (like Clean) and
// does not consider potential symbolic links that might occur
// if the file is finally created in the future.
func Canonical(fs FileSystem, path string, exist bool) (string, error) {
	if !IsAbs(fs, path) {
		wd, err := fs.Getwd()
		if err != nil {
			return "", err
		}
		path = Join(fs, wd, path)
	}
	return evalPath(fs, path, exist)
}

// EvalSymlinks resolves all symbolic links in a path
// and returns a path not containing any symbolic link
// anymore. It does not call Clean on a non-canonical path,
// so the result always denotes the same file than the original path.
// If the given path is a relative one, a
// relative one is returned as long as there is no
// absolute symbolic link. It may contain `..`, if it is above
// the current working directory
func EvalSymlinks(fs FileSystem, path string) (string, error) {
	return evalPath(fs, path, false)
}

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Symbolic links in the given path will be resolved, but not in
// the current working directory, if used to make the path absolute.
// The denoted file may not exist.
// Abs never calls Clean on the result, so the resulting path
// will denote the same file as the argument.
func Abs(fs FileSystem, path string) (string, error) {
	path, err := evalPath(fs, path, false)
	if err != nil {
		return "", err
	}
	if IsAbs(fs, path) {
		return path, nil
	}
	p, err := fs.Getwd()
	if err != nil {
		return "", err
	}
	return Join(fs, p, path), nil
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path. In contrast to filepath.Split the directory
// path does not end with a trailing Separator, so Split can
// subsequently called for the directory part, again.
func Split(fs FileSystem, path string) (dir, file string) {
	path = fs.Normalize(path)
	vol := fs.VolumeName(path)
	i := len(path) - 1
	for i >= len(vol) && !IsPathSeparator(path[i]) {
		i--
	}
	j := i
	for j > len(vol) && IsPathSeparator(path[j]) {
		j--
	}
	return path[:j+1], path[i+1:]
}

// SplitPath splits a path into a volume and an array of the path segments
func SplitPath(fs FileSystem, path string) (string, []string, bool) {
	vol, path := SplitVolume(fs, path)
	rest := path
	elems := []string{}
	for rest != "" {
		i := 0
		for i < len(rest) && IsPathSeparator(rest[i]) {
			i++
		}
		j := i
		for j < len(rest) && !IsPathSeparator(rest[j]) {
			j++
		}
		b := rest[i:j]
		rest = rest[j:]
		if b == "." || b == "" {
			continue
		}
		elems = append(elems, b)
	}
	return vol, elems, strings.HasPrefix(path, PathSeparatorString)
}

func Exists_(err error) bool {
	return err == nil || !os.IsNotExist(err)
}

// Exists checks if a file or directory exists.
func Exists(fs FileSystem, path string) (bool, error) {
	_, err := fs.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// DirExists checks if a path exists and is a directory.
func DirExists(fs FileSystem, path string) (bool, error) {
	fi, err := fs.Stat(path)
	if err == nil && fi.IsDir() {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// FileExists checks if a path exists and is a regular file.
func FileExists(fs FileSystem, path string) (bool, error) {
	fi, err := fs.Stat(path)
	if err == nil && fi.Mode()&os.ModeType == 0 {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// IsDir checks if a given path is a directory.
func IsDir(fs FileSystem, path string) (bool, error) {
	fi, err := fs.Stat(path)
	if err != nil {
		return false, err
	}
	return fi.IsDir(), nil
}

// IsFile checks if a given path is a file.
func IsFile(fs FileSystem, path string) (bool, error) {
	fi, err := fs.Stat(path)
	if err != nil {
		return false, err
	}
	return fi.Mode()&os.ModeType == 0, nil
}

func CopyFile(srcfs FileSystem, src string, dstfs FileSystem, dst string) error {

	fi, err := srcfs.Lstat(src)
	if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() {
		return errors.New("no regular file")
	}
	s, err := srcfs.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := dstfs.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fi.Mode()&os.ModePerm)
	if err != nil {
		return err
	}
	defer d.Close()

	_, err = io.Copy(d, s)
	if err != nil {
		return err
	}
	return dstfs.Chmod(dst, fi.Mode())
}

// CopyDir recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist, destination directory may exist.
// Symlinks are ignored and skipped.
func CopyDir(srcfs FileSystem, src string, dstfs FileSystem, dst string) error {
	src = Trim(srcfs, src)
	dst = Trim(dstfs, dst)

	si, err := srcfs.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return NewPathError("CopyDir", src, ErrNotDir)
	}

	di, err := dstfs.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil && !di.IsDir() {
		return NewPathError("CopyDir", dst, ErrNotDir)
	}

	err = dstfs.MkdirAll(dst, si.Mode())
	if err != nil {
		return err
	}

	entries, err := ReadDir(srcfs, src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := Join(srcfs, src, entry.Name())
		dstPath := Join(dstfs, dst, entry.Name())

		if entry.IsDir() {
			err = CopyDir(srcfs, srcPath, dstfs, dstPath)
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				var old string
				old, err = srcfs.Readlink(srcPath)
				if err == nil {
					err = dstfs.Symlink(old, dstPath)
				}
				if err == nil {
					err = os.Chmod(dst, entry.Mode())
				}
			} else {
				err = CopyFile(srcfs, srcPath, dstfs, dstPath)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func Touch(fs FileSystem, path string, perm os.FileMode) error {
	file, err := fs.OpenFile(path, os.O_CREATE, perm&os.ModePerm)
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

// ReadFile reads the file named by filename and returns the contents.
// A successful call returns err == nil, not err == EOF. Because ReadFile
// reads the whole file, it does not treat an EOF from Read as an error
// to be reported.
func ReadFile(fs FileSystem, path string) ([]byte, error) {
	file, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it with permissions perm
// (before umask); otherwise WriteFile truncates it before writing.
func WriteFile(fs FileSystem, filename string, data []byte, mode os.FileMode) error {
	f, err := fs.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

// ReadDir reads the directory named by path and returns
// a list of directory entries sorted by filename.
func ReadDir(fs FileSystem, path string) ([]os.FileInfo, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	list, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name() < list[j].Name() })
	return list, nil
}

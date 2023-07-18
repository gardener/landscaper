// Copyright 2017 by mandelsoft. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package filepath implements utility routines for manipulating filename paths
// in a way compatible with the target operating system-defined file paths.
// It is a modification of the original GO package path/filepath solving
// severe errors in handling symbolic links.
//
// The original package defines a function Clean that formally normalizes
// a file path by eliminating .. and . entries. This is done WITHOUT
// observing the actual file system. Although this is no problem for
// the function itself, because it is designed to do so, it becomes a severe
// problem for the whole package, because nearly all functions internally use
// Clean to clean the path. As a consequence even functions like Join deliver
// corrupted invalid results for valid inputs if the path incorporates
// symbolic links to directories. Especially EvalSymlinks cannot be used
// to evaluate paths to existing files, because Clean is internally used to
// normalize content of symbolic links.
//
// This package provides a set of functions that do not hamper the meaning
// of path names keeping the rest of the original specification as far as
// possible. Additionally some new functions (like Canonical) or alternate
// versions of existing functions (like Split2) are provided
// that offer a more useful specification than the original one.
package filepath

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
)

const PathSeparatorString = string(os.PathSeparator)

func debug(f string, args ...interface{}) {
	if false {
		fmt.Printf(f, args...)
	}
}

// IsAbs return true if the given path is an absolute one
// starting with a Separator or is quailified by a volume name.
func IsAbs(path string) bool {
	return strings.HasPrefix(path, PathSeparatorString) ||
		strings.HasPrefix(path, VolumeName(path)+PathSeparatorString)
}

// Canonical returns the canonical absolute path of a file.
// If exist=false the denoted file must not exist, but
// then the part of the initial path refering to a not existing
// directoy structure is lexically resolved (like Clean) and
// does not consider potential symbolic links that might occur
// if the file is finally created in the future.
func Canonical(path string, exist bool) (string, error) {
	return walk(path, -1, exist)
}

// EvalSymLinks resolves all symbolic links in a path
// and returns a path not containing any symbolic link
// anymore. It does not call Clean on a non-canonical path,
// so the result always denotes the same file than the original path.
// If the given path is a relative one, a
// reLative one is returned as long as there is no
// absolute symbolic link and the relative path does
// not goes up the current working diretory.
// If a relative path is returned, symbolic links
// up the current working directory are not resolved.
func EvalSymlinks(path string) (string, error) {
	return walk(path, 0, false)
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
func Abs(path string) (string, error) {
	path, err := walk(path, 0, false)
	if err != nil {
		return "", err
	}
	if IsAbs(path) {
		return path, nil
	}
	p, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return Join(p, path), nil
}

func walk(p string, parent int, exist bool) (string, error) {
	var rest []string = []string{}

	links := 0

	for !IsRoot(p) && p != "" {
		n, b := Split2(p)
		if b == "" {
			fmt.Printf("debug: ignoring empty base -> %s \n", n)
			p = n
			continue
		}
		fi, err := os.Lstat(p)
		debug("debug: %s // %s  %v\n", n, b, err)
		if exists_(err) {
			if err != nil && !os.IsPermission(err) {
				return "", err
			}
			debug("debug: file exists '%s'\n", p)
			if fi.Mode()&os.ModeSymlink != 0 {
				newpath, err := os.Readlink(p)
				if err != nil {
					return "", err
				}
				if IsAbs(newpath) {
					p = newpath
				} else {
					p = Join(n, newpath)
				}
				debug("LINK %s -> %s\n", newpath, p)
				links++
				if links > 255 {
					return "", errors.New("AbsPath: too many links")
				}
				continue
			}
		} else {
			if exist {
				return "", err
			}
			debug("debug: %s does not exist\n", p)
		}
		if b != "." {
			rest = append([]string{b}, rest...)
			if parent >= 0 && b == ".." {
				parent++
			} else {
				if parent > 0 {
					parent--
				}
			}
		}
		if parent != 0 && n == "" {
			p, err = os.Getwd()
			if err != nil {
				return "", err
			}
			debug("debug: continue with wd '%s'\n", p)
		} else {
			p = n
		}
	}
	if p == "" {
		return Clean(Join(rest...)), nil
	}
	return Clean(Join(append([]string{p}, rest...)...)), nil
}

// Exists checks whether a file exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return exists_(err)
}

func exists_(err error) bool {
	return err == nil || !os.IsNotExist(err)
}

// Dir2 returns the path's directory dropping the final element
// after removing trailing Separators, Dir2 goes not call Clean on the path.
// If the path is empty, Dir2 returns ".".
// If the path consists entirely of Separators, Dir2 returns a single Separator.
// The returned path does not end in a Separator unless it is the root directory.
// This function is the counterpart of Base
// Base("a/b/")="b" and Dir("a/b/") = "a".
// In general Trim(Join(Dir2(path),Base(path))) should be Trim(path)
func Dir2(path string) string {
	def := "."
	vol := VolumeName(path)
	i := len(path) - 1
	for i > len(vol) && os.IsPathSeparator(path[i]) {
		i--
	}
	for i >= len(vol) && !os.IsPathSeparator(path[i]) {
		i--
	}
	for i > len(vol) && os.IsPathSeparator(path[i]) {
		def = string(os.PathSeparator)
		i--
	}
	path = path[len(vol) : i+1]
	if path == "" {
		return def
	}
	return vol + path
}

// Dir acts like filepath.Dir, but does not
// clean the path
// Like the original Dir function this is NOT
// the counterpart of Base if the path ends with
// a trailing Separator. Base("a/b/")="b" and
// Dir("a/b/") = "a/b".
func Dir(path string) string {
	def := "."
	vol := VolumeName(path)
	i := len(path) - 1
	for i >= len(vol) && !os.IsPathSeparator(path[i]) {
		i--
	}
	for i > len(vol) && os.IsPathSeparator(path[i]) {
		def = string(os.PathSeparator)
		i--
	}

	path = path[len(vol) : i+1]
	if path == "" {
		path = def
	}
	return vol + path
}

func Base(path string) string {
	vol := VolumeName(path)
	i := len(path) - 1
	for i > len(vol) && os.IsPathSeparator(path[i]) {
		i--
	}
	j := i
	for j >= len(vol) && !os.IsPathSeparator(path[j]) {
		j--
	}
	path = path[j+1 : i+1]
	if path == "" {
		if j == len(vol) {
			return PathSeparatorString
		}
		return "."
	}
	return path
}

// Trim eleminates additional slashes and dot segments from a path name.
// An empty path is unchanged.
//
func Trim(path string) string {
	vol := VolumeName(path)
	i := len(path) - 1
	for i > len(vol) && os.IsPathSeparator(path[i]) {
		i--
	}

	path = path[:i+1]

	k := len(path)
	i = k - 1
	for i >= len(vol) {
		j := i
		for j >= len(vol) && os.IsPathSeparator(path[j]) {
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

// Split2 splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path. In contrast to Split the directory
// path does not end with a trailing Separator, so Split2 can
// subsequently called for the directory part, again.
func Split2(path string) (dir, file string) {
	vol := VolumeName(path)
	i := len(path) - 1
	for i >= len(vol) && !os.IsPathSeparator(path[i]) {
		i--
	}
	j := i
	for j > len(vol) && os.IsPathSeparator(path[j]) {
		j--
	}
	return path[:j+1], path[i+1:]
}

// Join joins any number of path elements into a single path, adding
// a Separator if necessary. Join never calls Clean on the result to
// assure the result denotes the same file as the input.
// On Windows, the result is a UNC path if and only if the first path
// element is a UNC path.
func Join(elems ...string) string {
	s := string(os.PathSeparator) + string(os.PathSeparator)
	for i := 0; i < len(elems); i++ {
		if elems[i] == "" {
			elems = append(elems[:i], elems[i+1:]...)
		}
	}
	r := strings.Join(elems, string(os.PathSeparator))
	for strings.Index(r, s) >= 0 {
		r = strings.ReplaceAll(r, s, string(os.PathSeparator))
	}
	return r
}

// IsRoot determines whether a given path is a root path.
// This might be the Separator or the Separator preceded by
// a volume name under Windows.
// This function is directly taken from the original filepath
// package.
func IsRoot(path string) bool {
	if runtime.GOOS != "windows" {
		return path == "/"
	}
	switch len(path) {
	case 1:
		return os.IsPathSeparator(path[0])
	case 3:
		return path[1] == ':' && os.IsPathSeparator(path[2])
	}
	return false
}

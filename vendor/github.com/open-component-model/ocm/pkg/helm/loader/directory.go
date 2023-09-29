// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"

	"github.com/open-component-model/ocm/pkg/contexts/oci/ociutils/helm/ignore"
	"github.com/open-component-model/ocm/pkg/contexts/oci/ociutils/helm/sympath"
)

var utf8bom = []byte{0xEF, 0xBB, 0xBF}

// LoadDir loads from a directory.
//
// This loads charts only from directories.
func LoadDir(fs vfs.FileSystem, dir string) (*chart.Chart, error) {
	topdir, err := vfs.Abs(fs, dir)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot determine absolute directory path")
	}

	// Just used for errors.
	c := &chart.Chart{}

	rules := ignore.Empty()
	ifile := vfs.Join(fs, topdir, ignore.HelmIgnore)
	if _, err := fs.Stat(ifile); err == nil {
		file, err := fs.Open(ifile)
		if err != nil {
			return c, err
		}
		defer file.Close()
		r, err := ignore.Parse(file)
		if err != nil {
			return c, err
		}
		rules = r
	}
	rules.AddDefaults()

	files := []*loader.BufferedFile{}
	topdir += vfs.PathSeparatorString

	walk := func(name string, fi os.FileInfo, err error) error {
		n := strings.TrimPrefix(name, topdir)
		if n == "" {
			// No need to process top level. Avoid bug with helmignore .* matching
			// empty names. See issue 1779.
			return nil
		}

		// Normalize to / since it will also work on Windows
		n = filepath.ToSlash(n)

		if err != nil {
			return err
		}

		if fi.IsDir() {
			// Directory-based ignore rules should involve skipping the entire
			// contents of that directory.
			if rules.Ignore(n, fi) {
				return vfs.SkipDir
			}
			return nil
		}

		// If a .helmignore file matches, skip this file.
		if rules.Ignore(n, fi) {
			return nil
		}

		// Irregular files include devices, sockets, and other uses of files that
		// are not regular files. In Go they have a file mode type bit set.
		// See https://golang.org/pkg/os/#FileMode for examples.
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("cannot load irregular file %s as it has file mode type bits set", name)
		}

		data, err := vfs.ReadFile(fs, name)
		if err != nil {
			return errors.Wrapf(err, "error reading %s", n)
		}

		data = bytes.TrimPrefix(data, utf8bom)

		files = append(files, &loader.BufferedFile{Name: n, Data: data})
		return nil
	}
	if err = sympath.Walk(fs, topdir, walk); err != nil {
		return c, err
	}

	return loader.LoadFiles(files)
}

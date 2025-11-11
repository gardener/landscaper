// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentarchive

import (
	"fmt"
	"os"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/pflag"

	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"
)

// DefaultOutputFormatUsage defines the default usage string for output format flag.
var DefaultOutputFormatUsage = fmt.Sprintf("output format of the component archive. Can be %q, %q or %q",
	ctf.ArchiveFormatFilesystem, ctf.ArchiveFormatTar, ctf.ArchiveFormatTarGzip)

var ArchiveOutputFormatUsage = fmt.Sprintf("archive format of the component archive. Can be %q or %q",
	ctf.ArchiveFormatTar, ctf.ArchiveFormatTarGzip)

// ValidateOutputFormat validates the outpu format
func ValidateOutputFormat(value ctf.ArchiveFormat, ignoreEmpty bool) error {
	if ignoreEmpty && value == "" {
		return nil
	}
	switch value {
	case ctf.ArchiveFormatFilesystem, ctf.ArchiveFormatTar, ctf.ArchiveFormatTarGzip:
	default:
		return fmt.Errorf("unsupported output format %q, use %q, %q, %q or leave it empty to be defaulted",
			value, ctf.ArchiveFormatFilesystem, ctf.ArchiveFormatTar, ctf.ArchiveFormatTarGzip)
	}
	return nil
}

type OutputFormatValue ctf.ArchiveFormat

func NewOutputFormatValue(p *ctf.ArchiveFormat, def ctf.ArchiveFormat) pflag.Value {
	*p = def
	return (*OutputFormatValue)(p)
}

func (f *OutputFormatValue) String() string {
	return string(*f)
}

func (f *OutputFormatValue) Set(s string) error {
	*f = OutputFormatValue(s)
	return nil
}

func (f *OutputFormatValue) Type() string {
	return "CAOutputFormat"
}

func OutputFormatVar(fs *pflag.FlagSet, p *ctf.ArchiveFormat, name string, value ctf.ArchiveFormat, usage string) {
	OutputFormatVarP(fs, p, name, "", value, usage)
}

func OutputFormatVarP(fs *pflag.FlagSet, p *ctf.ArchiveFormat, name, shorthand string, value ctf.ArchiveFormat, usage string) {
	if len(usage) == 0 {
		usage = DefaultOutputFormatUsage
	}
	fs.VarP(NewOutputFormatValue(p, value), name, shorthand, usage)
}

// Write writes the given component archive to the filesystem with the format.
func Write(fs vfs.FileSystem, path string, ca *ctf.ComponentArchive, format ctf.ArchiveFormat) error {
	if err := ValidateOutputFormat(format, false); err != nil {
		return err
	}

	if format == ctf.ArchiveFormatFilesystem {
		if err := ca.WriteToFilesystem(fs, path); err != nil {
			return fmt.Errorf("unable to write componant archive to %q: %s", path, err.Error())
		}
		return nil
	}

	// output format is either tar or tgz
	out, err := fs.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to open exported file %s: %s", path, err.Error())
	}
	if format == ctf.ArchiveFormatTarGzip {
		if err := ca.WriteTarGzip(out); err != nil {
			return fmt.Errorf("unable to export file to %s: %s", path, err.Error())
		}
	} else {
		if err := ca.WriteTar(out); err != nil {
			return fmt.Errorf("unable to export file to %s: %s", path, err.Error())
		}
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("unable to close file: %w", err)
	}
	return nil
}

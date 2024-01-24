package dirtree

import (
	"github.com/mandelsoft/vfs/pkg/vfs"
	"golang.org/x/exp/slices"

	"github.com/open-component-model/ocm/pkg/optionutils"
	"github.com/open-component-model/ocm/pkg/utils"
)

type Option = optionutils.Option[*Options]

type Options struct {
	// FileSystem defines the file system that contains the specified directory.
	FileSystem vfs.FileSystem
	MimeType   string
	// CompressWithGzip defines whether the specified directory should be compressed.
	CompressWithGzip *bool `json:"compress,omitempty"`
	// PreserveDir defines that the specified directory should be included in the blob.
	PreserveDir *bool `json:"preserveDir,omitempty"`
	// IncludeFiles is a list of shell file name patterns that describe the files that should be included.
	// If nothing is defined, all files are included.
	IncludeFiles []string `json:"includeFiles,omitempty"`
	// ExcludeFiles is a list of shell file name patterns that describe the files that should be excluded from the resulting tar.
	// Excluded files always overwrite included files.
	ExcludeFiles []string `json:"excludeFiles,omitempty"`
	// FollowSymlinks configures to follow and resolve symlinks when a directory is tarred.
	// This options will include the content of the symlink directly in the tar.
	// This option should be used with care.
	FollowSymlinks *bool `json:"followSymlinks,omitempty"`
}

func (o *Options) ApplyTo(opts *Options) {
	if opts == nil {
		return
	}
	if o.FileSystem != nil {
		opts.FileSystem = o.FileSystem
	}
	if o.MimeType != "" {
		opts.MimeType = o.MimeType
	}
	if o.CompressWithGzip != nil {
		opts.CompressWithGzip = utils.BoolP(*o.CompressWithGzip)
	}
	if o.PreserveDir != nil {
		opts.PreserveDir = utils.BoolP(*o.PreserveDir)
	}
	if len(o.IncludeFiles) != 0 {
		opts.IncludeFiles = slices.Clone(o.IncludeFiles)
	}
	if len(o.ExcludeFiles) != 0 {
		opts.ExcludeFiles = slices.Clone(o.ExcludeFiles)
	}
	if o.FollowSymlinks != nil {
		opts.FollowSymlinks = utils.BoolP(*o.FollowSymlinks)
	}
}

////////////////////////////////////////////////////////////////////////////////

type fileSystem struct {
	fs vfs.FileSystem
}

func (o *fileSystem) ApplyTo(opts *Options) {
	opts.FileSystem = o.fs
}

func WithFileSystem(fs vfs.FileSystem) Option {
	return &fileSystem{fs: fs}
}

////////////////////////////////////////////////////////////////////////////////

type mimeType string

func (o mimeType) ApplyTo(opts *Options) {
	opts.MimeType = string(o)
}

func WithMimeType(mime string) Option {
	return mimeType(mime)
}

type compressWithGzip bool

func (o compressWithGzip) ApplyTo(opts *Options) {
	opts.CompressWithGzip = utils.BoolP(o)
}

func WithCompressWithGzip(b ...bool) Option {
	return compressWithGzip(utils.OptionalDefaultedBool(true, b...))
}

type preserveDir bool

func (o preserveDir) ApplyTo(opts *Options) {
	opts.PreserveDir = utils.BoolP(o)
}

func WithPreserveDir(b ...bool) Option {
	return preserveDir(utils.OptionalDefaultedBool(true, b...))
}

type includeFiles []string

func (o includeFiles) ApplyTo(opts *Options) {
	opts.IncludeFiles = slices.Clone(o)
}

func WithIncludeFiles(files []string) Option {
	return includeFiles(files)
}

type excludeFiles []string

func (o excludeFiles) ApplyTo(opts *Options) {
	opts.ExcludeFiles = slices.Clone(o)
}

func WithExcludeFiles(files []string) Option {
	return excludeFiles(files)
}

type followSymlinks bool

func (o followSymlinks) ApplyTo(opts *Options) {
	opts.FollowSymlinks = utils.BoolP(o)
}

func WithFollowSymlinks(b ...bool) Option {
	return followSymlinks(utils.OptionalDefaultedBool(true, b...))
}

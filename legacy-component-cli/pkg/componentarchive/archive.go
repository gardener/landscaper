// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentarchive

import (
	"compress/gzip"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	cdvalidation "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2/validation"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/codec"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive/input"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/constants"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/utils"
)

type BuilderOptions struct {
	ComponentArchivePath string

	Name                 string
	Version              string
	BaseUrl              string
	ComponentNameMapping string

	Overwrite bool
}

func (o *BuilderOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.ComponentArchivePath, "archive", "a", "", "path to the component archive directory")
	fs.StringVar(&o.Name, "component-name", "", "name of the component")
	fs.StringVar(&o.Version, "component-version", "", "version of the component")
	fs.StringVar(&o.BaseUrl, "repo-ctx", "", "[OPTIONAL] repository context url for component to upload. The repository url will be automatically added to the repository contexts.")
	fs.StringVar(&o.ComponentNameMapping, "component-name-mapping", string(cdv2.OCIRegistryURLPathMapping), "[OPTIONAL] repository context name mapping")
}

// Default applies defaults to the builder options
func (o *BuilderOptions) Default() {
	// default component path to env var
	if len(o.ComponentArchivePath) == 0 {
		o.ComponentArchivePath = filepath.Dir(os.Getenv(constants.ComponentArchivePathEnvName))
	}
}

// Validate validates the component archive builder options.
func (o *BuilderOptions) Validate() error {
	if len(o.ComponentArchivePath) == 0 {
		return errors.New("a component archive path must be provided")
	}

	if len(o.Name) != 0 {
		if len(o.Version) == 0 {
			return errors.New("a version has to be provided for a minimal component descriptor")
		}
	}
	if len(o.ComponentNameMapping) != 0 {
		if o.ComponentNameMapping != string(cdv2.OCIRegistryURLPathMapping) &&
			o.ComponentNameMapping != string(cdv2.OCIRegistryDigestMapping) {
			return fmt.Errorf("unknown component name mapping method %q", o.ComponentNameMapping)
		}
	}
	return nil
}

// Build creates a component archives with the given configuration
func (o *BuilderOptions) Build(fs vfs.FileSystem) (*ctf.ComponentArchive, error) {
	o.Default()
	if err := o.Validate(); err != nil {
		return nil, err
	}

	compDescFilePath := filepath.Join(o.ComponentArchivePath, ctf.ComponentDescriptorFileName)
	if !o.Overwrite {
		_, err := fs.Stat(compDescFilePath)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		if err == nil {
			// add the input to the ctf format
			archiveFs, err := projectionfs.New(fs, o.ComponentArchivePath)
			if err != nil {
				return nil, fmt.Errorf("unable to create projectionfilesystem: %w", err)
			}

			archive, err := ctf.NewComponentArchiveFromFilesystem(archiveFs, codec.DisableValidation(true))
			if err != nil {
				return nil, fmt.Errorf("unable to parse component archive from %s: %w", o.ComponentArchivePath, err)
			}

			cd := archive.ComponentDescriptor

			if o.Name != "" {
				if cd.Name != "" && cd.Name != o.Name {
					return nil, errors.New("unable to overwrite the existing component name: forbidden")
				}
				cd.Name = o.Name
			}

			if o.Version != "" {
				if cd.Version != "" && cd.Version != o.Version {
					return nil, errors.New("unable to overwrite the existing component version: forbidden")
				}
				cd.Version = o.Version
			}

			if err = cdvalidation.Validate(cd); err != nil {
				return nil, fmt.Errorf("invalid component descriptor: %w", err)
			}

			return archive, nil
		}
	}

	// build minimal archive

	if err := fs.MkdirAll(o.ComponentArchivePath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("unable to create component-archive path %q: %w", o.ComponentArchivePath, err)
	}
	archiveFs, err := projectionfs.New(fs, o.ComponentArchivePath)
	if err != nil {
		return nil, fmt.Errorf("unable to create projectionfilesystem: %w", err)
	}

	cd := &cdv2.ComponentDescriptor{}
	cd.Metadata.Version = cdv2.SchemaVersion
	cd.ComponentSpec.Name = o.Name
	cd.ComponentSpec.Version = o.Version
	cd.Provider = "internal"
	cd.RepositoryContexts = make([]*cdv2.UnstructuredTypedObject, 0)
	if len(o.BaseUrl) != 0 {
		repoCtx, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository(o.BaseUrl, cdv2.ComponentNameMapping(o.ComponentNameMapping)))
		if err != nil {
			return nil, fmt.Errorf("unable to create repository context: %w", err)
		}
		cd.RepositoryContexts = []*cdv2.UnstructuredTypedObject{&repoCtx}
	}
	if err := cdv2.DefaultComponent(cd); err != nil {
		utils.PrintPrettyYaml(cd, true)
		return nil, fmt.Errorf("unable to default component descriptor: %w", err)
	}

	if err := cdvalidation.Validate(cd); err != nil {
		return nil, fmt.Errorf("unable to validate component descriptor: %w", err)
	}

	data, err := yaml.Marshal(cd)
	if err != nil {
		utils.PrintPrettyYaml(cd, true)
		return nil, fmt.Errorf("unable to marshal component descriptor: %w", err)
	}
	if err := vfs.WriteFile(fs, compDescFilePath, data, os.ModePerm); err != nil {
		utils.PrintPrettyYaml(cd, true)
		return nil, fmt.Errorf("unable to write component descriptor to %s: %w", compDescFilePath, err)
	}

	return ctf.NewComponentArchive(cd, archiveFs), nil
}

// Parse parses a component archive from a given path.
// It automatically detects the archive format.
// Supported formats are fs, tar or tgz
func Parse(fs vfs.FileSystem, path string) (*ctf.ComponentArchive, ctf.ArchiveFormat, error) {
	info, err := fs.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", fmt.Errorf("component archive at %q does not exist", path)
		}
		return nil, "", fmt.Errorf("unable to read %q: %w", path, err)
	}

	// if the path points to a directory we expect that the ca is in a fs format
	if info.IsDir() {
		archiveFs, err := projectionfs.New(fs, path)
		if err != nil {
			return nil, "", fmt.Errorf("unable to create filesystem from %s: %s", path, err.Error())
		}
		ca, err := ctf.NewComponentArchiveFromFilesystem(archiveFs)
		return ca, ctf.ArchiveFormatFilesystem, err
	}

	// the path points to a file
	mimetype, err := utils.GetFileType(fs, path)
	if err != nil {
		return nil, "", fmt.Errorf("unable to get mimetype of %q: %s", path, err.Error())
	}
	file, err := fs.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("unable to read component archive rom %q: %s", path, err.Error())
	}

	switch mimetype {
	case "application/x-gzip", input.MediaTypeGZip, "application/tar+gzip":
		zr, err := gzip.NewReader(file)
		if err != nil {
			return nil, "", fmt.Errorf("unable to open gzip reader: %w", err)
		}
		ca, err := ctf.NewComponentArchiveFromTarReader(zr)
		if err != nil {
			return nil, "", fmt.Errorf("unable to unzip componentarchive: %s", err.Error())
		}
		if err := zr.Close(); err != nil {
			return nil, "", fmt.Errorf("unable to close gzip reader: %w", err)
		}
		if err := file.Close(); err != nil {
			return nil, "", fmt.Errorf("unable to close file reader: %w", err)
		}
		return ca, ctf.ArchiveFormatTar, nil
	case "application/octet-stream": // expect that is has to be a tar
		ca, err := ctf.NewComponentArchiveFromTarReader(file)
		if err != nil {
			return nil, "", fmt.Errorf("unable to unzip componentarchive: %s", err.Error())
		}
		if err := file.Close(); err != nil {
			return nil, "", fmt.Errorf("unable to close file reader: %w", err)
		}
		return ca, ctf.ArchiveFormatTarGzip, nil
	default:
		return nil, "", fmt.Errorf("unsupported file type %q. Expected a tar or a tar.gz", mimetype)
	}
}

// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/validation/field"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2/cdutils"
	cdvalidation "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2/validation"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive/input"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/componentarchive"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/template"
)

// Options defines the options that are used to add resources to a component descriptor
type Options struct {
	componentarchive.BuilderOptions
	TemplateOptions template.Options

	// SourceObjectPaths defines the path to the source defined as yaml or json.
	// either components can be added by a yaml resource template or by input flags
	SourceObjectPaths []string

	// SourceObjectPath defines the path to the resources defined as yaml or json
	// DEPRECATED
	SourceObjectPath string
}

// SourceOptions contains options that are used to describe a source
type SourceOptions struct {
	cdv2.Source
	Input *input.BlobInput `json:"input,omitempty"`
}

// InternalSourceOptions contains the source options as well as the
// context path where to look for input data.
type InternalSourceOptions struct {
	SourceOptions
	Path string
}

// NewAddCommand creates a command to add additional resources to a component descriptor.
func NewAddCommand(ctx context.Context) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "add COMPONENT_ARCHIVE_PATH [source file]...",
		Args:  cobra.MinimumNArgs(1),
		Short: "Adds a source to a component descriptor",
		Long: fmt.Sprintf(`
add adds sources to the defined component descriptor.
The sources can be defined in a file or given through stdin.

The source definitions are expected to be a multidoc yaml of the following form

<pre>

---
name: 'myrepo'
type: 'git'
access:
  type: "git"
  repository: github.com/gardener/landscaper/legacy-component-cli
...
---
name: 'myconfig'
type: 'json'
input:
  type: "file"
  path: "some/path"
...
---
name: 'myothersrc'
type: 'git'
input:
  type: "dir"
  path: /my/path
  compress: true # defaults to false
  exclude: "*.txt"
  preserveDir: true # optional, defaulted to false; if true, the top level folder "my/path" is included
...

</pre>

%s
`, opts.TemplateOptions.Usage()),
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(args); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			if err := opts.Run(ctx, logger.Log, osfs.New()); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

func (o *Options) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	compDescFilePath := filepath.Join(o.ComponentArchivePath, ctf.ComponentDescriptorFileName)

	archive, err := o.BuilderOptions.Build(fs)
	if err != nil {
		return err
	}

	sources, err := o.generateSources(log, fs)
	if err != nil {
		return err
	}

	for _, src := range sources {
		if src.Input != nil {
			log.Info(fmt.Sprintf("add input blob from %q", src.Input.Path))
			if err := o.addInputBlob(ctx, fs, archive, src); err != nil {
				return err
			}
		} else {
			id := archive.ComponentDescriptor.GetSourceIndex(src.Source)
			if id != -1 {
				mergedSrc := cdutils.MergeSources(archive.ComponentDescriptor.Sources[id], src.Source)
				if errList := cdvalidation.ValidateSource(field.NewPath(""), mergedSrc); len(errList) != 0 {
					return fmt.Errorf("invalid component reference: %w", errList.ToAggregate())
				}
				archive.ComponentDescriptor.Sources[id] = mergedSrc
			} else {
				if errList := cdvalidation.ValidateSource(field.NewPath(""), src.Source); len(errList) != 0 {
					return fmt.Errorf("invalid component reference: %w", errList.ToAggregate())
				}
				archive.ComponentDescriptor.Sources = append(archive.ComponentDescriptor.Sources, src.Source)
			}
		}
		log.V(3).Info(fmt.Sprintf("Successfully added source %q to component descriptor", src.Name))
	}

	if err := cdvalidation.Validate(archive.ComponentDescriptor); err != nil {
		return fmt.Errorf("invalid component descriptor: %w", err)
	}

	data, err := yaml.Marshal(archive.ComponentDescriptor)
	if err != nil {
		return fmt.Errorf("unable to encode component descriptor: %w", err)
	}
	if err := vfs.WriteFile(fs, compDescFilePath, data, 0664); err != nil {
		return fmt.Errorf("unable to write modified comonent descriptor: %w", err)
	}
	log.V(1).Info("Successfully added all sources to component descriptor")
	return nil
}

func (o *Options) Complete(args []string) error {
	args = o.TemplateOptions.Parse(args)

	if len(args) == 0 {
		return errors.New("at least a component archive path argument has to be defined")
	}

	o.BuilderOptions.ComponentArchivePath = args[0]
	o.BuilderOptions.Default()

	o.SourceObjectPaths = append(o.SourceObjectPaths, args[1:]...)
	if len(o.SourceObjectPath) != 0 {
		o.SourceObjectPaths = append(o.SourceObjectPaths, o.SourceObjectPath)
	}

	return o.validate()
}

func (o *Options) validate() error {
	return o.BuilderOptions.Validate()
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.BuilderOptions.AddFlags(fs)
	// specify the resource
	fs.StringVarP(&o.SourceObjectPath, "resource", "r", "", "The path to the resources defined as yaml or json")
	_ = fs.MarkDeprecated("resource", "the resources flag is deprecated use the arguments instead.")
}

// generateSources parses component references from the given path and stdin.
func (o *Options) generateSources(log logr.Logger, fs vfs.FileSystem) ([]InternalSourceOptions, error) {
	if len(o.SourceObjectPaths) == 0 {
		// try to read from stdin if no resources are defined
		sourceOptions := make([]InternalSourceOptions, 0)
		stdinInfo, err := os.Stdin.Stat()
		if err != nil {
			log.V(3).Info("unable to read from stdin", "error", err.Error())
			return nil, nil
		}
		if (stdinInfo.Mode()&os.ModeNamedPipe != 0) || stdinInfo.Size() != 0 {
			stdinResources, err := o.generateSourcesFromReader(os.Stdin)
			if err != nil {
				return nil, fmt.Errorf("unable to read from stdin: %w", err)
			}
			sourceOptions = append(sourceOptions, convertToInternalSourceOptions(stdinResources, "")...)
		}
		return sourceOptions, nil
	}

	sourceOptions := make([]InternalSourceOptions, 0)
	for _, resourcePath := range o.SourceObjectPaths {
		if resourcePath == "-" {
			stdinInfo, err := os.Stdin.Stat()
			if err != nil {
				return nil, fmt.Errorf("unable to read from stdin: %w", err)
			}
			if (stdinInfo.Mode()&os.ModeNamedPipe != 0) || stdinInfo.Size() != 0 {
				stdinResources, err := o.generateSourcesFromReader(os.Stdin)
				if err != nil {
					return nil, fmt.Errorf("unable to read from stdin: %w", err)
				}
				sourceOptions = append(sourceOptions, convertToInternalSourceOptions(stdinResources, "")...)
			}
			continue
		}

		resourceObjectReader, err := fs.Open(resourcePath)
		if err != nil {
			return nil, fmt.Errorf("unable to read source object from %s: %w", resourcePath, err)
		}
		newResources, err := o.generateSourcesFromReader(resourceObjectReader)
		if err != nil {
			if err2 := resourceObjectReader.Close(); err2 != nil {
				log.Error(err, "unable to close file reader", "path", resourcePath)
			}
			return nil, fmt.Errorf("unable to read sources from %s: %w", resourcePath, err)
		}
		if err := resourceObjectReader.Close(); err != nil {
			return nil, fmt.Errorf("unable to read source from %q: %w", resourcePath, err)
		}
		sourceOptions = append(sourceOptions, convertToInternalSourceOptions(newResources, resourcePath)...)
	}
	return sourceOptions, nil
}

func (o *Options) generateSourcesFromReader(reader io.Reader) ([]SourceOptions, error) {
	var data bytes.Buffer
	if _, err := io.Copy(&data, reader); err != nil {
		return nil, fmt.Errorf("unable to read sources: %w", err)
	}
	tmplData, err := o.TemplateOptions.Template(data.String())
	if err != nil {
		return nil, fmt.Errorf("unable to template source definition: %w", err)
	}
	return generateSourcesFromReader(bytes.NewBufferString(tmplData))
}

// generateSourcesFromReader generates a resource given resource options and a resource template file.
func generateSourcesFromReader(reader io.Reader) ([]SourceOptions, error) {
	sources := make([]SourceOptions, 0)
	yamldecoder := yamlutil.NewYAMLOrJSONDecoder(reader, 1024)
	for {
		src := SourceOptions{}
		if err := yamldecoder.Decode(&src); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("unable to decode src: %w", err)
		}
		sources = append(sources, src)
	}

	return sources, nil
}

func (o *Options) addInputBlob(ctx context.Context, fs vfs.FileSystem, archive *ctf.ComponentArchive, src InternalSourceOptions) error {
	blob, err := src.Input.Read(ctx, fs, src.Path)
	if err != nil {
		return err
	}

	err = archive.AddSource(&src.Source, ctf.BlobInfo{
		MediaType: src.Type,
		Digest:    blob.Digest,
		Size:      blob.Size,
	}, blob.Reader)
	if err != nil {
		blob.Reader.Close()
		return fmt.Errorf("unable to add input blob to archive: %w", err)
	}
	if err := blob.Reader.Close(); err != nil {
		return fmt.Errorf("unable to close input file: %w", err)
	}
	return nil
}

func convertToInternalSourceOptions(srcOpts []SourceOptions, filepath string) []InternalSourceOptions {
	if len(srcOpts) == 0 {
		return nil
	}
	resources := make([]InternalSourceOptions, len(srcOpts))
	for i, srcOpt := range srcOpts {
		opt := srcOpt
		resources[i] = InternalSourceOptions{
			SourceOptions: opt,
			Path:          filepath,
		}
	}
	return resources
}

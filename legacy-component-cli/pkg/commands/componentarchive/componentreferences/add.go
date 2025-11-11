// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentreferences

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
	cdvalidation "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2/validation"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/componentarchive"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/template"
)

// Options defines the options that are used to add resources to a component descriptor
type Options struct {
	componentarchive.BuilderOptions
	TemplateOptions template.Options

	// ComponentReferenceObjectPaths describe the paths to the component reference resources defined as yaml or json.
	// either components can be added by a yaml resource template or by input flags
	ComponentReferenceObjectPaths []string

	// ComponentReferenceObjectPath defines the path to the resources defined as yaml or json
	// DEPRECATED
	ComponentReferenceObjectPath string
}

// NewAddCommand creates a command to add additional resources to a component descriptor.
func NewAddCommand(ctx context.Context) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "add COMPONENT_ARCHIVE_PATH [COMPONENT_REFERENCE_PATH...]",
		Args:  cobra.MinimumNArgs(1),
		Short: "Adds a component reference to a component descriptor",
		Long: fmt.Sprintf(`
adds component references to the defined component descriptor.
The component references can be defined in a file or given through stdin.

The component references are expected to be a multidoc yaml of the following form

<pre>

---
name: 'ubuntu'
componentName: 'github.com/gardener/ubuntu'
version: 'v0.0.1'
...
---
name: 'myref'
componentName: 'github.com/gardener/other'
version: 'v0.0.2'
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

	refs, err := o.generateComponentReferences(log, fs)
	if err != nil {
		return err
	}

	for _, ref := range refs {
		if errList := cdvalidation.ValidateComponentReference(field.NewPath(""), ref); len(errList) != 0 {
			return fmt.Errorf("invalid component reference: %w", errList.ToAggregate())
		}
		id := archive.ComponentDescriptor.GetComponentReferenceIndex(ref)
		if id != -1 {
			archive.ComponentDescriptor.ComponentReferences[id] = ref
		} else {
			archive.ComponentDescriptor.ComponentReferences = append(archive.ComponentDescriptor.ComponentReferences, ref)
		}
		log.V(3).Info(fmt.Sprintf("Successfully added component references %q of component %q to component descriptor", ref.Name, ref.ComponentName))
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
	log.V(1).Info("Successfully added all component references to component descriptor")
	return nil
}

func (o *Options) Complete(args []string) error {
	args = o.TemplateOptions.Parse(args)
	if len(args) == 0 {
		return errors.New("at least a component archive path argument has to be defined")
	}
	o.BuilderOptions.ComponentArchivePath = args[0]
	o.BuilderOptions.Default()

	if len(args) > 1 {
		o.ComponentReferenceObjectPaths = append(o.ComponentReferenceObjectPaths, args[1:]...)
	}
	if len(o.ComponentReferenceObjectPath) != 0 {
		o.ComponentReferenceObjectPaths = append(o.ComponentReferenceObjectPaths, o.ComponentReferenceObjectPath)
	}
	return o.validate()
}

func (o *Options) validate() error {
	return o.BuilderOptions.Validate()
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.BuilderOptions.AddFlags(fs)
	// specify the resource
	fs.StringVarP(&o.ComponentReferenceObjectPath, "resource", "r", "", "The path to the resources defined as yaml or json")
}

// generateComponentReferences parses component references from the given path and stdin.
func (o *Options) generateComponentReferences(log logr.Logger, fs vfs.FileSystem) ([]cdv2.ComponentReference, error) {
	if len(o.ComponentReferenceObjectPaths) == 0 {
		// try to read from stdin if no resources are defined
		componentReferences := make([]cdv2.ComponentReference, 0)
		stdinInfo, err := os.Stdin.Stat()
		if err != nil {
			log.V(3).Info("unable to read from stdin", "error", err.Error())
			return nil, nil
		}
		if (stdinInfo.Mode()&os.ModeNamedPipe != 0) || stdinInfo.Size() != 0 {
			stdinResources, err := o.generateComponentReferenceFromReader(os.Stdin)
			if err != nil {
				return nil, fmt.Errorf("unable to read from stdin: %w", err)
			}
			componentReferences = append(componentReferences, stdinResources...)
		}
		return componentReferences, nil
	}

	componentReferences := make([]cdv2.ComponentReference, 0)
	for _, resourcePath := range o.ComponentReferenceObjectPaths {
		if resourcePath == "-" {
			stdinInfo, err := os.Stdin.Stat()
			if err != nil {
				return nil, fmt.Errorf("unable to read from stdin: %w", err)
			}
			if (stdinInfo.Mode()&os.ModeNamedPipe != 0) || stdinInfo.Size() != 0 {
				stdinResources, err := o.generateComponentReferenceFromReader(os.Stdin)
				if err != nil {
					return nil, fmt.Errorf("unable to read from stdin: %w", err)
				}
				componentReferences = append(componentReferences, stdinResources...)
			}
			continue
		}

		resourceObjectReader, err := fs.Open(resourcePath)
		if err != nil {
			return nil, fmt.Errorf("unable to read component reference from %s: %w", resourcePath, err)
		}
		newResources, err := o.generateComponentReferenceFromReader(resourceObjectReader)
		if err != nil {
			if err2 := resourceObjectReader.Close(); err2 != nil {
				log.Error(err, "unable to close file reader", "path", resourcePath)
			}
			return nil, fmt.Errorf("unable to read component reference from %s: %w", resourcePath, err)
		}
		if err := resourceObjectReader.Close(); err != nil {
			return nil, fmt.Errorf("unable to read component reference from %q: %w", resourcePath, err)
		}
		componentReferences = append(componentReferences, newResources...)
	}
	return componentReferences, nil
}

func (o *Options) generateComponentReferenceFromReader(reader io.Reader) ([]cdv2.ComponentReference, error) {
	var data bytes.Buffer
	if _, err := io.Copy(&data, reader); err != nil {
		return nil, err
	}
	tmplData, err := o.TemplateOptions.Template(data.String())
	if err != nil {
		return nil, err
	}
	return generateComponentReferenceFromReader(bytes.NewBufferString(tmplData))
}

// generateComponentReferenceFromReader generates a resource given resource options and a resource template file.
func generateComponentReferenceFromReader(reader io.Reader) ([]cdv2.ComponentReference, error) {
	refs := make([]cdv2.ComponentReference, 0)
	yamldecoder := yamlutil.NewYAMLOrJSONDecoder(reader, 1024)
	for {
		ref := cdv2.ComponentReference{}
		if err := yamldecoder.Decode(&ref); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("unable to decode ref: %w", err)
		}
		refs = append(refs, ref)
	}

	return refs, nil
}

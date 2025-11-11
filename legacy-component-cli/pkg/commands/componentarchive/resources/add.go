// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/validation/field"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2/cdutils"
	cdvalidation "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2/validation"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive/input"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/componentarchive"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/template"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/utils"
)

// Options defines the options that are used to add resources to a component descriptor
type Options struct {
	componentarchive.BuilderOptions
	TemplateOptions template.Options

	// either components can be added by a yaml resource template or by input flags
	// ResourceObjectPaths defines the path to the resources defined as yaml or json
	// DEPRECATED
	ResourceObjectPath string
	// ResourceObjectPaths contains paths to read the yaml resource template from.
	// If "-" is provided, the resource is read from stdin
	ResourceObjectPaths []string
}

// ResourceOptions contains options that are used to describe a resource
type ResourceOptions struct {
	cdv2.Resource
	Input *input.BlobInput `json:"input,omitempty"`
}

// ResourceOptionList contains a list of options that are used to describe a resource.
type ResourceOptionList struct {
	Resources []ResourceOptions `json:"resources"`
}

// InternalResourceOptions contains options that are used to describe a resource
// as well as the filepath of the resource that is used to search for the input blob
type InternalResourceOptions struct {
	ResourceOptions
	Path string
}

// NewAddCommand creates a command to add additional resources to a component descriptor.
func NewAddCommand(ctx context.Context) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "add COMPONENT_ARCHIVE_PATH [RESOURCE_PATH...]",
		Args:  cobra.MinimumNArgs(1),
		Short: "Adds a resource to an component archive",
		Long: fmt.Sprintf(`
add generates resources from a resource template and adds it to the given component descriptor in the component archive.
If the resource is already defined (quality by identity) in the component-descriptor it will be overwritten.

The component archive can be specified by the first argument, the flag "--archive" or as env var "COMPONENT_ARCHIVE_PATH".
The component archive is expected to be a filesystem archive. If the archive is given as tar please use the export command.

The resource template can be defined by specifying a file with the template with "resource" or it can be given through stdin.

The resource template is a multidoc yaml file so multiple templates can be defined.

<pre>

---
name: 'myimage'
type: 'ociImage'
relation: 'external'
version: 0.2.0
access:
  type: ociRegistry
  imageReference: eu.gcr.io/gardener-project/component-cli:0.2.0
...
---
name: 'myconfig'
type: 'json'
relation: 'local'
input:
  type: "file"
  path: "some/path"
  mediaType: "application/octet-stream" # optional, defaulted to "application/octet-stream" or "application/gzip" if compress=true 
...
---
name: 'myconfig'
type: 'json'
relation: 'local'
input:
  type: "dir"
  path: /my/path
  compress: true # defaults to false
  includeFiles: # optional; list of shell file patterns
  - "*.txt"
  excludeFiles: # optional; list of shell file patterns
  - "*.txt"
  mediaType: "application/gzip" # optional, defaulted to "application/x-tar" or "application/gzip" if compress=true 
  preserveDir: true # optional, defaulted to false; if true, the top level folder "my/path" is included
  followSymlinks: true # optional, defaulted to false; if true, symlinks are resolved and the content is included in the tar
...

</pre>

Alternativly the resources can also be defined as list of resources (both methods can also be combined).

<pre>

---
resources:
- name: 'myimage'
  type: 'ociImage'
  relation: 'external'
  version: 0.2.0
  access:
    type: ociRegistry
    imageReference: eu.gcr.io/gardener-project/component-cli:0.2.0

- name: 'myconfig'
  type: 'json'
  relation: 'local'
  input:
    type: "file"
    path: "some/path"
    mediaType: "application/octet-stream" # optional, defaulted to "application/octet-stream" or "application/gzip" if compress=true

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

	resources, err := o.generateResources(log, fs, archive.ComponentDescriptor)
	if err != nil {
		return err
	}

	log.V(3).Info(fmt.Sprintf("Adding %d resources...", len(resources)))
	for _, resource := range resources {
		log := log.WithValues("resource-name", resource.Name, "resource-version", resource.Version)
		utils.PrintPrettyYaml(resource, log.V(5).Enabled())

		if resource.Input != nil {
			log.Info(fmt.Sprintf("add input blob from %q", resource.Input.Path))
			if err := o.addInputBlob(ctx, fs, archive, &resource); err != nil {
				return err
			}
		} else {
			id := archive.ComponentDescriptor.GetResourceIndex(resource.Resource)
			if id != -1 {
				log.V(5).Info("Found existing resource in component descriptor, attempt merge...")
				mergedRes := cdutils.MergeResources(archive.ComponentDescriptor.Resources[id], resource.Resource)
				if errList := cdvalidation.ValidateResource(field.NewPath(""), mergedRes); len(errList) != 0 {
					return errList.ToAggregate()
				}
				archive.ComponentDescriptor.Resources[id] = mergedRes
			} else {
				if errList := cdvalidation.ValidateResource(field.NewPath(""), resource.Resource); len(errList) != 0 {
					return errList.ToAggregate()
				}
				archive.ComponentDescriptor.Resources = append(archive.ComponentDescriptor.Resources, resource.Resource)
			}
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
		log.V(2).Info("Successfully added resource to component descriptor")
	}
	log.V(2).Info("Successfully added all resources to component descriptor")
	return nil
}

func (o *Options) Complete(args []string) error {
	args = o.TemplateOptions.Parse(args)

	if len(args) == 0 {
		return errors.New("at least a component archive path argument has to be defined")
	}
	o.BuilderOptions.ComponentArchivePath = args[0]
	o.BuilderOptions.Default()

	o.ResourceObjectPaths = append(o.ResourceObjectPaths, args[1:]...)
	if len(o.ResourceObjectPath) != 0 {
		o.ResourceObjectPaths = append(o.ResourceObjectPaths, o.ResourceObjectPath)
	}

	return o.validate()
}

func (o *Options) validate() error {
	return o.BuilderOptions.Validate()
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.BuilderOptions.AddFlags(fs)
	// specify the resource
	fs.StringVarP(&o.ResourceObjectPath, "resource", "r", "", "The path to the resources defined as yaml or json")
	_ = fs.MarkDeprecated("resource", "the flag r is deprecated use command args instead")
}

func (o *Options) generateResources(log logr.Logger, fs vfs.FileSystem, cd *cdv2.ComponentDescriptor) ([]InternalResourceOptions, error) {
	if len(o.ResourceObjectPaths) == 0 {
		// try to read from stdin if no resources are defined
		resources := make([]InternalResourceOptions, 0)
		stdinInfo, err := os.Stdin.Stat()
		if err != nil {
			log.V(3).Info("unable to read from stdin", "error", err.Error())
			return nil, nil
		}
		if (stdinInfo.Mode()&os.ModeNamedPipe != 0) || stdinInfo.Size() != 0 {
			stdinResources, err := o.generateResourcesFromReader(log, cd, os.Stdin)
			if err != nil {
				return nil, fmt.Errorf("unable to read from stdin: %w", err)
			}
			resources = append(resources, convertToInternalResourceOptions(stdinResources, "")...)
		}
		return resources, nil
	}

	resources := make([]InternalResourceOptions, 0)
	for _, resourcePath := range o.ResourceObjectPaths {
		if resourcePath == "-" {
			stdinInfo, err := os.Stdin.Stat()
			if err != nil {
				return nil, fmt.Errorf("unable to read from stdin: %w", err)
			}
			if (stdinInfo.Mode()&os.ModeNamedPipe != 0) || stdinInfo.Size() != 0 {
				stdinResources, err := o.generateResourcesFromReader(log, cd, os.Stdin)
				if err != nil {
					return nil, fmt.Errorf("unable to read from stdin: %w", err)
				}
				resources = append(resources, convertToInternalResourceOptions(stdinResources, "")...)
			}
			continue
		}

		resourceObjectReader, err := fs.Open(resourcePath)
		if err != nil {
			return nil, fmt.Errorf("unable to read resource object from %s: %w", resourcePath, err)
		}
		newResources, err := o.generateResourcesFromReader(log, cd, resourceObjectReader)
		if err != nil {
			if err2 := resourceObjectReader.Close(); err2 != nil {
				log.Error(err, "unable to close file reader", "path", resourcePath)
			}
			return nil, fmt.Errorf("unable to read resources from %s: %w", resourcePath, err)
		}
		if err := resourceObjectReader.Close(); err != nil {
			return nil, fmt.Errorf("unable to read resource from %q: %w", resourcePath, err)
		}
		resources = append(resources, convertToInternalResourceOptions(newResources, resourcePath)...)
	}

	return resources, nil
}

// generateResourcesFromPath generates a resource given resource options and a resource template file.
func (o *Options) generateResourcesFromReader(log logr.Logger, cd *cdv2.ComponentDescriptor, reader io.Reader) ([]ResourceOptions, error) {
	var data bytes.Buffer
	if _, err := io.Copy(&data, reader); err != nil {
		return nil, err
	}
	// template data
	tmplData, err := o.TemplateOptions.Template(data.String())
	if err != nil {
		return nil, fmt.Errorf("unable to template resource: %w", err)
	}
	log.V(5).Info(tmplData)
	return generateResourcesFromReader(cd, bytes.NewBuffer([]byte(tmplData)))
}

// generateResourcesFromPath generates a resource given resource options and a resource template file.
func generateResourcesFromReader(cd *cdv2.ComponentDescriptor, reader io.Reader) ([]ResourceOptions, error) {
	resources := make([]ResourceOptions, 0)
	yamldecoder := yamlutil.NewYAMLOrJSONDecoder(reader, 1024)
	for {
		// ResourceOption contains either a list of options that are used to describe a resource or a resource.
		type ResourceOption struct {
			*ResourceOptionList
			*ResourceOptions
		}
		opts := ResourceOption{}
		if err := yamldecoder.Decode(&opts); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("unable to decode resource: %w", err)
		}
		if opts.ResourceOptions != nil {
			resource := *opts.ResourceOptions
			// automatically set the version to the component descriptors version for local resources
			if resource.Relation == cdv2.LocalRelation && len(resource.Version) == 0 {
				resource.Version = cd.GetVersion()
			}

			if resource.Input != nil && resource.Access != nil {
				return nil, fmt.Errorf("the resources %q input and access is defind. Only one option is allowed", resource.Name)
			}
			resources = append(resources, resource)
		} else if opts.Resources != nil {
			resourcesList := opts.ResourceOptionList
			for _, res := range resourcesList.Resources {
				resource := res
				// automatically set the version to the component descriptors version for local resources
				if resource.Relation == cdv2.LocalRelation && len(resource.Version) == 0 {
					resource.Version = cd.GetVersion()
				}

				if resource.Input != nil && resource.Access != nil {
					return nil, fmt.Errorf("the resources %q input and access is defind. Only one option is allowed", resource.Name)
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources, nil
}

func (o *Options) addInputBlob(ctx context.Context, fs vfs.FileSystem, archive *ctf.ComponentArchive, resource *InternalResourceOptions) error {
	blob, err := resource.Input.Read(ctx, fs, resource.Path)
	if err != nil {
		return err
	}
	// default media type to binary data if nothing else is defined
	resource.Input.SetMediaTypeIfNotDefined(input.MediaTypeOctetStream)

	err = archive.AddResource(&resource.Resource, ctf.BlobInfo{
		MediaType: resource.Input.MediaType,
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

func convertToInternalResourceOptions(resOpts []ResourceOptions, filepath string) []InternalResourceOptions {
	if len(resOpts) == 0 {
		return nil
	}
	resources := make([]InternalResourceOptions, len(resOpts))
	for i, resOpt := range resOpts {
		opt := resOpt
		resources[i] = InternalResourceOptions{
			ResourceOptions: opt,
			Path:            filepath,
		}
	}
	return resources
}

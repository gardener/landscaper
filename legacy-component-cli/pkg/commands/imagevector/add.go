// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imagevector

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	iv "github.com/gardener/landscaper/legacy-image-vector/pkg"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	cdvalidation "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2/validation"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/codec"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/landscaper/legacy-component-spec/bindings-go/oci"

	ociopts "github.com/gardener/landscaper/legacy-component-cli/ociclient/options"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/components"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/constants"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/utils"
)

// AddOptions defines the options that are used to add resources defined by a image vector to a component descriptor
type AddOptions struct {
	// ComponentDescriptorPath is the path to the component descriptor
	ComponentDescriptorPath string
	// ImageVectorPath defines the path to the image vector defined as yaml or json
	ImageVectorPath string

	iv.ParseImageOptions
	// GenericDependencies is a comma separated list of generic dependency names.
	// The list will be merged with the parse image options names.
	GenericDependencies string

	// OciOptions contains all exposed options to configure the oci client.
	OciOptions ociopts.Options
}

// NewAddCommand creates a command to add additional resources to a component descriptor.
func NewAddCommand(ctx context.Context) *cobra.Command {
	opts := &AddOptions{}
	cmd := &cobra.Command{
		Use:   "add --comp-desc component-descriptor-file --image-vector images.yaml [--component-prefixes \"github.com/gardener/myproj\"]... [--generic-dependency image-source-name]... [--generic-dependencies \"image-name1,image-name2\"]",
		Short: "Adds all resources of a image vector to the component descriptor",
		Long: `
add parses a image vector and generates or enhances the corresponding component descriptor resources.

There are 4 different scenarios how images are added to the component descriptor.
1. The image is defined with a tag and will be directly translated as oci image resource.

<pre>
images:
- name: pause-container
  sourceRepository: github.com/kubernetes/kubernetes/blob/master/build/pause/Dockerfile
  repository: gcr.io/google_containers/pause-amd64
  tag: "3.1"
</pre>

<pre>
meta:
  schemaVersion: 'v2'
...
resources:
- name: pause-container
  version: "3.1"
  type: ociImage
  extraIdentity:
    "imagevector-gardener-cloud+tag": "3.1"
  labels:
  - name: imagevector.gardener.cloud/name
    value: pause-container
  - name: imagevector.gardener.cloud/repository
    value: gcr.io/google_containers/pause-amd64
  - name: imagevector.gardener.cloud/source-repository
    value: github.com/kubernetes/kubernetes/blob/master/build/pause/Dockerfile
  access:
    type: ociRegistry
    imageReference: gcr.io/google_containers/pause-amd64:3.1
</pre>

2. The image is defined by another component so the image is added as label ("imagevector.gardener.cloud/images") to the "componentReference".

Images that are defined by other components can be specified 
1. when the image's repository matches the given "--component-prefixes"
2. the image is labeled with "imagevector.gardener.cloud/component-reference"

If the component reference is not yet defined it will be automatically added.
If multiple images are defined for the same component reference they are added to the images list in the label.

<pre>
images:
- name: cluster-autoscaler
  sourceRepository: github.com/gardener/autoscaler
  repository: eu.gcr.io/gardener-project/gardener/autoscaler/cluster-autoscaler
  targetVersion: "< 1.16"
  tag: "v0.10.0"
  labels: # recommended bbut only needed when "--component-prefixes" is not defined
  - name: imagevector.gardener.cloud/component-reference
    value:
      name: cla # defaults to image.name
      componentName: github.com/gardener/autoscaler # defaults to image.sourceRepository
      version: v0.10.0 # defaults to image.version
</pre>

<pre>
meta:
  schemaVersion: 'v2'
...
componentReferences:
- name: cla
  componentName: github.com/gardener/autoscaler
  version: v0.10.0
  extraIdentity:
    imagevector-gardener-cloud+tag: v0.10.0
  labels:
  - name: imagevector.gardener.cloud/images
    value:
	  images:
	  - name: cluster-autoscaler
	    repository: eu.gcr.io/gardener-project/gardener/autoscaler/cluster-autoscaler
	    sourceRepository: github.com/gardener/autoscaler
	    tag: v0.10.0
	    targetVersion: '< 1.16'
</pre>

3. The image is a generic dependency where the actual images are defined by the overwrite.
A generic dependency image is not part of a component descriptor's resource but will be added as label ("imagevector.gardener.cloud/images") to the component descriptor. 

Generic dependencies can be defined by
1. defined as "--generic-dependency=<image name>"
2. the label "imagevector.gardener.cloud/generic"

<pre>
images:
- name: hyperkube
  sourceRepository: github.com/kubernetes/kubernetes
  repository: k8s.gcr.io/hyperkube
  targetVersion: "< 1.19"
  labels: # only needed if "--generic-dependency" is not set
  - name: imagevector.gardener.cloud/generic
</pre>

<pre>
meta:
  schemaVersion: 'v2'
component:
  labels:
  - name: imagevector.gardener.cloud/images
    value:
	  images:
	  - name: hyperkube
	    repository: k8s.gcr.io/hyperkube
	    sourceRepository: github.com/kubernetes/kubernetes
	    targetVersion: '< 1.19'
</pre>

4. The image has not tag and it's repository matches a already defined resource in the component descriptor.
This usually means that the image is build as part of the build pipeline and the version depends on the current component.
In this case only labels are added to the existing resource

<pre>
images:
- name: gardenlet
  sourceRepository: github.com/gardener/gardener
  repository: eu.gcr.io/gardener-project/gardener/gardenlet
</pre>

<pre>
meta:
  schemaVersion: 'v2'
...
resources:
- name: gardenlet
  version: "v0.0.0"
  type: ociImage
  relation: local
  labels:
  - name: imagevector.gardener.cloud/name
    value: gardenlet
  - name: imagevector.gardener.cloud/repository
    value: eu.gcr.io/gardener-project/gardener/gardenlet
  - name: imagevector.gardener.cloud/source-repository
    value: github.com/gardener/gardener
  access:
    type: ociRegistry
    imageReference: eu.gcr.io/gardener-project/gardener/gardenlet:v0.0.0
</pre>

`,
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
	utils.CleanMarkdownUsageFunc(cmd)
	return cmd
}

func (o *AddOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	data, err := vfs.ReadFile(fs, o.ComponentDescriptorPath)
	if err != nil {
		return fmt.Errorf("unable to read component descriptor from %q: %s", o.ComponentDescriptorPath, err.Error())
	}

	ociClient, _, err := o.OciOptions.Build(log, fs)
	if err != nil {
		return err
	}
	compResolver := cdoci.NewResolver(ociClient).
		WithLog(log)
	if len(os.Getenv(constants.ComponentRepositoryCacheDirEnvVar)) != 0 {
		compResolver.WithCache(components.NewLocalComponentCache(fs))
	}

	// add the input to the ctf format
	cd := &cdv2.ComponentDescriptor{}
	if err := codec.Decode(data, cd); err != nil {
		return fmt.Errorf("unable to decode component descriptor from %q: %s", o.ComponentDescriptorPath, err.Error())
	}

	if err := o.parseImageVector(ctx, compResolver, cd, fs); err != nil {
		return err
	}

	if err := cdvalidation.Validate(cd); err != nil {
		return fmt.Errorf("invalid component descriptor: %w", err)
	}

	data, err = yaml.Marshal(cd)
	if err != nil {
		return fmt.Errorf("unable to encode component descriptor: %w", err)
	}
	if err := vfs.WriteFile(fs, o.ComponentDescriptorPath, data, 0664); err != nil {
		return fmt.Errorf("unable to write modified comonent descriptor: %w", err)
	}
	log.V(2).Info("Successfully added all resources from the image vector to component descriptor")
	return nil
}

func (o *AddOptions) Complete(args []string) error {

	// default component path to env var
	if len(o.ComponentDescriptorPath) == 0 {
		o.ComponentDescriptorPath = filepath.Dir(os.Getenv(constants.ComponentDescriptorPathEnvName))
	}

	// parse generic dependencies
	if len(o.GenericDependencies) != 0 {
		for _, genericDepName := range strings.Split(o.GenericDependencies, ",") {
			o.ParseImageOptions.GenericDependencies = append(o.ParseImageOptions.GenericDependencies, strings.TrimSpace(genericDepName))
		}
	}

	return o.validate()
}

func (o *AddOptions) validate() error {
	if len(o.ComponentDescriptorPath) == 0 {
		return errors.New("component descriptor path must be provided")
	}
	if len(o.ImageVectorPath) == 0 {
		return errors.New("images path must be provided")
	}
	return nil
}

func (o *AddOptions) AddFlags(set *pflag.FlagSet) {
	set.StringVar(&o.ComponentDescriptorPath, "comp-desc", "", "path to the component descriptor directory")
	set.StringVar(&o.ImageVectorPath, "image-vector", "", "The path to the resources defined as yaml or json")
	set.StringArrayVar(&o.ParseImageOptions.ComponentReferencePrefixes, "component-prefixes", []string{}, "Specify all prefixes that define a image  from another component")
	set.StringArrayVar(&o.ParseImageOptions.ExcludeComponentReference, "exclude-component-reference", []string{}, "Specify all image name that should not be added as component reference")
	set.StringArrayVar(&o.ParseImageOptions.GenericDependencies, "generic-dependency", []string{}, "Specify all image source names that are a generic dependency.")
	set.StringVar(&o.GenericDependencies, "generic-dependencies", "", "Specify all prefixes that define a image  from another component")
	o.OciOptions.AddFlags(set)
}

// parseImageVector parses the given image vector and returns a list of all resources.
func (o *AddOptions) parseImageVector(ctx context.Context, compResolver ctf.ComponentResolver, cd *cdv2.ComponentDescriptor, fs vfs.FileSystem) error {
	file, err := fs.Open(o.ImageVectorPath)
	if err != nil {
		return fmt.Errorf("unable to open image vector file: %q: %w", o.ImageVectorPath, err)
	}
	defer file.Close()
	return iv.ParseImageVector(ctx, compResolver, cd, file, &o.ParseImageOptions)
}

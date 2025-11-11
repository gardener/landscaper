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
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/codec"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/landscaper/legacy-component-spec/bindings-go/oci"

	ociopts "github.com/gardener/landscaper/legacy-component-cli/ociclient/options"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/constants"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/components"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/utils"
)

// GenerateOverwriteOptions defines the options that are used to generate a image vector from component descriptors
type GenerateOverwriteOptions struct {
	// BaseURL defines the repository base url of the remote repository
	// +optional
	BaseURL string
	// ComponentRefOrPath is the name and version of the main component or a path to the local component descriptor
	// the component ref is expected to be of the format "<component-name>:<component-version>"
	// +optional
	ComponentRefOrPath string

	// AdditionalComponentsRefOrPath is a list of  name and version of the main component or a path to the local component descriptors
	// +optional
	AdditionalComponentsRefOrPath []string

	// ImageVectorPath defines the path to the image vector defined as yaml or json
	ImageVectorPath string
	// ResolveTags enables
	ResolveTags bool

	// OciOptions contains all exposed options to configure the oci client.
	OciOptions ociopts.Options

	ComponentRepository cdv2.Repository
}

// NewGenerateOverwriteCommand creates a command to add additional resources to a component descriptor.
func NewGenerateOverwriteCommand(ctx context.Context) *cobra.Command {
	opts := &GenerateOverwriteOptions{}
	cmd := &cobra.Command{
		Use:     "generate-overwrite --component=\"example.com/my/component/name:v0.0.1 | /path/to/local/component-descriptor\" -o IV_OVERWRITE_OUTPUT_PATH [--add-comp=ADDITIONAL_COMPONENT]...",
		Aliases: []string{"go"},
		Short:   "Get parses a component descriptor and returns the defined image vector",
		Long: `
generate-overwrite parses images defined in a component descriptor and returns them as image vector.

Images can be defined in a component descriptor in 3 different ways:
1. as 'ociImage' resource: The image is defined a default resource of type 'ociImage' with a access of type 'ociRegistry'.
   It is expected that the resource contains the following labels to be identified as image vector image.
   The resulting image overwrite will contain the repository and the tag/digest from the access method.
<pre>
resources:
- name: pause-container
  version: "3.1"
  type: ociImage
  relation: external
  extraIdentity:
    "imagevector-gardener-cloud+tag": "3.1"
  labels:
  - name: imagevector.gardener.cloud/name
    value: pause-container
  - name: imagevector.gardener.cloud/repository
    value: gcr.io/google_containers/pause-amd64
  - name: imagevector.gardener.cloud/source-repository
    value: github.com/kubernetes/kubernetes/blob/master/build/pause/Dockerfile
  - name: imagevector.gardener.cloud/target-version
    value: "< 1.16"
  access:
    type: ociRegistry
    imageReference: gcr.io/google_containers/pause-amd64:3.1
</pre>

2. as component reference: The images are defined in a label "imagevector.gardener.cloud/images".
   The resulting image overwrite will contain all images defined in the images label.
   Their repository and tag/digest will be matched from the resources defined in the actual component's resources.

   Note: The images from the label are matched to the resources using their name and version. The original image reference do not exit anymore.

<pre>
componentReferences:
- name: cluster-autoscaler-abc
  componentName: github.com/gardener/autoscaler
  version: v0.10.1
  labels:
  - name: imagevector.gardener.cloud/images
    value:
      images:
      - name: cluster-autoscaler
        repository: eu.gcr.io/gardener-project/gardener/autoscaler/cluster-autoscaler
        tag: "v0.10.1"
</pre>

3. as generic images from the component descriptor labels.
   Generic images are images that do not directly result in a resource.
   They will be matched with another component descriptor that actually defines the images.
   The other component descriptor MUST have the "imagevector.gardener.cloud/name" label in order to be matched.

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
        targetVersion: "< 1.19"
</pre>

<pre>
meta:
  schemaVersion: 'v2'
component:
  resources:
  - name: hyperkube
    version: "v1.19.4"
    type: ociImage
    extraIdentity:
      "imagevector-gardener-cloud+tag": "v1.19.4"
    labels:
    - name: imagevector.gardener.cloud/name
      value: hyperkube
    - name: imagevector.gardener.cloud/repository
      value: k8s.gcr.io/hyperkube
    access:
	  type: ociRegistry
	  imageReference: my-registry/hyperkube:v1.19.4
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

func (o *GenerateOverwriteOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	ctx = logr.NewContext(ctx, log)
	ociClient, _, err := o.OciOptions.Build(log, fs)
	if err != nil {
		return err
	}
	compResolver := cdoci.NewResolver(ociClient).
		WithLog(log)
	if len(os.Getenv(constants.ComponentRepositoryCacheDirEnvVar)) != 0 {
		compResolver.WithCache(components.NewLocalComponentCache(fs))
	}

	mainComponent, err := ResolveComponentDescriptorFromComponentRefOrPath(ctx, fs, compResolver, o.ComponentRepository, o.ComponentRefOrPath)
	if err != nil {
		return err
	}

	cdList := &cdv2.ComponentDescriptorList{}
	for _, additionalCompStr := range o.AdditionalComponentsRefOrPath {
		comp, err := ResolveComponentDescriptorFromComponentRefOrPath(ctx, fs, compResolver, o.ComponentRepository, additionalCompStr)
		if err != nil {
			return err
		}
		cdList.Components = append(cdList.Components, *comp)
	}

	imageVector, err := iv.GenerateImageOverwrite(ctx, compResolver, mainComponent, iv.GenerateImageOverwriteOptions{
		Components:         cdList,
		ReplaceWithDigests: o.ResolveTags,
		OciClient:          ociClient,
	})
	if err != nil {
		return fmt.Errorf("unable to parse image vector: %s", err.Error())
	}

	data, err := yaml.Marshal(imageVector)
	if err != nil {
		return fmt.Errorf("unable to encode image vector: %w", err)
	}
	if len(o.ImageVectorPath) != 0 {
		if err := fs.MkdirAll(filepath.Dir(o.ImageVectorPath), os.ModePerm); err != nil {
			return fmt.Errorf("unable to create directories for %q: %s", o.ImageVectorPath, err.Error())
		}
		if err := vfs.WriteFile(fs, o.ImageVectorPath, data, 06444); err != nil {
			return fmt.Errorf("unable to write image vector: %w", err)
		}
		fmt.Printf("Successfully generated image vector from component descriptor")
	} else {
		fmt.Println(string(data))
	}
	return nil
}

func (o *GenerateOverwriteOptions) Complete(args []string) error {
	if len(o.BaseURL) == 0 {
		o.BaseURL = os.Getenv(constants.ComponentRepositoryRepositoryBaseUrlEnvName)
	}

	if err := o.validate(); err != nil {
		return err
	}
	o.ComponentRepository = cdv2.NewOCIRegistryRepository(o.BaseURL, "")

	return nil
}

func (o *GenerateOverwriteOptions) validate() error {
	if len(o.ComponentRefOrPath) == 0 {
		return errors.New("component descriptor path or a remote component descriptor must be provided")
	}
	return nil
}

func (o *GenerateOverwriteOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.BaseURL, "repo-ctx", "", "base url of the component repository")
	fs.StringVarP(&o.ComponentRefOrPath, "component", "c", "", "name and version of the main component or a path to the local component descriptor. The component ref is expected to be of the format '<component-name>:<component-version>'")
	fs.StringArrayVar(&o.AdditionalComponentsRefOrPath, "add-comp", []string{}, "list of name and version of an additional component or a path to the local component descriptor. The component ref is expected to be of the format '<component-name>:<component-version>'")

	fs.StringVarP(&o.ImageVectorPath, "output", "o", "", "The path to the image vector that will be written.")
	fs.BoolVar(&o.ResolveTags, "resolve-tags", false, "enable that tags are automatically resolved to digests")
	o.OciOptions.AddFlags(fs)
}

type ComponentRefOrPath struct {
	Name    string
	Version string
	Path    string
}

// ParseComponentRefOrPath parses a component that is either defined by a component ref or a path.
func ParseComponentRefOrPath(c string) (ComponentRefOrPath, error) {
	// check if string is a ref by checking for ":"
	if strings.Contains(c, ":") {
		ref := strings.Split(c, ":")
		if len(ref) != 2 {
			return ComponentRefOrPath{}, fmt.Errorf("expected the ref to be of the form '<component name>:<component version>' but got %q", c)
		}
		return ComponentRefOrPath{
			Name:    ref[0],
			Version: ref[1],
		}, nil
	}
	// expect a path
	return ComponentRefOrPath{
		Path: c,
	}, nil
}

// ResolveComponentDescriptor resolves a component descriptor from a ComponentRefOrPath
func ResolveComponentDescriptor(ctx context.Context,
	fs vfs.FileSystem,
	resolver ctf.ComponentResolver,
	repoCtx cdv2.Repository,
	comp ComponentRefOrPath) (*cdv2.ComponentDescriptor, error) {
	if len(comp.Path) != 0 {
		// read component descriptor from local path
		data, err := vfs.ReadFile(fs, comp.Path)
		if err != nil {
			return nil, fmt.Errorf("unable to read component descriptor from %q: %s", comp.Path, err.Error())
		}

		// add the input to the ctf format
		cd := &cdv2.ComponentDescriptor{}
		if err := codec.Decode(data, cd); err != nil {
			return nil, fmt.Errorf("unable to decode component descriptor from %q: %s", comp.Path, err.Error())
		}
		return cd, nil
	}

	return resolver.Resolve(ctx, repoCtx, comp.Name, comp.Version)
}

// ResolveComponentDescriptorFromComponentRefOrPath resolves a component descriptor from a ComponentRefOrPath
func ResolveComponentDescriptorFromComponentRefOrPath(
	ctx context.Context,
	fs vfs.FileSystem,
	resolver ctf.ComponentResolver,
	repoCtx cdv2.Repository,
	compStr string) (*cdv2.ComponentDescriptor, error) {
	mainComponent, err := ParseComponentRefOrPath(compStr)
	if err != nil {
		return nil, err
	}
	return ResolveComponentDescriptor(ctx, fs, resolver, repoCtx, mainComponent)
}

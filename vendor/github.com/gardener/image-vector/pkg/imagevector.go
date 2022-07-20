// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/apis/v2/cdutils"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	"sigs.k8s.io/yaml"
)

// ParseImageOptions are options to configure the image vector parsing.
type ParseImageOptions struct {
	// ComponentReferencePrefixes are prefixes that are used to identify images from other components.
	// These images are then not added as direct resources but the source repository is used as the component reference.
	ComponentReferencePrefixes []string
	// ExcludeComponentReference defines a list of image names that should be added as component reference
	ExcludeComponentReference []string
	// GenericDependencies define images that should be untouched and not added as real dependency to the component descriptors.
	// These dependencies are added a specific label to the component descriptor.
	GenericDependencies []string
	// IgnoreDeprecatedFlags ignores the deprecated parse options.
	IgnoreDeprecatedFlags bool
}

// describes all available actions expressed through labels
var (
	ComponentReferenceAction = Label("component-reference")
	GenericDependencyAction  = Label("generic")
	IgnoreFlagsAction        = Label("ignore-flags")
)

// ComponentReferenceLabelValue is the value configuration for the component reference
type ComponentReferenceLabelValue struct {
	Name          string `json:"name,omitempty" yaml:"name,omitempty"`
	ComponentName string `json:"componentName,omitempty" yaml:"componentName,omitempty"`
	Version       string `json:"version,omitempty" yaml:"version,omitempty"`
}

// ParseImageVector parses a image vector and generates the corresponding component descriptor resources.
// It is expected that the image vector yaml is passed as io.Reader.
//
// There are 4 different scenarios how images are added to the component descriptor.
// 1. The image is defined with a tag and will be directly translated as oci image resource.
//
// <pre>
// images:
// - name: pause-container
//   sourceRepository: github.com/kubernetes/kubernetes/blob/master/build/pause/Dockerfile
//   repository: gcr.io/google_containers/pause-amd64
//   tag: "3.1"
// </pre>
//
// <pre>
// meta:
//   schemaVersion: 'v2'
// ...
// resources:
// - name: pause-container
//   version: "3.1"
//   type: ociImage
//   extraIdentity:
//     "imagevector-gardener-cloud+tag": "3.1"
//   labels:
//   - name: imagevector.gardener.cloud/name
//     value: pause-container
//   - name: imagevector.gardener.cloud/repository
//     value: gcr.io/google_containers/pause-amd64
//   - name: imagevector.gardener.cloud/source-repository
//     value: github.com/kubernetes/kubernetes/blob/master/build/pause/Dockerfile
//   access:
//     type: ociRegistry
//     imageReference: gcr.io/google_containers/pause-amd64:3.1
// </pre>
//
// 2. The image is defined by another component so the image is added as label ("imagevector.gardener.cloud/images") to the "componentReference".
//
// Images that are defined by other components can be specified
// 1. when the image's repository matches the given "--component-prefixes"
// 2. the image is labeled with "imagevector.gardener.cloud/component-reference"
//
// If the component reference is not yet defined it will be automatically added.
// If multiple images are defined for the same component reference they are added to the images list in the label.
//
// <pre>
// images:
// - name: cluster-autoscaler
//   sourceRepository: github.com/gardener/autoscaler
//   repository: eu.gcr.io/gardener-project/gardener/autoscaler/cluster-autoscaler
//   targetVersion: "< 1.16"
//   tag: "v0.10.0"
//   labels: # recommended bbut only needed when "--component-prefixes" is not defined
//   - name: imagevector.gardener.cloud/component-reference
//     value:
//       name: cla # defaults to image.name
//       componentName: github.com/gardener/autoscaler # defaults to image.sourceRepository
//       version: v0.10.0 # defaults to image.version
// </pre>
//
// <pre>
// meta:
//   schemaVersion: 'v2'
// ...
// componentReferences:
// - name: cla
//   componentName: github.com/gardener/autoscaler
//   version: v0.10.0
//   extraIdentity:
//     imagevector-gardener-cloud+tag: v0.10.0
//   labels:
//   - name: imagevector.gardener.cloud/images
//     value:
//     images:
//	 - name: cluster-autoscaler
//	   repository: eu.gcr.io/gardener-project/gardener/autoscaler/cluster-autoscaler
//	   sourceRepository: github.com/gardener/autoscaler
//	   tag: v0.10.0
//	   targetVersion: '< 1.16'
//  </pre>
//
// 3. The image is a generic dependency where the actual images are defined by the overwrite.
// A generic dependency image is not part of a component descriptor's resource but will be added as label ("imagevector.gardener.cloud/images") to the component descriptor.
//
// Generic dependencies can be defined by
// 1. defined as "--generic-dependency=<image name>"
// 2. the label "imagevector.gardener.cloud/generic"
//
// <pre>
// images:
// - name: hyperkube
//   sourceRepository: github.com/kubernetes/kubernetes
//   repository: k8s.gcr.io/hyperkube
//   targetVersion: "< 1.19"
//   labels: # only needed if "--generic-dependency" is not set
//   - name: imagevector.gardener.cloud/generic
// </pre>
//
// <pre>
// meta:
//   schemaVersion: 'v2'
// component:
//   labels:
//   - name: imagevector.gardener.cloud/images
//     value:
//     images:
//	 - name: hyperkube
//	   repository: k8s.gcr.io/hyperkube
//	   sourceRepository: github.com/kubernetes/kubernetes
//	   targetVersion: '< 1.19'
//  </pre>
//
// 4. The image has not tag and it's repository matches a already defined resource in the component descriptor.
// This usually means that the image is build as part of the build pipeline and the version depends on the current component.
// In this case only labels are added to the existing resource
//
// <pre>
// images:
// - name: gardenlet
//   sourceRepository: github.com/gardener/gardener
//   repository: eu.gcr.io/gardener-project/gardener/gardenlet
// </pre>
//
// <pre>
// meta:
//   schemaVersion: 'v2'
// ...
// resources:
// - name: gardenlet
//   version: "v0.0.0"
//   type: ociImage
//   relation: local
//   labels:
//   - name: imagevector.gardener.cloud/name
//     value: gardenlet
//   - name: imagevector.gardener.cloud/repository
//     value: eu.gcr.io/gardener-project/gardener/gardenlet
//   - name: imagevector.gardener.cloud/source-repository
//     value: github.com/gardener/gardener
//   access:
//     type: ociRegistry
//     imageReference: eu.gcr.io/gardener-project/gardener/gardenlet:v0.0.0
// </pre>
func ParseImageVector(ctx context.Context,
	compResolver ctf.ComponentResolver,
	cd *cdv2.ComponentDescriptor,
	reader io.Reader,
	opts *ParseImageOptions) error {

	imageVector, err := DecodeImageVector(reader)
	if err != nil {
		return fmt.Errorf("unable to decode image vector: %w", err)
	}

	if _, ok := cdutils.GetLabel(imageVector.Labels, IgnoreFlagsAction); ok {
		opts.IgnoreDeprecatedFlags = true
	}

	ip := imageParser{
		opts:               opts,
		genericImageVector: &ImageVector{},
		cd:                 cd,
		compResolver:       compResolver,
	}

	for _, image := range imageVector.Images {
		if err := ip.Parse(ctx, image); err != nil {
			return err
		}
	}

	// parse label
	if len(ip.genericImageVector.Images) != 0 {
		genericImageVectorBytes, err := json.Marshal(ip.genericImageVector)
		if err != nil {
			return fmt.Errorf("unable to parse generic image vector: %w", err)
		}
		cd.Labels = cdutils.SetRawLabel(cd.Labels,
			ImagesLabel, genericImageVectorBytes)
	}

	return nil
}

func DecodeImageVector(r io.Reader) (*ImageVector, error) {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return nil, err
	}
	data, err := yaml.YAMLToJSON(buf.Bytes())
	if err != nil {
		return nil, err
	}

	imageVector := &ImageVector{}
	if err := json.Unmarshal(data, imageVector); err != nil {
		return nil, fmt.Errorf("unable to decode image vector: %w", err)
	}
	return imageVector, nil
}

type imageParser struct {
	opts               *ParseImageOptions
	genericImageVector *ImageVector
	cd                 *cdv2.ComponentDescriptor
	compResolver       ctf.ComponentResolver
}

func (ip *imageParser) Parse(ctx context.Context, image ImageEntry) error {
	if image.Tag == nil {
		// directly set explicit generic images.
		if ImageEntryIsGenericDependency(image, ip.opts) {
			ip.genericImageVector.Images = append(ip.genericImageVector.Images, image)
			return nil
		}

		// check if the image does already exist in the component descriptor
		found, err := addLabelsToInlineResource(ip.cd.Resources, image)
		if err != nil {
			return err
		}
		if found {
			return nil
		}

		// default all non inlined resources that have no tag as generic images.
		ip.genericImageVector.Images = append(ip.genericImageVector.Images, image)
		return nil
	}

	if ImageEntryIsComponentReference(image, ip.opts) {
		return ip.AddAsComponentReference(ctx, image)
	}

	res := cdv2.Resource{
		IdentityObjectMeta: cdv2.IdentityObjectMeta{
			Labels: make([]cdv2.Label, 0),
		},
	}
	res.Name = image.Name
	res.Type = cdv2.OCIImageType
	res.Relation = cdv2.ExternalRelation

	if err := addLabelsToResource(&res, image); err != nil {
		return err
	}

	var ociImageAccess cdv2.TypedObjectAccessor
	if TagIsDigest(*image.Tag) {
		res.Version = ip.cd.GetVersion() // default to component descriptor version
		ociImageAccess = cdv2.NewOCIRegistryAccess(image.Repository + "@" + *image.Tag)
	} else {
		res.Version = *image.Tag
		ociImageAccess = cdv2.NewOCIRegistryAccess(image.Repository + ":" + *image.Tag)
	}

	uObj, err := cdv2.NewUnstructured(ociImageAccess)
	if err != nil {
		return fmt.Errorf("unable to create oci registry access for %q: %w", image.Name, err)
	}
	res.Access = &uObj

	// add resource
	id := ip.cd.GetResourceIndex(res)
	if id != -1 {
		if err := preventLossOfTargetVersionLabel(&ip.cd.Resources[id], &res); err != nil {
			return err
		}

		ip.cd.Resources[id] = cdutils.MergeResources(ip.cd.Resources[id], res)
	} else {
		ip.cd.Resources = append(ip.cd.Resources, res)
	}
	return nil
}

// preventLossOfTargetVersionLabel throws an error if the provided resources both have a target version label
// with different values. In this case, one of the labels would get lost if the resources are merged.
func preventLossOfTargetVersionLabel(res1, res2 *cdv2.Resource) error {
	var (
		targetVersion1 string
		targetVersion2 string
	)

	hasLabel1, err := getLabel(res1.Labels, TargetVersionLabel, &targetVersion1)
	if err != nil {
		return err
	}

	hasLabel2, err := getLabel(res2.Labels, TargetVersionLabel, &targetVersion2)
	if err != nil {
		return err
	}

	if hasLabel1 && hasLabel2 && targetVersion1 != targetVersion2 {
		tag := res2.IdentityObjectMeta.GetIdentity()[TagExtraIdentity]

		return fmt.Errorf(`there is more than one target version expression specified for name %q and tag %q. `+
			`A solution might be to combine the target version expressions by using a range, for example: `+
			`targetVersion: ">= 1.18, < 1.22"`,
			res2.Name, tag)
	}

	return nil
}

// ImageEntryIsGenericDependency checks if the image entry should be parsed as generic dependency
func ImageEntryIsGenericDependency(image ImageEntry, opts *ParseImageOptions) bool {
	// favor labels over deprecated cli flags
	if entryHasAction(image, GenericDependencyAction) {
		return true
	}
	if opts.IgnoreDeprecatedFlags || entryHasAction(image, IgnoreFlagsAction) {
		return false
	}
	return entryMatchesPrefix(opts.GenericDependencies, image.Name)
}

// ImageEntryIsComponentReference checks if the image entry should be parsed as component reference
func ImageEntryIsComponentReference(image ImageEntry, opts *ParseImageOptions) bool {
	// favor labels over deprecated cli flags
	if entryHasAction(image, ComponentReferenceAction) {
		return true
	}
	if opts.IgnoreDeprecatedFlags || entryHasAction(image, IgnoreFlagsAction) {
		return false
	}
	if isOneOf(opts.ExcludeComponentReference, image.Name) {
		return false
	}
	return entryMatchesPrefix(opts.ComponentReferencePrefixes, image.Repository)
}

func (ip *imageParser) AddAsComponentReference(ctx context.Context, image ImageEntry) error {
	// add image as component reference
	ref := cdv2.ComponentReference{
		Name:          image.Name,
		ComponentName: image.SourceRepository,
		Version:       *image.Tag,
		ExtraIdentity: map[string]string{
			TagExtraIdentity: *image.Tag,
		},
		Labels: make([]cdv2.Label, 0),
	}

	if label, ok := cdutils.GetLabel(image.Labels, ComponentReferenceAction); ok {
		// overwrite default values from the image that are given by the labels
		values := ComponentReferenceLabelValue{}
		if err := json.Unmarshal(label.Value, &values); err != nil {
			return fmt.Errorf("unable to parse component reference value: %w", err)
		}
		if len(values.Name) != 0 {
			ref.Name = values.Name
		}
		if len(values.ComponentName) != 0 {
			ref.ComponentName = values.ComponentName
		}
		if len(values.Version) != 0 {
			ref.Version = values.Version
			ref.ExtraIdentity = map[string]string{
				TagExtraIdentity: values.Version,
			}
		}
	}

	// resolve component
	refCompDesc, err := ip.compResolver.Resolve(ctx, ip.cd.GetEffectiveRepositoryContext(), ref.ComponentName, ref.Version)
	if err != nil {
		return fmt.Errorf("image %s is defined by a external component %s:%s but it's ComponentDescriptor cannot be resolved: %w",
			image.Name, ref.ComponentName, ref.Version, err)
	}

	// try to find the correct resource name/identity for the image
	resourceID, err := tryFindResourceForImage(ctx, refCompDesc, image)
	if err != nil {
		return fmt.Errorf("image %s is defined by a external component %s:%s but cannot be found in that component descriptor",
			image.Name, refCompDesc.Name, refCompDesc.Version)
	}

	// add complete image as label
	ip.cd.ComponentReferences, err = addComponentReference(ip.cd.ComponentReferences, ref, ComponentReferenceImageEntry{
		ImageEntry: image,
		ResourceID: resourceID,
	})
	if err != nil {
		return fmt.Errorf("unable to add component reference for %q: %w", image.Name, err)
	}
	return nil
}

func tryFindResourceForImage(ctx context.Context, cd *cdv2.ComponentDescriptor, image ImageEntry) (cdv2.Identity, error) {
	log := logr.FromContextOrDiscard(ctx)
	var matchedImgID cdv2.Identity
	for _, res := range cd.Resources {
		// first try to match the resource name
		if image.Name == res.Name {
			return res.GetIdentity(), nil
		}
		// otherwise try to match the imageReference
		if res.Type != cdv2.OCIImageType {
			continue
		}
		// the ref can only be matched if the oci image is defined by a ociRegistry access
		if res.Access.GetType() != cdv2.OCIRegistryType {
			continue
		}
		acc := &cdv2.OCIRegistryAccess{}
		if err := res.Access.DecodeInto(acc); err != nil {
			log.Error(err, "unable to decode into oci registry", "resource", res.Name)
			continue
		}
		repo, _, err := ParseImageRef(acc.ImageReference)
		if err != nil {
			log.Error(err, "unable to parse image reference", "resource", res.Name, "ref", acc.ImageReference)
			continue
		}
		if repo == image.Repository {
			matchedImgID = res.GetIdentity()
		}
	}
	if matchedImgID != nil {
		return matchedImgID, nil
	}

	return nil, ReferencedResourceNotFoundError
}

// addLabelsToInlineResource adds the image entry labels to the resource that matches the repository
func addLabelsToInlineResource(resources []cdv2.Resource, imageEntry ImageEntry) (bool, error) {
	for i, res := range resources {
		if res.GetType() != cdv2.OCIImageType {
			continue
		}
		if res.Access.GetType() != cdv2.OCIRegistryType {
			continue
		}
		// resource is a oci image with a registry type
		ociImageAccess := &cdv2.OCIRegistryAccess{}
		if err := res.Access.DecodeInto(ociImageAccess); err != nil {
			return false, fmt.Errorf("unable to decode resource access into oci registry access for %q: %w", res.GetName(), err)
		}

		repo, _, err := ParseImageRef(ociImageAccess.ImageReference)
		if err != nil {
			return false, fmt.Errorf("unable to parse image reference for %q: %w", res.GetName(), err)
		}
		if repo != imageEntry.Repository {
			continue
		}

		if err := addLabelsToResource(&resources[i], imageEntry); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// addLabelsToResource adds internal image vector labels to the given resource.
func addLabelsToResource(res *cdv2.Resource, imageEntry ImageEntry) error {
	var err error
	res.Labels, err = cdutils.SetLabel(res.Labels, NameLabel, imageEntry.Name)
	if err != nil {
		return fmt.Errorf("unable to add name label to resource for image %q: %w", imageEntry.Name, err)
	}

	for _, label := range imageEntry.Labels {
		res.Labels = cdutils.SetRawLabel(res.Labels, label.Name, label.Value)
	}

	if len(imageEntry.Repository) != 0 {
		res.Labels, err = cdutils.SetLabel(res.Labels, RepositoryLabel, imageEntry.Repository)
		if err != nil {
			return fmt.Errorf("unable to add repository label to resource for image %q: %w", imageEntry.Name, err)
		}
	}
	if len(imageEntry.SourceRepository) != 0 {
		res.Labels, err = cdutils.SetLabel(res.Labels, SourceRepositoryLabel, imageEntry.SourceRepository)
		if err != nil {
			return fmt.Errorf("unable to add source repository label to resource for image %q: %w", imageEntry.Name, err)
		}
	}
	if imageEntry.TargetVersion != nil {
		res.Labels, err = cdutils.SetLabel(res.Labels, TargetVersionLabel, imageEntry.TargetVersion)
		if err != nil {
			return fmt.Errorf("unable to add target version label to resource for image %q: %w", imageEntry.Name, err)
		}
	}
	if imageEntry.RuntimeVersion != nil {
		res.Labels, err = cdutils.SetLabel(res.Labels, RuntimeVersionLabel, imageEntry.RuntimeVersion)
		if err != nil {
			return fmt.Errorf("unable to add target version label to resource for image %q: %w", imageEntry.Name, err)
		}
	}

	// set the tag as identity
	if imageEntry.Tag != nil {
		cdutils.SetExtraIdentityField(&res.IdentityObjectMeta, TagExtraIdentity, *imageEntry.Tag)
	}
	return nil
}

// addComponentReference adds the given component to the list of component references.
// if the component is already declared, the given image entry is appended to the images label
func addComponentReference(refs []cdv2.ComponentReference, new cdv2.ComponentReference, entry ComponentReferenceImageEntry) ([]cdv2.ComponentReference, error) {
	for i, ref := range refs {
		if ref.Name == new.Name && ref.Version == new.Version {
			// parse current images and add the image
			imageVector := &ComponentReferenceImageVector{
				Images: []ComponentReferenceImageEntry{entry},
			}
			data, ok := ref.GetLabels().Get(ImagesLabel)
			if ok {
				if err := json.Unmarshal(data, imageVector); err != nil {
					return nil, err
				}
				imageVector.Images = append(imageVector.Images, entry)
			}
			var err error
			ref.Labels, err = cdutils.SetLabel(ref.Labels, ImagesLabel, imageVector)
			if err != nil {
				return nil, err
			}
			refs[i] = ref
			return refs, nil
		}
	}

	imageVector := ComponentReferenceImageVector{
		Images: []ComponentReferenceImageEntry{entry},
	}
	var err error
	new.Labels, err = cdutils.SetLabel(new.Labels, ImagesLabel, imageVector)
	if err != nil {
		return nil, err
	}
	return append(refs, new), nil
}

// parseResourceAccess parses a resource's access and sets the repository and tag on the given image entry.
// Currently only access of type 'ociRegistry' is supported.
func parseResourceAccess(imageEntry *ImageEntry, res cdv2.Resource) error {
	access := &cdv2.OCIRegistryAccess{}
	if err := cdv2.NewCodec(nil, nil, nil).Decode(res.Access.Raw, access); err != nil {
		return fmt.Errorf("unable to decode ociRegistry access: %w", err)
	}

	ref, tag, err := ParseImageRef(access.ImageReference)
	if err != nil {
		return fmt.Errorf("unable to parse image reference %q: %w", access.ImageReference, err)
	}

	imageEntry.Repository = ref
	imageEntry.Tag = &tag
	return nil
}

func getLabel(labels cdv2.Labels, name string, into interface{}) (bool, error) {
	val, ok := labels.Get(name)
	if !ok {
		return false, nil
	}

	if err := json.Unmarshal(val, into); err != nil {
		return true, err
	}
	return true, nil
}

func entryHasAction(entry ImageEntry, action string) bool {
	_, ok := cdutils.GetLabel(entry.Labels, action)
	return ok
}

func entryMatchesPrefix(prefixes []string, val string) bool {
	for _, pref := range prefixes {
		if strings.HasPrefix(val, pref) {
			return true
		}
	}
	return false
}

func isOneOf(keys []string, key string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

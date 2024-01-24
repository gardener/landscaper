// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v3alpha1

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/versions/ocm.software/v3alpha1/jsonscheme"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	SchemaVersion = GroupVersion

	VersionName  = "v3alpha1"
	GroupVersion = metav1.GROUP + "/" + VersionName
	Kind         = metav1.KIND
)

func init() {
	compdesc.RegisterScheme(&DescriptorVersion{})
}

type DescriptorVersion struct{}

var _ compdesc.Scheme = (*DescriptorVersion)(nil)

func (v *DescriptorVersion) GetVersion() string {
	return SchemaVersion
}

func (v *DescriptorVersion) Decode(data []byte, opts *compdesc.DecodeOptions) (compdesc.ComponentDescriptorVersion, error) {
	var cd ComponentDescriptor
	if !opts.DisableValidation {
		if err := jsonscheme.Validate(data); err != nil {
			return nil, err
		}
	}
	var err error
	if opts.StrictMode {
		err = opts.Codec.DecodeStrict(data, &cd)
	} else {
		err = opts.Codec.Decode(data, &cd)
	}
	if err != nil {
		return nil, err
	}

	if err := cd.Default(); err != nil {
		return nil, err
	}

	if !opts.DisableValidation {
		err = cd.Validate()
		if err != nil {
			return nil, err
		}
	}
	return &cd, err
}

////////////////////////////////////////////////////////////////////////////////
// convert to internal version
////////////////////////////////////////////////////////////////////////////////

func (v *DescriptorVersion) ConvertTo(obj compdesc.ComponentDescriptorVersion) (out *compdesc.ComponentDescriptor, err error) {
	if obj == nil {
		return nil, nil
	}
	in, ok := obj.(*ComponentDescriptor)
	if !ok {
		return nil, errors.Newf("%T is no version v2 descriptor", obj)
	}
	if in.Kind != Kind {
		return nil, errors.ErrInvalid("kind", in.Kind)
	}

	defer compdesc.CatchConversionError(&err)
	out = &compdesc.ComponentDescriptor{
		Metadata: compdesc.Metadata{ConfiguredVersion: in.APIVersion},
		ComponentSpec: compdesc.ComponentSpec{
			ObjectMeta:         *in.ObjectMeta.Copy(),
			RepositoryContexts: in.RepositoryContexts.Copy(),
			Sources:            convertSourcesTo(in.Spec.Sources),
			Resources:          convertResourcesTo(in.Spec.Resources),
			References:         convertReferencesTo(in.Spec.References),
		},
		Signatures:    in.Signatures.Copy(),
		NestedDigests: in.NestedDigests.Copy(),
	}
	return out, nil
}

func convertReferenceTo(in Reference) compdesc.ComponentReference {
	return compdesc.ComponentReference{
		ElementMeta:   convertElementmetaTo(in.ElementMeta),
		ComponentName: in.ComponentName,
		Digest:        in.Digest.Copy(),
	}
}

func convertReferencesTo(in []Reference) compdesc.References {
	out := make(compdesc.References, len(in))
	for i := range in {
		out[i] = convertReferenceTo(in[i])
	}
	return out
}

func convertSourceTo(in Source) compdesc.Source {
	return compdesc.Source{
		SourceMeta: compdesc.SourceMeta{
			ElementMeta: convertElementmetaTo(in.ElementMeta),
			Type:        in.Type,
		},
		Access: compdesc.GenericAccessSpec(in.Access.DeepCopy()),
	}
}

func convertSourcesTo(in Sources) compdesc.Sources {
	if in == nil {
		return nil
	}
	out := make(compdesc.Sources, len(in))
	for i := range in {
		out[i] = convertSourceTo(in[i])
	}
	return out
}

func convertElementmetaTo(in ElementMeta) compdesc.ElementMeta {
	return compdesc.ElementMeta{
		Name:          in.Name,
		Version:       in.Version,
		ExtraIdentity: in.ExtraIdentity.Copy(),
		Labels:        in.Labels.Copy(),
	}
}

func convertResourceTo(in Resource) compdesc.Resource {
	return compdesc.Resource{
		ResourceMeta: compdesc.ResourceMeta{
			ElementMeta: convertElementmetaTo(in.ElementMeta),
			Type:        in.Type,
			Relation:    in.Relation,
			SourceRef:   ConvertSourcerefsTo(in.SourceRef),
			Digest:      in.Digest.Copy(),
		},
		Access: compdesc.GenericAccessSpec(in.Access),
	}
}

func convertResourcesTo(in Resources) compdesc.Resources {
	if in == nil {
		return nil
	}
	out := make(compdesc.Resources, len(in))
	for i := range in {
		out[i] = convertResourceTo(in[i])
	}
	return out
}

func convertSourcerefTo(in SourceRef) compdesc.SourceRef {
	return compdesc.SourceRef{
		IdentitySelector: in.IdentitySelector.Copy(),
		Labels:           in.Labels.Copy(),
	}
}

func ConvertSourcerefsTo(in []SourceRef) []compdesc.SourceRef {
	if in == nil {
		return nil
	}
	out := make([]compdesc.SourceRef, len(in))
	for i := range in {
		out[i] = convertSourcerefTo(in[i])
	}
	return out
}

////////////////////////////////////////////////////////////////////////////////
// convert from internal version
////////////////////////////////////////////////////////////////////////////////

func (v *DescriptorVersion) ConvertFrom(in *compdesc.ComponentDescriptor) (compdesc.ComponentDescriptorVersion, error) {
	if in == nil {
		return nil, nil
	}
	out := &ComponentDescriptor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemaVersion,
			Kind:       Kind,
		},
		ObjectMeta:         *in.ObjectMeta.Copy(),
		RepositoryContexts: in.RepositoryContexts.Copy(),
		Spec: ComponentVersionSpec{
			Sources:    convertSourcesFrom(in.Sources),
			Resources:  convertResourcesFrom(in.Resources),
			References: convertReferencesFrom(in.References),
		},
		Signatures:    in.Signatures.Copy(),
		NestedDigests: in.NestedDigests.Copy(),
	}
	if err := out.Default(); err != nil {
		return nil, err
	}
	return out, nil
}

func convertReferenceFrom(in compdesc.ComponentReference) Reference {
	return Reference{
		ElementMeta:   convertElementmetaFrom(in.ElementMeta),
		ComponentName: in.ComponentName,
		Digest:        in.Digest.Copy(),
	}
}

func convertReferencesFrom(in []compdesc.ComponentReference) []Reference {
	if in == nil {
		return nil
	}
	out := make([]Reference, len(in))
	for i := range in {
		out[i] = convertReferenceFrom(in[i])
	}
	return out
}

func convertSourceFrom(in compdesc.Source) Source {
	acc, err := runtime.ToUnstructuredTypedObject(in.Access)
	if err != nil {
		compdesc.ThrowConversionError(err)
	}
	return Source{
		SourceMeta: SourceMeta{
			ElementMeta: convertElementmetaFrom(in.ElementMeta),
			Type:        in.Type,
		},
		Access: acc,
	}
}

func convertSourcesFrom(in compdesc.Sources) Sources {
	if in == nil {
		return nil
	}
	out := make(Sources, len(in))
	for i := range in {
		out[i] = convertSourceFrom(in[i])
	}
	return out
}

func convertElementmetaFrom(in compdesc.ElementMeta) ElementMeta {
	return ElementMeta{
		Name:          in.Name,
		Version:       in.Version,
		ExtraIdentity: in.ExtraIdentity.Copy(),
		Labels:        in.Labels.Copy(),
	}
}

func convertResourceFrom(in compdesc.Resource) Resource {
	acc, err := runtime.ToUnstructuredTypedObject(in.Access)
	if err != nil {
		compdesc.ThrowConversionError(err)
	}
	return Resource{
		ElementMeta: convertElementmetaFrom(in.ElementMeta),
		Type:        in.Type,
		Relation:    in.Relation,
		SourceRef:   convertSourcerefsFrom(in.SourceRef),
		Access:      acc,
		Digest:      in.Digest.Copy(),
	}
}

func convertResourcesFrom(in compdesc.Resources) Resources {
	if in == nil {
		return nil
	}
	out := make(Resources, len(in))
	for i := range in {
		out[i] = convertResourceFrom(in[i])
	}
	return out
}

func convertSourcerefFrom(in compdesc.SourceRef) SourceRef {
	return SourceRef{
		IdentitySelector: in.IdentitySelector.Copy(),
		Labels:           in.Labels.Copy(),
	}
}

func convertSourcerefsFrom(in []compdesc.SourceRef) []SourceRef {
	if in == nil {
		return nil
	}
	out := make([]SourceRef, len(in))
	for i := range in {
		out[i] = convertSourcerefFrom(in[i])
	}
	return out
}

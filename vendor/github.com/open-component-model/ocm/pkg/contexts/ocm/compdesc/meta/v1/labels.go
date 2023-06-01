// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/json"
	"regexp"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/open-component-model/ocm/pkg/errors"
)

// Label is a label that can be set on objects.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type Label struct {
	// Name is the unique name of the label.
	Name string `json:"name"`
	// Value is the json/yaml data of the label
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Value json.RawMessage `json:"value"`

	// Version is the optional specification version of the attribute value
	Version string `json:"version,omitempty"`
	// Signing describes whether the label should be included into the signature
	Signing bool `json:"signing,omitempty"`
}

// DeepCopyInto copies labels.
func (in *Label) DeepCopyInto(out *Label) {
	*out = *in
	out.Value = append(out.Value[:0:0], in.Value...)
}

var versionRegex = regexp.MustCompile("^v[0-9]+$")

func NewLabel(name string, value interface{}, opts ...LabelOption) (*Label, error) {
	var data []byte
	var err error
	var ok bool

	if data, ok = value.([]byte); ok {
		var v interface{}
		err = json.Unmarshal(data, &v)
		if err != nil {
			return nil, errors.ErrInvalid("label value", string(data), name)
		}
	} else {
		data, err = json.Marshal(value)
		if err != nil {
			return nil, errors.ErrInvalid("label value", "<object>", name)
		}
	}
	l := &Label{Name: name, Value: data}
	for _, o := range opts {
		if err := o.ApplyToLabel(l); err != nil {
			return nil, errors.Wrapf(err, "label %q", name)
		}
	}
	return l, nil
}

// Labels describe a list of labels
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type Labels []Label

// Get returns the label value with the given name as json string.
func (l Labels) Get(name string) ([]byte, bool) {
	for _, label := range l {
		if label.Name == name {
			return label.Value, true
		}
	}
	return nil, false
}

// GetValue returns the label value with the given name as parsed object.
func (l Labels) GetValue(name string, dest interface{}) (bool, error) {
	for _, label := range l {
		if label.Name == name {
			return true, json.Unmarshal(label.Value, dest)
		}
	}
	return false, nil
}

func (l *Labels) Set(name string, value interface{}, opts ...LabelOption) error {
	newLabel, err := NewLabel(name, value, opts...)
	if err != nil {
		return err
	}
	for _, label := range *l {
		if label.Name == name {
			label.Value = newLabel.Value
			return nil
		}
	}
	*l = append(*l, *newLabel)
	return nil
}

func (l *Labels) Remove(name string) bool {
	for i, label := range *l {
		if label.Name == name {
			*l = append((*l)[:i], (*l)[i+1:]...)
			return true
		}
	}
	return false
}

// AsMap return an unmarshalled map representation.
func (l *Labels) AsMap() map[string]interface{} {
	labels := map[string]interface{}{}
	if l != nil {
		for _, label := range *l {
			var m interface{}
			json.Unmarshal(label.Value, &m)
			labels[label.Name] = m
		}
	}
	return labels
}

// Copy copies labels.
func (l Labels) Copy() Labels {
	if l == nil {
		return nil
	}
	n := make(Labels, len(l))
	copy(n, l)
	return n
}

// ValidateLabels validates a list of labels.
func ValidateLabels(fldPath *field.Path, labels Labels) field.ErrorList {
	allErrs := field.ErrorList{}
	labelNames := make(map[string]struct{})
	for i, label := range labels {
		labelPath := fldPath.Index(i)
		if len(label.Name) == 0 {
			allErrs = append(allErrs, field.Required(labelPath.Child("name"), "must specify a name"))
			continue
		}

		if _, ok := labelNames[label.Name]; ok {
			allErrs = append(allErrs, field.Duplicate(labelPath, "duplicate label name"))
			continue
		}
		labelNames[label.Name] = struct{}{}
	}
	return allErrs
}

type LabelOption interface {
	ApplyToLabel(l *Label) error
}

type labelOptVersion struct {
	version string
}

var _ LabelOption = (*labelOptVersion)(nil)

func WithVersion(v string) LabelOption {
	return &labelOptVersion{v}
}

func CheckLabelVersion(v string) bool {
	return versionRegex.MatchString(v)
}

func (o labelOptVersion) ApplyToLabel(l *Label) error {
	if !CheckLabelVersion(o.version) {
		return errors.ErrInvalid("version", o.version)
	}
	l.Version = o.version
	return nil
}

type labelOptSigning struct {
	sign bool
}

var _ LabelOption = (*labelOptSigning)(nil)

func WithSigning(b ...bool) LabelOption {
	s := true
	for _, o := range b {
		s = o
	}
	return &labelOptSigning{s}
}

func (o *labelOptSigning) ApplyToLabel(l *Label) error {
	l.Signing = o.sign
	return nil
}

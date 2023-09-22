// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/json"
	"reflect"
	"regexp"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/equivalent"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/listformat"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	KIND_LABEL                 = "label"
	KIND_VALUE_MERGE_ALGORITHM = "label merge algorithm"
)

type MergeAlgorithmSpecification struct {
	// Algorithm optionally described the Merge algorithm used to
	// merge the label value during a transfer.
	Algorithm string `json:"algorithm"`
	// eConfig contains optional config for the merge algorithm.
	Config json.RawMessage `json:"config,omitempty"`
}

var _ listformat.DirectDescriptionSource = (*MergeAlgorithmSpecification)(nil)

func (s *MergeAlgorithmSpecification) Description() string {
	return s.Algorithm
}

func NewMergeAlgorithmSpecification(algo string, spec interface{}) (*MergeAlgorithmSpecification, error) {
	m, err := runtime.AsRawMessage(spec)
	if err != nil {
		return nil, err
	}
	return &MergeAlgorithmSpecification{
		Algorithm: algo,
		Config:    m,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////

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

	// MergeAlgorithm optionally describes the desired merge handling used to
	// merge the label value during a transfer.
	Merge *MergeAlgorithmSpecification `json:"merge,omitempty"`
}

// DeepCopyInto copies labels.
func (in *Label) DeepCopyInto(out *Label) {
	*out = *in
	out.Value = append(out.Value[:0:0], in.Value...)
}

// GetValue returns the label value with the given name as parsed object.
func (in *Label) GetValue(dest interface{}) error {
	return json.Unmarshal(in.Value, dest)
}

// SetValue sets the label value by marshalling the given object.
// A passed byte slice is validated to be valid json.
func (in *Label) SetValue(value interface{}) error {
	var v runtime.RawValue
	err := v.SetValue(value)
	if err != nil {
		return err
	}
	in.Value = v.RawMessage
	return nil
}

var versionRegex = regexp.MustCompile("^v[0-9]+$")

func NewLabel(name string, value interface{}, opts ...LabelOption) (*Label, error) {
	l := Label{Name: name}
	err := l.SetValue(value)
	if err != nil {
		return nil, err
	}

	for _, o := range opts {
		if err := o.ApplyToLabel(&l); err != nil {
			return nil, errors.Wrapf(err, "label %q", name)
		}
	}
	return &l, nil
}

// Labels describe a list of labels
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type Labels []Label

// GetIndex returns the index of the given label or -1 if not found.
func (l Labels) GetIndex(name string) int {
	for i, label := range l {
		if label.Name == name {
			return i
		}
	}
	return -1
}

// GetDef returns the label definition of the given label.
func (l Labels) GetDef(name string) *Label {
	for i, label := range l {
		if label.Name == name {
			return &l[i]
		}
	}
	return nil
}

// SetDef ets a label definition.
func (l *Labels) SetDef(name string, value *Label) {
	for i, label := range *l {
		if label.Name == name {
			(*l)[i] = *value
			return
		}
	}
	*l = append(*l, *value)
}

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
			return true, label.GetValue(dest)
		}
	}
	return false, nil
}

// Set sets or modifies a label including its meta data.
func (l *Labels) Set(name string, value interface{}, opts ...LabelOption) error {
	newLabel, err := NewLabel(name, value, opts...)
	if err != nil {
		return err
	}
	for i, label := range *l {
		if label.Name == name {
			(*l)[i] = *newLabel
			return nil
		}
	}
	*l = append(*l, *newLabel)
	return nil
}

// Set sets or modifies the label meta data.
func (l *Labels) SetOptions(name string, opts ...LabelOption) error {
	newLabel, err := NewLabel(name, nil, opts...)
	if err != nil {
		return err
	}
	for i, label := range *l {
		if label.Name == name {
			newLabel.Value = label.Value
			(*l)[i] = *newLabel
			return nil
		}
	}
	return errors.ErrNotFound(KIND_LABEL, name)
}

// SetValue sets or modifies the value of a label, the label metadata
// is not touched.
func (l *Labels) SetValue(name string, value interface{}) error {
	newLabel, err := NewLabel(name, value)
	if err != nil {
		return err
	}
	for i, label := range *l {
		if label.Name == name {
			(*l)[i].Value = newLabel.Value
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

func (l *Labels) Clear() {
	*l = nil
}

func (l Labels) Equivalent(o Labels) equivalent.EqualState {
	state := equivalent.StateEquivalent()

	for _, ol := range o {
		ll := l.GetDef(ol.Name)
		if ol.Signing {
			if ll == nil || !reflect.DeepEqual(&ol, ll) {
				state = state.NotLocalHashEqual()
			}
		} else {
			if ll != nil {
				if ll.Signing {
					state = state.NotLocalHashEqual()
				}
				if !reflect.DeepEqual(&ol, ll) {
					state = state.NotEquivalent()
				}
			} else {
				state = state.NotEquivalent()
			}
		}
	}
	for _, ll := range l {
		ol := o.GetDef(ll.Name)
		if ol == nil {
			if ll.Signing {
				state = state.NotLocalHashEqual()
			}
			state = state.NotEquivalent()
		}
	}
	return state
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

////////////////////////////////////////////////////////////////////////////////

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

////////////////////////////////////////////////////////////////////////////////

// LabelMergeHandlerConfig must be label merge handler config. but cannot be checked
// because of cyclic package dependencies.
type LabelMergeHandlerConfig interface{}

type labelOptMerge struct {
	cfg  json.RawMessage
	algo string
}

var _ LabelOption = (*labelOptMerge)(nil)

func WithMerging(algo string, cfg LabelMergeHandlerConfig) LabelOption {
	var data []byte
	if cfg != nil {
		var err error
		data, err = json.Marshal(cfg)
		if err != nil {
			return nil
		}
	}
	return &labelOptMerge{algo: algo, cfg: data}
}

func (o *labelOptMerge) ApplyToLabel(l *Label) error {
	if o.algo != "" || len(o.cfg) > 0 {
		l.Merge = &MergeAlgorithmSpecification{}
		if o.algo != "" {
			l.Merge.Algorithm = o.algo
		}
		if len(o.cfg) > 0 {
			l.Merge.Config = o.cfg
		}
	} else {
		l.Merge = nil
	}
	return nil
}

// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"reflect"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/equivalent"
	"github.com/open-component-model/ocm/pkg/errors"
)

const (
	SystemIdentityName    = "name"
	SystemIdentityVersion = "version"
)

// Metadata defines the metadata of the component descriptor.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type Metadata struct {
	// Version is the schema version of the component descriptor.
	Version string `json:"schemaVersion"`
}

// ProviderName describes the provider type of component in the origin's context.
// Defines whether the component is created by a third party or internally.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type ProviderName string

// ResourceRelation describes the type of a resource.
// Defines whether the component is created by a third party or internally.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type ResourceRelation string

const (
	// LocalRelation defines a internal relation
	// which describes a internally maintained resource in the origin's context.
	LocalRelation ResourceRelation = "local"
	// ExternalRelation defines a external relation
	// which describes a resource maintained by a third party vendor in the origin's context.
	ExternalRelation ResourceRelation = "external"
)

func ValidateRelation(fldPath *field.Path, relation ResourceRelation) *field.Error {
	if len(relation) == 0 {
		return field.Required(fldPath, "relation must be set")
	}
	if relation != LocalRelation && relation != ExternalRelation {
		return field.NotSupported(fldPath, relation, []string{string(LocalRelation), string(ExternalRelation)})
	}
	return nil
}

const (
	GROUP = "ocm.software"
	KIND  = "ComponentVersion"
)

// TypeMeta describes the schema of a descriptor.
type TypeMeta struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
}

// ObjectMeta defines the metadata of the component descriptor.
type ObjectMeta struct {
	// Name is the name of the component.
	Name string `json:"name"`
	// Version is the version of the component.
	Version string `json:"version"`
	// Labels describe additional properties of the component version
	Labels Labels `json:"labels,omitempty"`
	// Provider described the component provider
	Provider Provider `json:"provider"`
	// CreationTime is the creation time of component version
	// +optional
	CreationTime *Timestamp `json:"creationTime,omitempty"`
}

func (o *ObjectMeta) Equal(obj interface{}) bool {
	if e, ok := obj.(*ObjectMeta); ok {
		if o.Name == e.Name &&
			o.Version == e.Version &&
			reflect.DeepEqual(o.Provider, e.Provider) &&
			reflect.DeepEqual(o.Labels, e.Labels) {
			return true
		}
		// check Creation time ?
	}
	return false
}

func (o ObjectMeta) Equivalent(a ObjectMeta) equivalent.EqualState {
	state := equivalent.StateLocalHashEqual(o.Name == a.Name && o.Version == a.Version)
	return state.Apply(
		o.Provider.Equivalent(a.Provider),
		o.Labels.Equivalent(a.Labels),
	)
}

// GetName returns the name of the object.
func (o *ObjectMeta) GetName() string {
	return o.Name
}

// SetName sets the name of the object.
func (o *ObjectMeta) SetName(name string) {
	o.Name = name
}

// GetVersion returns the version of the object.
func (o ObjectMeta) GetVersion() string {
	return o.Version
}

// SetVersion sets the version of the object.
func (o *ObjectMeta) SetVersion(version string) {
	o.Version = version
}

// GetLabels returns the label of the object.
func (o ObjectMeta) GetLabels() Labels {
	return o.Labels
}

// SetLabels sets the labels of the object.
func (o *ObjectMeta) SetLabels(labels []Label) {
	o.Labels = labels
}

// GetName returns the name of the object.
func (o *ObjectMeta) Copy() *ObjectMeta {
	return &ObjectMeta{
		Name:     o.Name,
		Version:  o.Version,
		Labels:   o.Labels.Copy(),
		Provider: *o.Provider.Copy(),
	}
}

////////////////////////////////////////////////////////////////////////////////

// Provider describes the provider information of a component version.
type Provider struct {
	Name ProviderName `json:"name"`
	// Labels describe additional properties of provider
	Labels Labels `json:"labels,omitempty"`
}

// GetName returns the name of the provider.
func (o Provider) GetName() ProviderName {
	return o.Name
}

// SetName sets the name of the provider.
func (o *Provider) SetName(name ProviderName) {
	o.Name = name
}

// GetLabels returns the label of the provider.
func (o Provider) GetLabels() Labels {
	return o.Labels
}

// SetLabels sets the labels of the provider.
func (o *Provider) SetLabels(labels []Label) {
	o.Labels = labels
}

// Copy copies the provider info.
func (o *Provider) Copy() *Provider {
	return &Provider{
		Name:   o.Name,
		Labels: o.Labels.Copy(),
	}
}

func (o Provider) Equivalent(a Provider) equivalent.EqualState {
	state := equivalent.StateLocalHashEqual(o.Name == a.Name)
	return state.Apply(o.Labels.Equivalent(a.Labels))
}

////////////////////////////////////////////////////////////////////////////////

type _time = v1.Time

// Timestamp is time rounded to seconds.
// +k8s:deepcopy-gen=true
type Timestamp struct {
	_time `json:",inline"`
}

func NewTimestamp() Timestamp {
	return Timestamp{
		_time: v1.NewTime(time.Now().Round(time.Second)),
	}
}

func NewTimestampP() *Timestamp {
	return &Timestamp{
		_time: v1.NewTime(time.Now().Round(time.Second)),
	}
}

func NewTimestampFor(t time.Time) Timestamp {
	return Timestamp{
		_time: v1.NewTime(t.Round(time.Second)),
	}
}

func NewTimestampPFor(t time.Time) *Timestamp {
	return &Timestamp{
		_time: v1.NewTime(t.Round(time.Second)),
	}
}

// MarshalJSON implements the json.Marshaler interface.
// The time is a quoted string in RFC 3339 format, with sub-second precision added if present.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	if y := t.Year(); y < 0 || y >= 10000 {
		// RFC 3339 is clear that years are 4 digits exactly.
		// See golang.org/issue/4556#c15 for more discussion.
		return nil, errors.New("Time.MarshalJSON: year outside of range [0,9999]")
	}

	b := make([]byte, 0, len(time.RFC3339)+2)
	b = append(b, '"')
	b = t.AppendFormat(b, time.RFC3339)
	b = append(b, '"')
	return b, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// The time is expected to be a quoted string in RFC 3339 format.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	// Ignore null, like in the main JSON package.
	if string(data) == "null" {
		return nil
	}
	// Fractional seconds are handled implicitly by Parse.
	tt, err := time.Parse(`"`+time.RFC3339+`"`, string(data))
	*t = NewTimestampFor(tt)
	return err
}

func (t *Timestamp) Time() time.Time {
	return t._time.Time
}

func (t *Timestamp) Equal(o Timestamp) bool {
	return t._time.Equal(&o._time)
}

func (t *Timestamp) UTC() Timestamp {
	return NewTimestampFor(t._time.UTC())
}

func (t *Timestamp) Add(d time.Duration) Timestamp {
	return NewTimestampFor(t._time.Add(d))
}

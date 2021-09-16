// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubernetescheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	configinstall "github.com/gardener/landscaper/apis/config/install"
	coreinstall "github.com/gardener/landscaper/apis/core/install"
)

var (
	// LandscaperScheme is the scheme used in the landscaper cluster.
	LandscaperScheme = runtime.NewScheme()
	// ConfigScheme is the scheme used for configurations.
	ConfigScheme = runtime.NewScheme()
	// Scheme is the scheme used in the landscaper cluster and for configurations.
	Scheme = runtime.NewScheme()
	// Decoder is a decoder for the landscaper and landscaper config scheme.
	Decoder runtime.Decoder
)

func init() {
	coreinstall.Install(LandscaperScheme)
	utilruntime.Must(kubernetescheme.AddToScheme(LandscaperScheme))
	configinstall.Install(ConfigScheme)

	coreinstall.Install(Scheme)
	utilruntime.Must(kubernetescheme.AddToScheme(Scheme))
	configinstall.Install(Scheme)
	Decoder = NewDecoder(Scheme)
}

// NewDecoder creates a new universal decoder.
func NewDecoder(scheme *runtime.Scheme) runtime.Decoder {
	return &UniversalInternalVersionDecoder{
		scheme:  scheme,
		decoder: serializer.NewCodecFactory(scheme).UniversalDecoder(),
	}
}

// UniversalInternalVersionDecoder is a decoder that can decode kubernetes-like versioned resources.
// The universal decoder does automatically use the internal type to perform conversion
// which is a missing feature in the kubernetes codec factory.
type UniversalInternalVersionDecoder struct {
	scheme  *runtime.Scheme
	decoder runtime.Decoder
}

var _ runtime.Decoder = &UniversalInternalVersionDecoder{}

// todo: implement shortcut for actual.gvk == into.gvk
func (d *UniversalInternalVersionDecoder) Decode(data []byte, defaults *schema.GroupVersionKind, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	if into == nil && defaults == nil {
		return nil, nil, errors.New("no default group version kind nor a object are defined")
	}

	if into == nil && defaults != nil {
		var err error
		into, err = d.scheme.New(*defaults)
		if err != nil {
			return nil, nil, err
		}
	}

	if _, isUnstructured := into.(*unstructured.Unstructured); isUnstructured {
		return d.decoder.Decode(data, defaults, into)
	}

	gvk, err := apiutil.GVKForObject(into, d.scheme)
	if err != nil {
		return nil, nil, err
	}

	internalGVK := gvk
	internalGVK.Version = runtime.APIVersionInternal
	if !d.scheme.Recognizes(internalGVK) {
		return d.decoder.Decode(data, defaults, into)
	}

	// decode to internal type
	internalObj, _, err := d.decoder.Decode(data, &internalGVK, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to decode to internal type: %w", err)
	}

	if err := d.scheme.Convert(internalObj, into, nil); err != nil {
		return nil, nil, err
	}

	return into, &gvk, nil
}

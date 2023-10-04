// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

const (
	T_OCIARTIFACT = "artifact"
	T_OCIINDEX    = "index"
	T_OCIMANIFEST = "manifest"
)

type ociArtifact struct {
	base
	kind    string
	artfunc func(a oci.ArtifactAccess) error
	ns      cpi.NamespaceAccess
	cpi.ArtifactAccess
	tags  []string
	annos *map[string]string
}

func (r *ociArtifact) setAnno(name, value string) {
	if r.annos == nil {
		return
	}
	if *r.annos == nil {
		*r.annos = map[string]string{}
	}
	(*r.annos)[name] = value
}

func (r *ociArtifact) Type() string {
	return r.kind
}

func (r *ociArtifact) Set() {
	r.Builder.oci_nsacc = r.ns
	r.Builder.oci_artacc = r.ArtifactAccess
	r.Builder.oci_tags = &r.tags
	r.Builder.oci_annofunc = r.setAnno

	if r.ns != nil {
		r.Builder.oci_artfunc = r.addArtifact
	}
}

func (r *ociArtifact) Close() error {
	err := r.ArtifactAccess.Close()
	if err != nil {
		return err
	}
	blob, err := r.Builder.oci_nsacc.AddArtifact(r.ArtifactAccess, r.tags...)
	if err == nil && r.artfunc != nil {
		err = r.artfunc(r.ArtifactAccess)
	}
	if err == nil {
		r.result = artdesc.DefaultBlobDescriptor(blob)
	}
	return err
}

func (r *ociArtifact) addArtifact(a oci.ArtifactAccess) error {
	_, err := r.ArtifactAccess.AddArtifact(a, nil)
	return err
}

////////////////////////////////////////////////////////////////////////////////

func (b *Builder) artifact(tag string, ns cpi.NamespaceAccess, t func(access oci.ArtifactAccess) (string, *map[string]string, error), f ...func()) *artdesc.Descriptor {
	var k string
	var annos *map[string]string
	var err error
	var v oci.ArtifactAccess

	b.expect(b.oci_nsacc, T_OCINAMESPACE)

	if tag != "" {
		v, err = b.oci_nsacc.GetArtifact(tag)
		if err != nil {
			if errors.IsErrNotFound(err) {
				if b, _ := artdesc.IsDigest(tag); !b {
					err = nil
				}
			}
			if errors.IsErrUnknown(err) {
				if b, _ := artdesc.IsDigest(tag); !b {
					err = nil
				}
			}
		}
		b.failOn(err, 1)
	}
	b.oci_cleanuplayers = true
	if v == nil {
		v, err = b.oci_nsacc.NewArtifact()
	}
	if v != nil {
		k, annos, err = t(v)
	}
	b.failOn(err, 1)
	tags := []string{}
	if tag != "" {
		tags = append(tags, tag)
	}
	r := b.configure(&ociArtifact{ArtifactAccess: v, kind: k, tags: tags, ns: ns, artfunc: b.oci_artfunc, annos: annos}, f, 1)
	if r == nil {
		return nil
	}
	return r.(*artdesc.Descriptor)
}

func (b *Builder) Index(tag string, f ...func()) *artdesc.Descriptor {
	return b.artifact(tag, b.oci_nsacc, func(a oci.ArtifactAccess) (string, *map[string]string, error) {
		if m := a.IndexAccess(); m != nil {
			return T_OCIINDEX, &m.GetDescriptor().Annotations, nil
		}
		return "", nil, errors.Newf("artifact is manifest")
	}, f...)
}

func (b *Builder) Manifest(tag string, f ...func()) *artdesc.Descriptor {
	return b.artifact(tag, nil, func(a oci.ArtifactAccess) (string, *map[string]string, error) {
		if m := a.ManifestAccess(); m != nil {
			return T_OCIMANIFEST, &m.GetDescriptor().Annotations, nil
		}
		return "", nil, errors.Newf("artifact is index")
	}, f...)
}

func (b *Builder) Artifact(desc *artdesc.Descriptor) {
	b.expect(b.oci_artacc, T_OCIINDEX, func() bool { return b.oci_artacc.IsIndex() })

	m := b.oci_artacc.IndexAccess().GetDescriptor()
	m.Manifests = append(m.Manifests, *desc)
}

// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/signing"
	"github.com/open-component-model/ocm/pkg/signing/signutils"
	"github.com/open-component-model/ocm/pkg/utils"
)

type RootContextInfo struct {
	CtxKey     common.NameVersion
	Sign       bool
	DigestType ocm.DigesterType
	Hasher     signing.Hasher
	In         map[common.NameVersion]*metav1.NestedComponentDigests
	Out        map[common.NameVersion]*metav1.NestedComponentDigests
}

func (dc *RootContextInfo) GetPreset(nv common.NameVersion) *metav1.NestedComponentDigests {
	if p := dc.Out[nv]; p != nil {
		return p
	}
	if p := dc.In[nv]; p != nil {
		return p
	}
	return nil
}

type DigestContext struct {
	*RootContextInfo

	Key        common.NameVersion
	Parent     *DigestContext
	Descriptor *compdesc.ComponentDescriptor
	Digest     *metav1.DigestSpec
	Signed     bool
	Source     common.NameVersion
	Refs       map[common.NameVersion]*metav1.DigestSpec
}

func NewDigestContext(cd *compdesc.ComponentDescriptor, parent *DigestContext) *DigestContext {
	var root *RootContextInfo

	key := common.VersionedElementKey(cd)
	if parent == nil {
		root = &RootContextInfo{
			CtxKey: key,
			Out:    map[common.NameVersion]*metav1.NestedComponentDigests{},
			In:     map[common.NameVersion]*metav1.NestedComponentDigests{},
		}
		for _, c := range cd.NestedDigests {
			nv := common.NewNameVersion(c.Name, c.Version)
			digs := metav1.NestedComponentDigests{
				Name:    nv.GetName(),
				Version: nv.GetVersion(),
			}
			for _, r := range c.Resources {
				digs.Resources = append(digs.Resources, *r.Copy())
			}
			root.In[nv] = &digs
		}
		digs, all := GetDigests(cd)
		if len(root.In) == 0 && all {
			if len(digs.Resources) > 0 {
				root.In[key] = digs
			}
		}
	} else {
		root = parent.RootContextInfo
	}

	return &DigestContext{
		RootContextInfo: root,
		Key:             key,
		Parent:          parent,
		Descriptor:      cd,
		Refs:            map[common.NameVersion]*metav1.DigestSpec{},
	}
}

func GetDigests(cd *compdesc.ComponentDescriptor) (*metav1.NestedComponentDigests, bool) {
	all := true
	digs := &metav1.NestedComponentDigests{
		Name:    cd.GetName(),
		Version: cd.GetVersion(),
	}
	for _, r := range cd.Resources {
		if !compdesc.IsNoneAccess(r.Access) {
			if r.Digest != nil {
				ad := ArtefactDigest(&r)
				digs.Resources = append(digs.Resources, ad)
			} else {
				all = false
			}
		}
	}
	return digs, all
}

func (dc *DigestContext) IsRoot() bool {
	return dc.CtxKey == dc.Key
}

func (dc *DigestContext) GetDigests() metav1.NestedDigests {
	var result metav1.NestedDigests
	keys := utils.SortedMapKeys(dc.Refs)
	for _, k := range keys {
		result = append(result, *dc.Out[k])
	}
	return result
}

func (dc *DigestContext) Propagate(d *metav1.DigestSpec) error {
	digs, all := GetDigests(dc.Descriptor)
	if !all {
		return fmt.Errorf("not all digests calculated")
	}
	digs.Digest = d
	dc.Digest = d
	preset := dc.GetPreset(dc.Key)

	if preset != nil {
		if !digs.Resources.Match(preset.Resources) {
			return fmt.Errorf("digest set for %s does not match", dc.Key)
		}
		digs = preset
	}
	dc.Out[dc.Key] = digs
	if dc.Parent != nil {
		for nv, d := range dc.Refs {
			dc.Parent.Refs[nv] = d
		}
	}
	return nil
}

func (dc *DigestContext) Use(ctx *DigestContext) error {
	for nv, digs := range ctx.Out {
		if cur := dc.Out[nv]; cur != nil {
			if !cur.Resources.Match(digs.Resources) {
				return fmt.Errorf("digest set mismatch")
			}
		} else {
			dc.Out[nv] = digs
		}
	}
	for nv, d := range ctx.Refs {
		dc.Refs[nv] = d
	}
	dc.Digest = ctx.Digest
	dc.Descriptor = ctx.Descriptor
	dc.Signed = ctx.Signed
	return nil
}

func (dc *DigestContext) ValidFor(ctx *DigestContext) bool {
	for nv, digs := range dc.Out {
		if preset := ctx.GetPreset(nv); preset != nil {
			if !preset.Resources.Match(digs.Resources) {
				return false
			}
		}
	}
	for nv, digs := range dc.In {
		if dc.Out[nv] == nil {
			if preset := ctx.GetPreset(nv); preset != nil {
				if !preset.Resources.Match(digs.Resources) {
					return false
				}
			}
		}
	}
	return true
}

func (dc *DigestContext) determineSignatureInfo(state WalkingState, cv ocm.ComponentVersionAccess, opts *Options) (*Options, error) {
	if opts.SignatureName() != "" {
		// determine digester type
		var found bool
		for _, sig := range dc.Descriptor.Signatures {
			if sig.Name == opts.SignatureName() {
				dc.DigestType = DigesterType(&sig.Digest)
				found = true
				break
			}
		}
		if !found {
			return nil, errors.ErrNotFound(compdesc.KIND_SIGNATURE, opts.SignatureName())
		}
	}

	var signatures []string
	// setup verifiable signatures, the first one we
	// have a public key for determines the
	// digester type we can commonly check.
	for _, sig := range dc.Descriptor.Signatures {
		st := DigesterType(&sig.Digest)
		//nolint: gocritic //yes
		if opts.Keyless {
			if dc.DigestType.IsInitial() {
				dc.DigestType = st
			}
			if dc.DigestType == st {
				signatures = append(signatures, sig.Name)
			}
		} else if opts.PublicKey(sig.Name) != nil {
			if dc.DigestType.IsInitial() {
				dc.DigestType = st
			}
			if dc.DigestType == st {
				signatures = append(signatures, sig.Name)
			} else {
				opts.Printer.Printf("Warning: digest type %s for signature %q in %s does not match (signature ignored)\n", dc.DigestType.String(), sig.Name, state.History)
			}
		} else {
			if opts.SignatureConfigured(sig.Name) || opts.SignatureName() == "" {
				i := cv.GetDescriptor().GetSignatureIndex(sig.Name)
				if i < 0 {
					return nil, errors.ErrNotFound(compdesc.KIND_SIGNATURE, sig.Name)
				}
				s := cv.GetDescriptor().Signatures[i]
				if s.Signature.MediaType == signutils.MediaTypePEM {
					_, _, _, err := signutils.GetSignatureFromPem([]byte(s.Signature.Value))
					if err != nil {
						return nil, errors.Wrapf(err, "cannot decode signature PEM for %q", sig.Name)
					}
					signatures = append(signatures, sig.Name)
					dc.DigestType = DigesterType(&sig.Digest)
				} else {
					if opts.SignatureName() != "" {
						return nil, errors.ErrNotFound(compdesc.KIND_PUBLIC_KEY, sig.Name)
					}
				}
			}
		}
	}
	opts = opts.Dup()
	opts.SignatureNames = signatures
	if len(signatures) == 0 {
		return nil, errors.Newf("no signature found")
	}
	return opts, nil
}

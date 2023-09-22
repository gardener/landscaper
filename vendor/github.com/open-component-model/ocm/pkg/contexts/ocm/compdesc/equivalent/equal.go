// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package equivalent

// EqualState describes the equivalence state of elements
// of a component version. For a complete component version
// the equivalence state describes the local component version
// content, not the state of the complete graph spanned by
// component references.
type EqualState struct {
	equivalent   bool
	hashEqual    bool
	contentequal bool
	detectable   bool
}

// IsEquivalent returns true if everything besides the access methods
// is identical, including the artifact digests.
func (s EqualState) IsEquivalent() bool {
	return s.equivalent
}

// IsHashEqual returns true if the signature relevant parts
// are identical, including the artifact digests.
func (s EqualState) IsHashEqual() bool {
	return s.hashEqual && s.detectable && s.contentequal
}

// IsLocalHashEqual returns true if the signature relevant parts
// excluding the artifact digests are identical.
func (s EqualState) IsLocalHashEqual() bool {
	return s.hashEqual
}

// IsArtifactEqual returns true if the signature relevant parts
// artifact digests are identical.
func (s EqualState) IsArtifactEqual() bool {
	return s.detectable && s.contentequal
}

// IsArtifactEqual returns true if the signature relevant
// artifact digests are all known on both sides.
func (s EqualState) IsArtifactDetectable() bool {
	return s.detectable
}

func (s EqualState) NotLocalHashEqual(b ...bool) EqualState {
	if len(b) == 0 {
		s.equivalent = false
		s.hashEqual = false
	}
	for _, ok := range b {
		if !ok {
			s.equivalent = false
			s.hashEqual = false
			break
		}
	}
	return s
}

func (s EqualState) NotEquivalent() EqualState {
	s.equivalent = false
	return s
}

func (s EqualState) Apply(states ...EqualState) EqualState {
	for _, o := range states {
		s.equivalent = s.equivalent && o.equivalent
		s.hashEqual = s.hashEqual && o.hashEqual
		s.contentequal = s.contentequal && o.contentequal
		s.detectable = s.detectable && o.detectable
	}
	return s
}

func StateEquivalent() EqualState {
	return EqualState{true, true, true, true}
}

func StateLocalHashEqual(ok bool) EqualState {
	return EqualState{ok, ok, true, true}
}

func StateNotLocalHashEqual() EqualState {
	return EqualState{false, false, true, true}
}

func StateNotArtifactEqual(detect bool) EqualState {
	return EqualState{false, true, false, detect}
}

func StateNotEquivalent() EqualState {
	return EqualState{false, true, true, true}
}

func StateNotDetectable() EqualState {
	return EqualState{false, true, false, false}
}

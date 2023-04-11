// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1

// ResourceReference describes re resource identity relative to an (aggregation)
// component version.
type ResourceReference struct {
	Resource      Identity   `json:"resource"`
	ReferencePath []Identity `json:"referencePath,omitempty"`
}

func NewResourceRef(id Identity) ResourceReference {
	return ResourceReference{Resource: id}
}

func NewNestedResourceRef(id Identity, path []Identity) ResourceReference {
	return ResourceReference{Resource: id, ReferencePath: path}
}

func (r ResourceReference) String() string {
	s := r.Resource.String()

	for i := 1; i <= len(r.ReferencePath); i++ {
		s += "@" + r.ReferencePath[len(r.ReferencePath)-i].String()
	}
	return s
}

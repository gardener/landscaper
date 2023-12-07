// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package descriptor

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/generics"
)

type RepositoryContext struct {
	ContextType    string `json:"contextType"`
	RepositoryType string `json:"repositoryType"`
}

func (k RepositoryContext) HasRepo() bool {
	return k.ContextType != "" || k.RepositoryType != ""
}

func (k RepositoryContext) IsValid() bool {
	return k.HasRepo() || (k.ContextType == "" && k.RepositoryType == "")
}

func (k RepositoryContext) String() string {
	if k.HasRepo() {
		return fmt.Sprintf("[%s:%s]", k.ContextType, k.RepositoryType)
	}
	return ""
}

func (k RepositoryContext) Describe() string {
	if k.HasRepo() {
		return fmt.Sprintf("Default Repository Upload:\n  Context Type:   %s\n  RepositoryType: %s", k.ContextType, k.RepositoryType)
	}
	return ""
}

type ArtifactContext struct {
	ArtifactType string `json:"artifactType"`
	MediaType    string `json:"mediaType"`
}

func (k ArtifactContext) IsValid() bool {
	return k.ArtifactType != "" || k.MediaType != ""
}

func (k ArtifactContext) GetArtifactType() string {
	return k.ArtifactType
}

func (k ArtifactContext) GetMediaType() string {
	return k.MediaType
}

func (k ArtifactContext) String() string {
	return fmt.Sprintf("%s:%s", k.ArtifactType, k.MediaType)
}

func (k ArtifactContext) Describe() string {
	return fmt.Sprintf("Artifact Type: %s\nMedia Type   :%s", k.ArtifactType, k.MediaType)
}

func (k ArtifactContext) SetArtifact(arttype, mediatype string) ArtifactContext {
	k.ArtifactType = arttype
	k.MediaType = mediatype
	return k
}

type UploaderKey struct {
	RepositoryContext `json:",inline"`
	ArtifactContext   `json:",inline"`
}

func (k UploaderKey) IsValid() bool {
	return k.ArtifactContext.IsValid() && k.RepositoryContext.IsValid()
}

func (k UploaderKey) String() string {
	return fmt.Sprintf("%s%s", k.ArtifactContext.String(), k.RepositoryContext.String())
}

func (k UploaderKey) Describe() string {
	return fmt.Sprintf("%s%s", k.ArtifactContext.Describe(), k.RepositoryContext.Describe())
}

func (k UploaderKey) SetArtifact(arttype, mediatype string) UploaderKey {
	k.ArtifactType = arttype
	k.MediaType = mediatype
	return k
}

func (k UploaderKey) SetRepo(contexttype, repotype string) UploaderKey {
	k.ContextType = contexttype
	k.RepositoryType = repotype
	return k
}

type UploaderKeySet = generics.Set[UploaderKey]

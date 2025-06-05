// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v2

// KnownAccessTypes contains all known access serializer
var KnownAccessTypes = KnownTypes{
	OCIRegistryType:          DefaultJSONTypedObjectCodec,
	OCIBlobType:              DefaultJSONTypedObjectCodec,
	RelativeOciReferenceType: DefaultJSONTypedObjectCodec,
	GitHubAccessType:         DefaultJSONTypedObjectCodec,
	WebType:                  DefaultJSONTypedObjectCodec,
	LocalFilesystemBlobType:  DefaultJSONTypedObjectCodec,
}

// OCIRegistryType is the access type of a oci registry.
const OCIRegistryType = "ociRegistry"

// OCIRegistryAccess describes the access for a oci registry.
type OCIRegistryAccess struct {
	ObjectType `json:",inline"`

	// ImageReference is the actual reference to the oci image repository and tag.
	ImageReference string `json:"imageReference"`
}

// NewOCIRegistryAccess creates a new OCIRegistryAccess accessor
func NewOCIRegistryAccess(ref string) *OCIRegistryAccess {
	return &OCIRegistryAccess{
		ObjectType: ObjectType{
			Type: OCIRegistryType,
		},
		ImageReference: ref,
	}
}

func (a *OCIRegistryAccess) GetType() string {
	return OCIRegistryType
}

// RelativeOciReferenceType is the access type of a relative oci reference.
const RelativeOciReferenceType = "relativeOciReference"

// RelativeOciAccess describes the access for a relative oci reference.
type RelativeOciAccess struct {
	ObjectType `json:",inline"`

	// Reference is the relative reference to the oci image repository and tag.
	Reference string `json:"reference"`
}

// NewRelativeOciAccess creates a new RelativeOciAccess accessor
func NewRelativeOciAccess(ref string) *RelativeOciAccess {
	return &RelativeOciAccess{
		ObjectType: ObjectType{
			Type: RelativeOciReferenceType,
		},
		Reference: ref,
	}
}

func (_ *RelativeOciAccess) GetType() string {
	return RelativeOciReferenceType
}

// OCIBlobType is the access type of a oci blob in a manifest.
const OCIBlobType = "ociBlob"

// OCIBlobAccess describes the access for a oci registry.
type OCIBlobAccess struct {
	ObjectType `json:",inline"`

	// Reference is the oci reference to the manifest
	Reference string `json:"ref"`

	// MediaType is the media type of the object this schema refers to.
	MediaType string `json:"mediaType,omitempty"`

	// Digest is the digest of the targeted content.
	Digest string `json:"digest"`

	// Size specifies the size in bytes of the blob.
	Size int64 `json:"size"`
}

// NewOCIBlobAccess creates a new OCIBlob accessor
func NewOCIBlobAccess(ref, mediaType, digest string, size int64) *OCIBlobAccess {
	return &OCIBlobAccess{
		ObjectType: ObjectType{
			Type: OCIBlobType,
		},
		Reference: ref,
		MediaType: mediaType,
		Digest:    digest,
		Size:      size,
	}
}

func (_ *OCIBlobAccess) GetType() string {
	return OCIBlobType
}

// LocalOCIBlobType is the access type of a oci blob in the current component descriptor manifest.
const LocalOCIBlobType = "localOciBlob"

// NewLocalOCIBlobAccess creates a new LocalOCIBlob accessor
func NewLocalOCIBlobAccess(digest string) *LocalOCIBlobAccess {
	return &LocalOCIBlobAccess{
		ObjectType: ObjectType{
			Type: LocalOCIBlobType,
		},
		Digest: digest,
	}
}

// LocalOCIBlobAccess describes the access for a blob that is stored in the component descriptors oci manifest.
type LocalOCIBlobAccess struct {
	ObjectType `json:",inline"`
	// Digest is the digest of the targeted content.
	Digest string `json:"digest"`
}

func (_ *LocalOCIBlobAccess) GetType() string {
	return LocalOCIBlobType
}

// LocalFilesystemBlobType is the access type of a blob in a local filesystem.
const LocalFilesystemBlobType = "localFilesystemBlob"

// NewLocalFilesystemBlobAccess creates a new localFilesystemBlob accessor.
func NewLocalFilesystemBlobAccess(path string, mediaType string) *LocalFilesystemBlobAccess {
	return &LocalFilesystemBlobAccess{
		ObjectType: ObjectType{
			Type: LocalFilesystemBlobType,
		},
		Filename:  path,
		MediaType: mediaType,
	}
}

// LocalFilesystemBlobAccess describes the access for a blob on the filesystem.
type LocalFilesystemBlobAccess struct {
	ObjectType `json:",inline"`
	// Filename is the name of the blob in the local filesystem.
	// The blob is expected to be at <fs-root>/blobs/<name>
	Filename string `json:"filename"`
	// MediaType is the media type of the object this filename refers to.
	MediaType string `json:"mediaType,omitempty"`
}

func (_ *LocalFilesystemBlobAccess) GetType() string {
	return LocalFilesystemBlobType
}

// WebType is the type of a web component
const WebType = "web"

// Web describes a web resource access that can be fetched via http GET request.
type Web struct {
	ObjectType `json:",inline"`

	// URL is the http get accessible url resource.
	URL string `json:"url"`
}

// NewWebAccess creates a new Web accessor
func NewWebAccess(url string) *Web {
	return &Web{
		ObjectType: ObjectType{
			Type: OCIBlobType,
		},
		URL: url,
	}
}

func (_ *Web) GetType() string {
	return WebType
}

// GitHubAccessType is the type of a git object.
const GitHubAccessType = "github"

// GitHubAccess describes a github repository resource access.
type GitHubAccess struct {
	ObjectType `json:",inline"`

	// RepoURL is the url pointing to the remote repository.
	RepoURL string `json:"repoUrl"`
	// Ref describes the git reference.
	Ref string `json:"ref"`
	// Commit describes the git commit of the referenced repository.
	// +optional
	Commit string `json:"commit,omitempty"`
}

// NewGitHubAccess creates a new Web accessor
func NewGitHubAccess(url, ref, commit string) *GitHubAccess {
	return &GitHubAccess{
		ObjectType: ObjectType{
			Type: GitHubAccessType,
		},
		RepoURL: url,
		Ref:     ref,
		Commit:  commit,
	}
}

func (a GitHubAccess) GetType() string {
	return GitHubAccessType
}

// S3AccessType is the type of a s3 access.
const S3AccessType = "s3"

// S3AccessType describes a s3 resource access.
type S3Access struct {
	ObjectType `json:",inline"`

	// BucketName is the name of the s3 bucket.
	BucketName string `json:"bucketName"`
	// ObjectKey describes the referenced object.
	ObjectKey string `json:"objectKey"`
}

// NewS3Access creates a new s3 accessor
func NewS3Access(bucketName, objectKey string) *S3Access {
	return &S3Access{
		ObjectType: ObjectType{
			Type: S3AccessType,
		},
		BucketName: bucketName,
		ObjectKey:  objectKey,
	}
}

func (a S3Access) GetType() string {
	return S3AccessType
}

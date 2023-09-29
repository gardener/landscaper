// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package options

// HintOption.
var HintOption = RegisterOption(NewStringOptionType("hint", "(repository) hint for local artifacts"))

// MediatypeOption.
var MediatypeOption = RegisterOption(NewStringOptionType("mediaType", "media type for artifact blob representation"))

// SizeOption.
var SizeOption = RegisterOption(NewIntOptionType("size", "blob size"))

// DigestOption.
var DigestOption = RegisterOption(NewStringOptionType("digest", "blob digest"))

// ReferenceOption.
var ReferenceOption = RegisterOption(NewStringOptionType("reference", "reference name"))

// PackageOption.
var PackageOption = RegisterOption(NewStringOptionType("accessPackage", "package or object name"))

// RepositoryOption.
var RepositoryOption = RegisterOption(NewStringOptionType("accessRepository", "repository URL"))

// RegistryOption.
var RegistryOption = RegisterOption(NewStringOptionType("accessRegistry", "registry base URL"))

// HostnameOption.
var HostnameOption = RegisterOption(NewStringOptionType("accessHostname", "hostname used for access"))

// CommitOption.
var CommitOption = RegisterOption(NewStringOptionType("commit", "git commit id"))

// GlobalAccessOption.
var GlobalAccessOption = RegisterOption(NewValueMapYAMLOptionType("globalAccess", "access specification for global access"))

// RegionOption.
var RegionOption = RegisterOption(NewStringOptionType("region", "region name"))

// BucketOption.
var BucketOption = RegisterOption(NewStringOptionType("bucket", "bucket name"))

// VersionOption.
var VersionOption = RegisterOption(NewStringOptionType("accessVersion", "version for access specification"))

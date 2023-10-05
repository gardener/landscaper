// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	. "github.com/open-component-model/ocm/pkg/regex"
)

const (
	// RepositorySeparatorChar is the separator character used to separate
	// repository name components.
	RepositorySeparatorChar = '/'

	// RepositorySeparator is the separator string used to separate
	// repository name components.
	RepositorySeparator = string(RepositorySeparatorChar)

	TagSeparatorChar = ":"
	TagSeparator     = string(TagSeparatorChar)

	DigestSeparatorChar = "@"
	DigestSeparator     = string(DigestSeparatorChar)
)

var (
	// TypeRegexp describes a type name for a repository.
	TypeRegexp = Optional(Identifier)

	// CapturedSchemeRegexp matches an optional scheme.
	CapturedSchemeRegexp = Sequence(Capture(Match(`[a-z]+`)), Match("://"))

	// AnchoredRegistryRegexp parses a uniform repository spec.
	AnchoredRegistryRegexp = Anchored(
		Optional(Capture(TypeRegexp), Literal("::")),
		Optional(CapturedSchemeRegexp),
		Capture(DomainPortRegexp),
	)

	// AnchoredGenericRegistryRegexp describes a CTF reference.
	AnchoredGenericRegistryRegexp = Anchored(
		Optional(Capture(TypeRegexp), Literal("::")),
		Capture(Match(".*")),
	)

	// RepositorySeparatorRegexp is the separator used to separate
	// repository name components.
	RepositorySeparatorRegexp = Literal(RepositorySeparator)

	// alphaNumericRegexp defines the alpha numeric atom, typically a
	// component of names. This only allows lower case characters and digits.
	AlphaNumericRegexp = Match(`[a-z0-9]+`)

	// separatorRegexp defines the separators allowed to be embedded in name
	// components. This allow one period, one or two underscore and multiple
	// dashes.
	separatorRegexp = Match(`(?:[._]|__|[-]*)`)

	// dockerOrgSeparatorRegexp defines the separators allowed to be
	// embedded in a docker organization name.
	// https://docs.docker.com/docker-hub/repos/
	dockerOrgSeparatorRegexp = Match(`(?:_|__|[-]*)`)

	// DockerOrgRegexp restricts registry path component names to start
	// with at least one letter or number, with following parts able to be
	// separated by one or two underscore and multiple dashes.
	DockerOrgRegexp = Sequence(
		AlphaNumericRegexp,
		Optional(Repeated(dockerOrgSeparatorRegexp, AlphaNumericRegexp)))

	// NameComponentRegexp restricts registry path component names to start
	// with at least one letter or number, with following parts able to be
	// separated by one period, one or two underscore and multiple dashes.
	NameComponentRegexp = Sequence(
		AlphaNumericRegexp,
		Optional(Repeated(separatorRegexp, AlphaNumericRegexp)))

	// DomainComponentRegexp restricts the registry domain component of a
	// repository name to start with a component as defined by DomainPortRegexp
	// and followed by an optional port.
	DomainComponentRegexp = Match(`(?:[a-zA-Z0-9]|(?:[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9]))`)

	IPRegexp = Sequence(Match("[0-9]+"), Literal(`.`), Match("[0-9]+"), Literal(`.`), Match("[0-9]+"), Literal(`.`), Match("[0-9]+"))

	// DomainRegexp defines the structure of potential domain components
	// that may be part of image names. This is purposely a subset of what is
	// allowed by DNS to ensure backwards compatibility with Docker image
	// names.
	DomainRegexp = Sequence(
		DomainComponentRegexp, Literal(`.`), DomainComponentRegexp,
		Optional(Repeated(Literal(`.`), DomainComponentRegexp)))

	// DomainPortRegexp defines the structure of potential domain components
	// that may be part of image names. This is purposely a subset of what is
	// allowed by DNS to ensure backwards compatibility with Docker image
	// names followed by an optional port part.
	DomainPortRegexp = Sequence(
		DomainRegexp,
		Optional(Literal(`:`), Match(`[0-9]+`)))

	// HostPortRegexp describes a non-DNS simple hostname like localhost.
	HostPortRegexp = Sequence(
		Or(DomainComponentRegexp, IPRegexp),
		Optional(Literal(`:`), Match(`[0-9]+`)))

	PathRegexp = Sequence(
		Optional(Literal("/")),
		Match(`[a-zA-Z0-9-_.]+(?:/[a-zA-Z0-9-_.]+)+`))

	PathPortRegexp = Sequence(
		PathRegexp,
		Optional(Literal(`:`), Match(`[0-9]+`)))

	// TagRegexp matches valid tag names. From docker/docker:graph/tags.go.
	TagRegexp = Match(`[\w][\w.-]{0,127}`)

	// AnchoredTagRegexp matches valid tag names, anchored at the start and
	// end of the matched string.
	AnchoredTagRegexp = Anchored(TagRegexp)

	// DigestRegexp matches valid digests.
	DigestRegexp = Match(`[A-Za-z][A-Za-z0-9]*(?:[-_+.][A-Za-z][A-Za-z0-9]*)*[:][[:xdigit:]]{32,}`)

	// RepositoryRegexp is the format of a repository ppart of references.
	RepositoryRegexp = Sequence(
		NameComponentRegexp,
		Optional(Repeated(RepositorySeparatorRegexp, NameComponentRegexp)))

	// AnchoredRepositoryRegexp matches a plain OCI repository name.
	AnchoredRepositoryRegexp = Anchored(RepositoryRegexp)

	// AnchoredNameRegexp is used to parse a name value, capturing the
	// domain and trailing components.
	AnchoredNameRegexp = Anchored(
		Optional(Capture(DomainPortRegexp), RepositorySeparatorRegexp),
		Capture(RepositoryRegexp))

	// CapturedArtifactVersionRegexp is used to parse an artifact version sped
	// consisting of a repository part and an optional version part.
	CapturedArtifactVersionRegexp = Sequence(
		Capture(RepositoryRegexp),
		CapturedVersionRegexp)

	// AnchoredArtifactVersionRegexp is used to parse artifact versions.
	AnchoredArtifactVersionRegexp = Anchored(CapturedArtifactVersionRegexp)

	// CapturedVersionRegexp described the version part of a reference.
	CapturedVersionRegexp = Sequence(
		Optional(Literal(TagSeparator), Capture(TagRegexp)),
		Optional(Literal(DigestSeparator), Capture(DigestRegexp)))

	// ErrorCheckRegexp matches even wrong tags and/or digests.
	ErrorCheckRegexp = Anchored(
		Optional(Capture(Match(".*?")), Literal("::")),
		Capture(Match(".*?")),
		Optional(Literal(TagSeparator), Capture(Match(".*?"))),
		Optional(Literal(DigestSeparator), Capture(Match(".*?"))))

	////////////////////////////////////////////////////////////////////////////
	// now the various full flegded artifact flavors.

	// ReferenceRegexp is the full supported format of a reference. The regexp
	// is anchored and has capturing groups for name, tag, and digest
	// components.
	ReferenceRegexp = Anchored(
		Optional(Optional(CapturedSchemeRegexp), Capture(DomainPortRegexp), RepositorySeparatorRegexp),
		CapturedArtifactVersionRegexp)

	// DockerLibraryReferenceRegexp is a shortened docker library reference.
	DockerLibraryReferenceRegexp = Anchored(
		Capture(NameComponentRegexp),
		CapturedVersionRegexp)

	// DockerReferenceRegexp is a shortened docker reference.
	DockerReferenceRegexp = Anchored(
		Capture(DockerOrgRegexp, RepositorySeparatorRegexp, NameComponentRegexp),
		CapturedVersionRegexp)

	TypedRepoRegexp = Anchored(
		Capture(TypeRegexp), Literal("::"),
		Optional(CapturedSchemeRegexp), Capture(DomainPortRegexp))

	TypedURIRegexp = Anchored(
		Capture(TypeRegexp), Literal("::"),
		Optional(CapturedSchemeRegexp, Optional(Literal("//")), Capture(PathPortRegexp)),
		Optional(RepositorySeparatorRegexp, RepositorySeparatorRegexp, Optional(CapturedArtifactVersionRegexp)))

	TypedReferenceRegexp = Anchored(
		Capture(TypeRegexp), Literal("::"),
		Optional(CapturedSchemeRegexp, Optional(Literal("//"))), Capture(DomainPortRegexp),
		Optional(RepositorySeparatorRegexp, Optional(CapturedArtifactVersionRegexp)))

	TypedGenericReferenceRegexp = Anchored(
		Optional(Capture(TypeRegexp), Literal("::")),
		Capture(Match(".*?"), Match("[^:]")), Match(RepositorySeparator+RepositorySeparator),
		Optional(CapturedArtifactVersionRegexp))

	FileReferenceRegexp = Anchored(
		Optional(Capture(TypeRegexp), Literal("::")),
		Capture(Match("[./].*?"), Match("[^:]")), Match(RepositorySeparator+RepositorySeparator),
		Optional(CapturedArtifactVersionRegexp))

	// Unused.

	// IdentifierRegexp is the format for string identifier used as a
	// content addressable identifier using sha256. These identifiers
	// are like digests without the algorithm, since sha256 is used.
	IdentifierRegexp = Match(`([a-f0-9]{64})`)

	// ShortIdentifierRegexp is the format used to represent a prefix
	// of an identifier. A prefix may be used to match a sha256 identifier
	// within a list of trusted identifiers.
	ShortIdentifierRegexp = Match(`([a-f0-9]{6,64})`)

	// AnchoredIdentifierRegexp is used to check or match an
	// identifier value, anchored at start and end of string.
	AnchoredIdentifierRegexp = Anchored(IdentifierRegexp)
)

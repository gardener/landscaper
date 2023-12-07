// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"unicode"

	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessio/downloader"
	hd "github.com/open-component-model/ocm/pkg/common/accessio/downloader/http"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/github/identity"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/accspeccpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/mime"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// Type is the access type of GitHub registry.
const (
	Type   = "gitHub"
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

const (
	LegacyType   = "github"
	LegacyTypeV1 = LegacyType + runtime.VersionSeparator + "v1"
)

const ShaLength = 40

func init() {
	accspeccpi.RegisterAccessType(accspeccpi.NewAccessSpecType[*AccessSpec](Type, accspeccpi.WithDescription(usage)))
	accspeccpi.RegisterAccessType(accspeccpi.NewAccessSpecType[*AccessSpec](TypeV1, accspeccpi.WithFormatSpec(formatV1), accspeccpi.WithConfigHandler(ConfigHandler())))
	accspeccpi.RegisterAccessType(accspeccpi.NewAccessSpecType[*AccessSpec](LegacyType))
	accspeccpi.RegisterAccessType(accspeccpi.NewAccessSpecType[*AccessSpec](LegacyTypeV1))
}

func Is(spec accspeccpi.AccessSpec) bool {
	return spec != nil && spec.GetKind() == Type || spec.GetKind() == LegacyType
}

// AccessSpec describes the access for a GitHub registry.
type AccessSpec struct {
	runtime.ObjectVersionedType `json:",inline"`

	// RepoUrl is the repository URL, with host, owner and repository
	RepoURL string `json:"repoUrl"`

	// APIHostname is an optional different hostname for accessing the GitHub REST API
	// for enterprise installations
	APIHostname string `json:"apiHostname,omitempty"`

	// Commit defines the hash of the commit
	Commit string `json:"commit"`

	client     *http.Client
	downloader downloader.Downloader
}

var _ accspeccpi.AccessSpec = (*AccessSpec)(nil)

// AccessSpecOptions defines a set of options which can be applied to the access spec.
type AccessSpecOptions func(s *AccessSpec)

// WithClient creates an access spec with a custom http client.
func WithClient(client *http.Client) AccessSpecOptions {
	return func(s *AccessSpec) {
		s.client = client
	}
}

// WithDownloader defines a client with a custom downloader.
func WithDownloader(downloader downloader.Downloader) AccessSpecOptions {
	return func(s *AccessSpec) {
		s.downloader = downloader
	}
}

// New creates a new GitHub registry access spec version v1.
func New(repoURL, apiHostname, commit string, opts ...AccessSpecOptions) *AccessSpec {
	s := &AccessSpec{
		ObjectVersionedType: runtime.NewVersionedTypedObject(Type),
		RepoURL:             repoURL,
		APIHostname:         apiHostname,
		Commit:              commit,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (a *AccessSpec) Describe(ctx accspeccpi.Context) string {
	return fmt.Sprintf("GitHub commit %s[%s]", a.RepoURL, a.Commit)
}

func (_ *AccessSpec) IsLocal(accspeccpi.Context) bool {
	return false
}

func (a *AccessSpec) GlobalAccessSpec(ctx accspeccpi.Context) accspeccpi.AccessSpec {
	return a
}

func (_ *AccessSpec) GetType() string {
	return Type
}

func (a *AccessSpec) AccessMethod(c accspeccpi.ComponentVersionAccess) (accspeccpi.AccessMethod, error) {
	return accspeccpi.AccessMethodForImplementation(newMethod(c, a))
}

func (a *AccessSpec) GetInexpensiveContentVersionIdentity(access accspeccpi.ComponentVersionAccess) string {
	return a.Commit
}

func (a *AccessSpec) createHTTPClient(token string) *http.Client {
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		ctx := context.Background()
		// set up the test client if we have one
		if a.client != nil {
			ctx = context.WithValue(ctx, oauth2.HTTPClient, a.client)
		}
		return oauth2.NewClient(ctx, ts)
	}
	return a.client
}

// RepositoryService defines capabilities of a GitHub repository.
type RepositoryService interface {
	GetArchiveLink(ctx context.Context, owner, repo string, archiveformat github.ArchiveFormat, opts *github.RepositoryContentGetOptions, followRedirects bool) (*url.URL, *github.Response, error)
}

type accessMethod struct {
	lock   sync.Mutex
	access blobaccess.BlobAccess

	compvers          accspeccpi.ComponentVersionAccess
	spec              *AccessSpec
	repositoryService RepositoryService
	owner             string
	repo              string
	cid               credentials.ConsumerIdentity
}

var _ accspeccpi.AccessMethodImpl = (*accessMethod)(nil)

func newMethod(c accspeccpi.ComponentVersionAccess, a *AccessSpec) (*accessMethod, error) {
	if err := validateCommit(a.Commit); err != nil {
		return nil, fmt.Errorf("failed to validate commit: %w", err)
	}

	unparsed := a.RepoURL
	if !strings.HasPrefix(unparsed, "https://") && !strings.HasPrefix(unparsed, "http://") {
		unparsed = "https://" + unparsed
	}
	u, err := url.Parse(unparsed)
	if err != nil {
		return nil, errors.ErrInvalidWrap(err, "repository url", a.RepoURL)
	}

	path := strings.Trim(u.Path, "/")
	pathcomps := strings.Split(path, "/")
	if len(pathcomps) != 2 {
		return nil, errors.ErrInvalid("repository path", path, a.RepoURL)
	}

	token, cid, err := getCreds(unparsed, path, c.GetContext().CredentialsContext())
	if err != nil {
		return nil, fmt.Errorf("failed to get creds: %w", err)
	}

	var client *github.Client
	httpclient := a.createHTTPClient(token)

	if u.Hostname() == "github.com" {
		client = github.NewClient(httpclient)
	} else {
		t := *u
		t.Path = ""
		if a.APIHostname != "" {
			t.Host = a.APIHostname
		}

		client, err = github.NewEnterpriseClient(t.String(), t.String(), httpclient)
		if err != nil {
			return nil, err
		}
	}

	return &accessMethod{
		spec:              a,
		compvers:          c,
		owner:             pathcomps[0],
		repo:              pathcomps[1],
		cid:               cid,
		repositoryService: client.Repositories,
	}, nil
}

func validateCommit(commit string) error {
	if len(commit) != ShaLength {
		return fmt.Errorf("commit is not a SHA")
	}
	for _, c := range commit {
		if !unicode.IsOneOf([]*unicode.RangeTable{unicode.Letter, unicode.Digit}, c) {
			return fmt.Errorf("commit contains invalid characters for a SHA")
		}
	}
	return nil
}

func getCreds(serverurl, path string, cctx credentials.Context) (string, credentials.ConsumerIdentity, error) {
	id := identity.GetConsumerId(serverurl, path)
	creds, err := credentials.CredentialsForConsumer(cctx.CredentialsContext(), id, identity.IdentityMatcher)
	if creds == nil || err != nil {
		return "", id, err
	}
	return creds.GetProperty(credentials.ATTR_TOKEN), id, nil
}

func (_ *accessMethod) IsLocal() bool {
	return false
}

func (m *accessMethod) GetKind() string {
	return Type
}

func (m *accessMethod) MimeType() string {
	return mime.MIME_TGZ
}

func (m *accessMethod) AccessSpec() accspeccpi.AccessSpec {
	return m.spec
}

func (m *accessMethod) Get() ([]byte, error) {
	if err := m.setup(); err != nil {
		return nil, err
	}
	return m.access.Get()
}

func (m *accessMethod) Reader() (io.ReadCloser, error) {
	if err := m.setup(); err != nil {
		return nil, err
	}
	return m.access.Reader()
}

func (m *accessMethod) Close() error {
	if m.access == nil {
		return nil
	}
	return m.access.Close()
}

func (m *accessMethod) setup() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.access != nil {
		return nil
	}

	// to get the download link technical access to the github repo is required.
	// therefore, this part has to be delayed until the effective access and cannot
	// be done during creation of the access method object.
	link, err := m.getDownloadLink()
	if err != nil {
		return fmt.Errorf("failed to get download link: %w", err)
	}

	d := hd.NewDownloader(link)
	if m.spec.downloader != nil {
		d = m.spec.downloader
	}

	w := accessio.NewWriteAtWriter(d.Download)
	cacheBlobAccess := accessobj.CachedBlobAccessForWriter(m.compvers.GetContext(), m.MimeType(), w)
	m.access = cacheBlobAccess
	return nil
}

func (m *accessMethod) getDownloadLink() (string, error) {
	link, resp, err := m.repositoryService.GetArchiveLink(context.Background(), m.owner, m.repo, github.Tarball, &github.RepositoryContentGetOptions{
		Ref: m.spec.Commit,
	}, true)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return link.String(), nil
}

func (m *accessMethod) GetConsumerId(uctx ...credentials.UsageContext) credentials.ConsumerIdentity {
	return m.cid
}

func (m *accessMethod) GetIdentityMatcher() string {
	return identity.CONSUMER_TYPE
}

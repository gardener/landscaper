// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocireg

import (
	"context"
	"path"
	"strings"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/remotes/docker/config"
	"github.com/mandelsoft/logging"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	ociidentity "github.com/open-component-model/ocm/pkg/contexts/oci/identity"
	"github.com/open-component-model/ocm/pkg/docker"
	"github.com/open-component-model/ocm/pkg/docker/resolve"
	"github.com/open-component-model/ocm/pkg/errors"
	ocmlog "github.com/open-component-model/ocm/pkg/logging"
	"github.com/open-component-model/ocm/pkg/utils"
)

type RepositoryInfo struct {
	Scheme  string
	Locator string
	Creds   credentials.Credentials
	Legacy  bool
}

func (r *RepositoryInfo) HostPort() string {
	i := strings.Index(r.Locator, "/")
	if i < 0 {
		return r.Locator
	} else {
		return r.Locator[:i]
	}
}

func (r *RepositoryInfo) HostInfo() (string, string, string) {
	return utils.SplitLocator(r.Locator)
}

type RepositoryImpl struct {
	cpi.RepositoryImplBase
	logger logging.UnboundLogger
	spec   *RepositorySpec
	info   *RepositoryInfo
}

var (
	_ cpi.RepositoryImpl                   = (*RepositoryImpl)(nil)
	_ credentials.ConsumerIdentityProvider = &RepositoryImpl{}
)

func NewRepository(ctx cpi.Context, spec *RepositorySpec, info *RepositoryInfo) (cpi.Repository, error) {
	urs := spec.UniformRepositorySpec()
	i := &RepositoryImpl{
		RepositoryImplBase: cpi.NewRepositoryImplBase(ctx),
		logger:             logging.DynamicLogger(ctx, REALM, logging.NewAttribute(ocmlog.ATTR_HOST, urs.Host)),
		spec:               spec,
		info:               info,
	}
	return cpi.NewRepository(i), nil
}

func GetRepositoryImplementation(r cpi.Repository) (*RepositoryImpl, error) {
	i, err := cpi.GetRepositoryImplementation(r)
	if err != nil {
		return nil, err
	}
	return i.(*RepositoryImpl), nil
}

func (r *RepositoryImpl) GetSpecification() cpi.RepositorySpec {
	return r.spec
}

func (r *RepositoryImpl) Close() error {
	return nil
}

func (r *RepositoryImpl) GetConsumerId(uctx ...credentials.UsageContext) credentials.ConsumerIdentity {
	if c, ok := utils.Optional(uctx...).(credentials.StringUsageContext); ok {
		return ociidentity.GetConsumerId(r.info.Locator, c.String())
	}
	return ociidentity.GetConsumerId(r.info.Locator, "")
}

func (r *RepositoryImpl) GetIdentityMatcher() string {
	return ociidentity.CONSUMER_TYPE
}

func (r *RepositoryImpl) NamespaceLister() cpi.NamespaceLister {
	return nil
}

func (r *RepositoryImpl) IsReadOnly() bool {
	return false
}

func (r *RepositoryImpl) getCreds(comp string) (credentials.Credentials, error) {
	if r.info.Creds != nil {
		return r.info.Creds, nil
	}
	return ociidentity.GetCredentials(r.GetContext(), r.info.Locator, comp)
}

func (r *RepositoryImpl) getResolver(comp string) (resolve.Resolver, error) {
	creds, err := r.getCreds(comp)
	if err != nil {
		if !errors.IsErrUnknownKind(err, credentials.KIND_CONSUMER) {
			return nil, err
		}
	}
	logger := r.logger.BoundLogger().WithValues(ocmlog.ATTR_NAMESPACE, comp)
	if creds == nil {
		logger.Trace("no credentials")
	}

	opts := docker.ResolverOptions{
		Hosts: docker.ConvertHosts(config.ConfigureHosts(context.Background(), config.HostOptions{
			Credentials: func(host string) (string, string, error) {
				if creds != nil {
					p := creds.GetProperty(credentials.ATTR_IDENTITY_TOKEN)
					if p == "" {
						p = creds.GetProperty(credentials.ATTR_PASSWORD)
					}
					pw := ""
					if p != "" {
						pw = "***"
					}
					logger.Trace("query credentials", ocmlog.ATTR_USER, creds.GetProperty(credentials.ATTR_USERNAME), "pass", pw)
					return creds.GetProperty(credentials.ATTR_USERNAME), p, nil
				}
				logger.Trace("no credentials")
				return "", "", nil
			},
			DefaultScheme: r.info.Scheme,
		})),
	}

	return docker.NewResolver(opts), nil
}

func (r *RepositoryImpl) GetRef(comp, vers string) string {
	base := path.Join(r.info.Locator, comp)
	if vers == "" {
		return base
	}
	if ok, d := artdesc.IsDigest(vers); ok {
		return base + "@" + d.String()
	}
	return base + ":" + vers
}

func (r *RepositoryImpl) GetBaseURL() string {
	return r.spec.BaseURL
}

func (r *RepositoryImpl) ExistsArtifact(name string, version string) (bool, error) {
	res, err := r.getResolver(name)
	if err != nil {
		return false, err
	}
	ref := r.GetRef(name, version)
	_, _, err = res.Resolve(context.Background(), ref)

	if err != nil {
		if errdefs.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *RepositoryImpl) LookupArtifact(name string, version string) (acc cpi.ArtifactAccess, err error) {
	ns, err := NewNamespace(r, name)
	if err != nil {
		return nil, err
	}
	defer accessio.PropagateCloseTemporary(&err, ns) // temporary namespace object not exposed.

	return ns.GetArtifact(version)
}

func (r *RepositoryImpl) LookupNamespace(name string) (cpi.NamespaceAccess, error) {
	return NewNamespace(r, name)
}

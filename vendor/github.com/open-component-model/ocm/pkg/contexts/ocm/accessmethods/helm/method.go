// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/accspeccpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/helm"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// Type is the access type for a blob in an OCI repository.
const (
	Type   = "helm"
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

func init() {
	accspeccpi.RegisterAccessType(accspeccpi.NewAccessSpecType[*AccessSpec](Type, accspeccpi.WithDescription(usage)))
	accspeccpi.RegisterAccessType(accspeccpi.NewAccessSpecType[*AccessSpec](TypeV1, accspeccpi.WithFormatSpec(formatV1), accspeccpi.WithConfigHandler(ConfigHandler())))
}

// New creates a new Helm Chart accessor for helm repositories.
func New(chart string, repourl string) *AccessSpec {
	return &AccessSpec{
		ObjectVersionedType: runtime.NewVersionedTypedObject(Type),
		HelmChart:           chart,
		HelmRepository:      repourl,
	}
}

// AccessSpec describes the access for a helm repository.
type AccessSpec struct {
	runtime.ObjectVersionedType `json:",inline"`

	// HelmRepository is the URL og the helm repository to load the chart from.
	HelmRepository string `json:"helmRepository"`

	// HelmChart if the name of the helm chart and its version separated by a colon.
	HelmChart string `json:"helmChart"`

	// Version can either be specified as part of the chart name or separately.
	Version string `json:"version,omitempty"`

	// CACert is an optional root TLS certificate
	CACert string `json:"caCert,omitempty"`

	// Keyring is an optional keyring to verify the chart.
	Keyring string `json:"keyring,omitempty"`
}

var _ accspeccpi.AccessSpec = (*AccessSpec)(nil)

func (a *AccessSpec) Describe(ctx accspeccpi.Context) string {
	return fmt.Sprintf("Helm chart %s:%s in repository %s", a.GetChartName(), a.GetVersion(), a.HelmRepository)
}

func (a *AccessSpec) IsLocal(ctx accspeccpi.Context) bool {
	return false
}

func (a *AccessSpec) GlobalAccessSpec(ctx accspeccpi.Context) accspeccpi.AccessSpec {
	return a
}

func (a *AccessSpec) GetMimeType() string {
	return helm.ChartMediaType
}

func (a *AccessSpec) AccessMethod(access accspeccpi.ComponentVersionAccess) (accspeccpi.AccessMethod, error) {
	return accspeccpi.AccessMethodForImplementation(&accessMethod{comp: access, spec: a}, nil)
}

func (a *AccessSpec) GetInexpensiveContentVersionIdentity(access accspeccpi.ComponentVersionAccess) string {
	return ""
	// TODO: research possibilities with provenance file
}

///////////////////

func (a *AccessSpec) GetVersion() string {
	parts := strings.Split(a.HelmChart, ":")
	if len(parts) > 1 {
		return parts[1]
	}
	return a.Version
}

func (a *AccessSpec) GetChartName() string {
	parts := strings.Split(a.HelmChart, ":")
	return parts[0]
}

////////////////////////////////////////////////////////////////////////////////

type accessMethod struct {
	lock sync.Mutex
	blob blobaccess.BlobAccess
	comp accspeccpi.ComponentVersionAccess
	spec *AccessSpec

	acc helm.ChartAccess
}

var _ accspeccpi.AccessMethodImpl = (*accessMethod)(nil)

func (_ *accessMethod) IsLocal() bool {
	return false
}

func (m *accessMethod) GetKind() string {
	return Type
}

func (m *accessMethod) AccessSpec() accspeccpi.AccessSpec {
	return m.spec
}

func (m *accessMethod) Close() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.blob != nil {
		m.blob.Close()
		m.acc.Close()
		m.blob = nil
	}
	return nil
}

func (m *accessMethod) Get() ([]byte, error) {
	return blobaccess.BlobData(m.getBlob())
}

func (m *accessMethod) Reader() (io.ReadCloser, error) {
	return blobaccess.BlobReader(m.getBlob())
}

func (m *accessMethod) MimeType() string {
	return helm.ChartMediaType
}

func (m *accessMethod) getBlob() (blobaccess.BlobAccess, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.blob != nil {
		return m.blob, nil
	}

	vers := m.spec.GetVersion()
	name := m.spec.GetChartName()

	parts := strings.Split(m.spec.HelmChart, ":")
	switch len(parts) {
	case 1:
		if vers == "" {
			return nil, errors.ErrInvalid("helm chart", m.spec.HelmChart)
		}
	case 2:
		if vers != parts[1] {
			return nil, errors.ErrInvalid("helm chart", m.spec.HelmChart+"["+vers+"]")
		}
	default:
		return nil, errors.ErrInvalid("helm chart", m.spec.HelmChart)
	}

	acc, err := helm.DownloadChart(common.NonePrinter, m.comp.GetContext(), name, vers, m.spec.HelmRepository,
		helm.WithCredentials(identity.GetCredentials(m.comp.GetContext(), m.spec.HelmRepository, m.spec.GetChartName())),
		helm.WithKeyring([]byte(m.spec.Keyring)),
		helm.WithRootCert([]byte(m.spec.CACert)))
	if err != nil {
		return nil, err
	}
	m.blob, err = acc.Chart()
	if err != nil {
		acc.Close()
	}
	m.acc = acc
	return m.blob, nil
}

func (m *accessMethod) GetConsumerId(uctx ...credentials.UsageContext) credentials.ConsumerIdentity {
	return identity.GetConsumerId(m.spec.HelmRepository, m.spec.GetChartName())
}

func (m *accessMethod) GetIdentityMatcher() string {
	return identity.CONSUMER_TYPE
}

// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signingattr

import (
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"golang.org/x/exp/slices"

	cfgcpi "github.com/open-component-model/ocm/pkg/contexts/config/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/signing"
	"github.com/open-component-model/ocm/pkg/utils"
)

const (
	ConfigType   = "keys" + cfgcpi.OCM_CONFIG_TYPE_SUFFIX
	ConfigTypeV1 = ConfigType + runtime.VersionSeparator + "v1"
)

func init() {
	cfgcpi.RegisterConfigType(cfgcpi.NewConfigType[*Config](ConfigType, usage))
	cfgcpi.RegisterConfigType(cfgcpi.NewConfigType[*Config](ConfigTypeV1, usage))
}

type Issuer struct {
	CommonName         string   `json:"commonName,omitempty"`
	Organization       []string `json:"organization,omitempty"`
	OrganizationalUnit []string `json:"organizationalUnit,omitempty"`

	Country       []string `json:"country,omitempty"`
	Locality      []string `json:"locality,omitempty"`
	Province      []string `json:"province,omitempty"`
	StreetAddress []string `json:"streetAddress,omitempty"`
	PostalCode    []string `json:"postalCode,omitempty"`
}

func (i *Issuer) Get() *pkix.Name {
	return &pkix.Name{
		CommonName: i.CommonName,

		Country:            slices.Clone(i.Country),
		Organization:       slices.Clone(i.Organization),
		OrganizationalUnit: slices.Clone(i.OrganizationalUnit),
		Locality:           slices.Clone(i.Locality),
		Province:           slices.Clone(i.Province),
		StreetAddress:      slices.Clone(i.StreetAddress),
		PostalCode:         slices.Clone(i.PostalCode),
	}
}

func (i *Issuer) Set(issuer *pkix.Name) {
	i.CommonName = issuer.CommonName

	i.Country = slices.Clone(issuer.Country)
	i.Organization = slices.Clone(issuer.Organization)
	i.OrganizationalUnit = slices.Clone(issuer.OrganizationalUnit)
	i.Locality = slices.Clone(issuer.Locality)
	i.Province = slices.Clone(issuer.Province)
	i.StreetAddress = slices.Clone(issuer.StreetAddress)
	i.PostalCode = slices.Clone(issuer.PostalCode)
}

// Config describes a memory based repository interface.
type Config struct {
	runtime.ObjectVersionedType `json:",inline"`
	PublicKeys                  map[string]KeySpec `json:"publicKeys,omitempty"`
	PrivateKeys                 map[string]KeySpec `json:"privateKeys,omitempty"`
	Issuers                     map[string]Issuer  `json:"issuers,omitempty"`
	TSAUrl                      string             `json:"tsaURL,omitempty"`
}

type RawData []byte

var _ json.Unmarshaler = (*RawData)(nil)

func (r RawData) MarshalJSON() ([]byte, error) {
	return json.Marshal(base64.StdEncoding.EncodeToString(r))
}

func (r *RawData) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	*r, err = base64.StdEncoding.DecodeString(s)
	return err
}

type KeySpec struct {
	Data       RawData        `json:"data,omitempty"`
	StringData string         `json:"stringdata,omitempty"`
	Path       string         `json:"path,omitempty"`
	Parsed     interface{}    `json:"-"`
	FileSystem vfs.FileSystem `json:"-"`
}

func (k *KeySpec) Get() (interface{}, error) {
	if k.Parsed != nil {
		return k.Parsed, nil
	}
	if k.Data != nil {
		if k.StringData != "" || k.Path != "" {
			return nil, errors.Newf("only one of data, stringdata or path may be set")
		}
		return []byte(k.Data), nil
	}
	if k.StringData != "" {
		if k.Path != "" {
			return nil, errors.Newf("only one of data, stringdata or path may be set")
		}
		return []byte(k.StringData), nil
	}
	fs := k.FileSystem
	if fs == nil {
		fs = osfs.New()
	}

	return utils.ReadFile(k.Path, fs)
}

// New creates a new memory ConfigSpec.
func New() *Config {
	return &Config{
		ObjectVersionedType: runtime.NewVersionedTypedObject(ConfigType),
	}
}

func (a *Config) GetType() string {
	return ConfigType
}

func (a *Config) AddIssuer(name string, issuer *pkix.Name) {
	var i Issuer

	i.Set(issuer)
	if a.Issuers == nil {
		a.Issuers = map[string]Issuer{}
	}
	a.Issuers[name] = i
}

func (a *Config) addKey(set *map[string]KeySpec, name string, key interface{}) {
	if *set == nil {
		*set = map[string]KeySpec{}
	}
	(*set)[name] = KeySpec{Parsed: key}
}

func (a *Config) AddPublicKey(name string, key interface{}) {
	a.addKey(&a.PublicKeys, name, key)
}

func (a *Config) AddPrivateKey(name string, key interface{}) {
	a.addKey(&a.PrivateKeys, name, key)
}

func (a *Config) addKeyFile(set *map[string]KeySpec, name, path string, fss ...vfs.FileSystem) {
	var fs vfs.FileSystem
	for _, fs = range fss {
		if fs != nil {
			break
		}
	}
	if *set == nil {
		*set = map[string]KeySpec{}
	}
	(*set)[name] = KeySpec{Path: path, FileSystem: fs}
}

func (a *Config) AddPublicKeyFile(name, path string, fss ...vfs.FileSystem) {
	a.addKeyFile(&a.PublicKeys, name, path, fss...)
}

func (a *Config) AddPrivateKeyFile(name, path string, fss ...vfs.FileSystem) {
	a.addKeyFile(&a.PrivateKeys, name, path, fss...)
}

func (a *Config) addKeyData(set *map[string]KeySpec, name string, data []byte) {
	if *set == nil {
		*set = map[string]KeySpec{}
	}
	(*set)[name] = KeySpec{Data: data}
}

func (a *Config) AddPublicKeyData(name string, data []byte) {
	a.addKeyData(&a.PublicKeys, name, data)
}

func (a *Config) AddPrivateKeyData(name string, data []byte) {
	a.addKeyData(&a.PrivateKeys, name, data)
}

func (a *Config) ApplyTo(ctx cfgcpi.Context, target interface{}) error {
	t, ok := target.(Context)
	if !ok {
		return cfgcpi.ErrNoContext(ConfigType)
	}
	return errors.Wrapf(a.ApplyToRegistry(Get(t)), "applying config failed")
}

func (a *Config) ApplyToRegistry(registry signing.Registry) error {
	for n, k := range a.PublicKeys {
		key, err := k.Get()
		if err != nil {
			return errors.Wrapf(err, "cannot get public key %s", n)
		}
		registry.RegisterPublicKey(n, key)
	}
	for n, k := range a.PrivateKeys {
		key, err := k.Get()
		if err != nil {
			return errors.Wrapf(err, "cannot get private key %s", n)
		}
		registry.RegisterPrivateKey(n, key)
	}
	for n, k := range a.Issuers {
		registry.RegisterIssuer(n, k.Get())
	}
	if a.TSAUrl != "" {
		registry.SetTSAUrl(a.TSAUrl)
	}
	return nil
}

const usage = `
The config type <code>` + ConfigType + `</code> can be used to define
public and private keys. A key value might be given by one of the fields:
- <code>path</code>: path of file with key data
- <code>data</code>: base64 encoded binary data
- <code>stringdata</code>: data a string parsed by key handler

<pre>
    type: ` + ConfigType + `
    privateKeys:
       &lt;name>:
         path: &lt;file path>
       ...
    publicKeys:
       &lt;name>:
         data: &lt;base64 encoded key representation>
       ...
    issuers:
       &lt;name>:
         commonName: acme.org
</pre>

Issuers define an expected distinguished name for
public key certificates optionally provided together with 
signatures. They support the following fields:
- <code>commonName</code> *string*
- <code>organization</code> *string array*
- <code>organizationalUnit</code> *string array*
- <code>country</code> *string array*
- <code>locality</code> *string array*
- <code>province</code> *string array*
- <code>streetAddress</code> *string array*
- <code>postalCode</code> *string array*

At least the given values must be present in the certificate
to be accepted for a successful signature validation.

`

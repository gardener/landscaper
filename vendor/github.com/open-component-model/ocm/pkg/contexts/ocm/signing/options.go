// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"crypto/x509/pkix"
	"fmt"
	"strings"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/signingattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/generics"
	"github.com/open-component-model/ocm/pkg/signing"
	"github.com/open-component-model/ocm/pkg/signing/hasher/sha256"
	"github.com/open-component-model/ocm/pkg/signing/signutils"
	"github.com/open-component-model/ocm/pkg/utils"
)

type Option interface {
	ApplySigningOption(o *Options)
}

////////////////////////////////////////////////////////////////////////////////

type printer struct {
	printer common.Printer
}

// Printer provides an option configuring a printer for a signing/verification
// operation.
func Printer(p common.Printer) Option {
	return &printer{p}
}

func (o *printer) ApplySigningOption(opts *Options) {
	opts.Printer = o.printer
}

////////////////////////////////////////////////////////////////////////////////

const (
	DIGESTMODE_LOCAL = "local" // (default) store nested digests locally in component descriptor
	DIGESTMODE_TOP   = "top"   // store aggregated nested digests in signed component version
)

type digestmode struct {
	mode string
}

// DigestMode provides an option configuring the digest mode for a signing/verification
// operation. Possible values are
//   - DIGESTMODE_LOCAL(default) all digest information is store along with a component version
//   - DIGESTMODE_TOP (experimental) all digest information is gathered for referenced component versions in the initially signed component version.
func DigestMode(name string) Option {
	return &digestmode{name}
}

func (o *digestmode) ApplySigningOption(opts *Options) {
	opts.DigestMode = o.mode
}

////////////////////////////////////////////////////////////////////////////////

type recursive struct {
	flag bool
}

// Recursive provides an option configuring recursion for a signing/verification
// operation. If enabled the operation will be done for all component versions
// in the reference graph.
func Recursive(flags ...bool) Option {
	return &recursive{utils.GetOptionFlag(flags...)}
}

func (o *recursive) ApplySigningOption(opts *Options) {
	opts.Recursively = o.flag
}

////////////////////////////////////////////////////////////////////////////////

type update struct {
	flag bool
}

// Update provides an option configuring the update mode for a signing/verification
// operation. Only if enabled, state changes will be persisted.
func Update(flags ...bool) Option {
	return &update{utils.GetOptionFlag(flags...)}
}

func (o *update) ApplySigningOption(opts *Options) {
	opts.Update = o.flag
}

////////////////////////////////////////////////////////////////////////////////

type verify struct {
	flag bool
}

// VerifyDigests provides an option requesting signature verification for a
// signing/verification operation.
func VerifyDigests(flags ...bool) Option {
	return &verify{utils.GetOptionFlag(flags...)}
}

func (o *verify) ApplySigningOption(opts *Options) {
	opts.Verify = o.flag
}

////////////////////////////////////////////////////////////////////////////////

type signer struct {
	algo   string
	signer signing.Signer
	name   string
}

// Sign provides an option requesting signing for a dedicated name and signer for a
// signing operation.
func Sign(h signing.Signer, name string) Option {
	return &signer{"", h, name}
}

// Signer provides an option requesting to use a dedicated signer for a
// signing/verification operation.
func Signer(h signing.Signer) Option {
	return &signer{"", h, ""}
}

// SignByAlgo provides an option requesting signing with a signing algorithm
// for a signing operation. The effective signer is taken from
// the signer registry provided by the OCM context.
func SignByAlgo(algo string, name string) Option {
	return &signer{algo, nil, name}
}

// SignerByAlgo provides an option requesting to use a dedicated signer by
// algorithm for a signing operation. The effective signer is taken from
// the signer registry provided by the OCM context.
func SignerByAlgo(algo string) Option {
	return &signer{algo, nil, ""}
}

// SignerByName set a signer by algorithm name.
//
// Deprecated: use SignerByAlgo.
func SignerByName(algo string) Option {
	return SignerByAlgo(algo)
}

func (o *signer) ApplySigningOption(opts *Options) {
	n := strings.TrimSpace(o.name)
	if n != "" {
		opts.SignatureNames = append([]string{n}, opts.SignatureNames...)
	}
	opts.SignAlgo = o.algo
	opts.Signer = o.signer
}

////////////////////////////////////////////////////////////////////////////////

type hasher struct {
	algo   string
	hasher signing.Hasher
}

// Hash provides an option requesting hashing with a dedicated hasher for a
// signing/hash operation.
func Hash(h signing.Hasher) Option {
	return &hasher{"", h}
}

// HashByAlgo provides an option requesting to use a dedicated hasher by name
// for a signing/hash operation. The effective hasher is taken from
// the hasher registry provided by the OCM context.
func HashByAlgo(algo string) Option {
	return &hasher{algo, nil}
}

func (o *hasher) ApplySigningOption(opts *Options) {
	opts.HashAlgo = o.algo
	opts.Hasher = o.hasher
}

////////////////////////////////////////////////////////////////////////////////

type verifier struct {
	name string
}

// VerifySignature provides an option requesting verification for dedicated
// signature names for a signing/verification operation. If no name is specified
// the names are taken from the component version.
func VerifySignature(names ...string) Option {
	name := ""
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n != "" {
			name = n
			break
		}
	}
	return &verifier{name}
}

func (o *verifier) ApplySigningOption(opts *Options) {
	opts.VerifySignature = true
	if o.name != "" {
		opts.SignatureNames = append(opts.SignatureNames, o.name)
	}
}

////////////////////////////////////////////////////////////////////////////////

type resolver struct {
	resolver []ocm.ComponentVersionResolver
}

// Resolver provides an option requesting to use a dedicated component version
// resolver for a signing/verification operation. It is used to resolve
// references in component versions.
func Resolver(h ...ocm.ComponentVersionResolver) Option {
	return &resolver{h}
}

func (o *resolver) ApplySigningOption(opts *Options) {
	opts.Resolver = ocm.NewCompoundResolver(append([]ocm.ComponentVersionResolver{opts.Resolver}, o.resolver...)...)
}

////////////////////////////////////////////////////////////////////////////////

type skip struct {
	skip map[string]bool
}

// SkipAccessTypes provides an option to declare dedicated resource types
// which should be excluded from digesting. This is a legacy options,
// required only for the handling of older component version not yet
// completely configured with resource digests. The content of resources with
// the given types will be marked as not signature relevant.
func SkipAccessTypes(names ...string) Option {
	m := map[string]bool{}
	for _, n := range names {
		m[n] = true
	}
	return &skip{m}
}

func (o *skip) ApplySigningOption(opts *Options) {
	if len(o.skip) > 0 {
		if opts.SkipAccessTypes == nil {
			opts.SkipAccessTypes = map[string]bool{}
		}
		for k, v := range o.skip {
			opts.SkipAccessTypes[k] = v
		}
	}
}

////////////////////////////////////////////////////////////////////////////////

type registry struct {
	registry signing.Registry
}

// Registry provides an option requesting to use a dedicated signing registry
// for a signing/verification operation. It is used to lookup
// signers, verifiers, hashers and signing public/private keys by name.
func Registry(h signing.Registry) Option {
	return &registry{h}
}

func (o *registry) ApplySigningOption(opts *Options) {
	opts.Registry = o.registry
}

////////////////////////////////////////////////////////////////////////////////

type signame struct {
	name  string
	reset bool
}

// SignatureName provides an option requesting to use dedicated signature names
// for a signing/verification operation.
func SignatureName(name string, reset ...bool) Option {
	return &signame{name, utils.Optional(reset...)}
}

func (o *signame) ApplySigningOption(opts *Options) {
	if o.reset {
		opts.SignatureNames = nil
	}
	if o.name != "" {
		opts.SignatureNames = append(opts.SignatureNames, o.name)
	}
}

////////////////////////////////////////////////////////////////////////////////

type issuer struct {
	issuer pkix.Name
	name   string
	err    error
}

// Issuer provides an option requesting to use a dedicated issuer name
// for a signing operation.
func Issuer(is string) Option {
	dn, err := signutils.ParseDN(is)
	if err != nil {
		return &issuer{err: err}
	}
	return &issuer{issuer: *dn}
}

func IssuerFor(name string, is string) Option {
	dn, err := signutils.ParseDN(is)
	if err != nil {
		return &issuer{err: err}
	}
	return PKIXIssuerFor(name, *dn)
}

// PKIXIssuer provides an option requesting to use a dedicated issuer name
// for a signing operation.
func PKIXIssuer(is pkix.Name) Option {
	return &issuer{issuer: is}
}

func PKIXIssuerFor(name string, is pkix.Name) Option {
	return &issuer{issuer: is, name: name}
}

func (o *issuer) ApplySigningOption(opts *Options) {
	if o.name != "" {
		if opts.Keys == nil {
			opts.Keys = signing.NewKeyRegistry()
		}
		opts.Keys.RegisterIssuer(o.name, generics.Pointer(o.issuer))
	} else {
		opts.Issuer = generics.Pointer(o.issuer)
	}
}

////////////////////////////////////////////////////////////////////////////////

type rootcerts struct {
	pool signutils.GenericCertificatePool
}

// RootCertificates provides an option requesting to dedicated root certificates
// for a signing/verification operation using certificates.
func RootCertificates(pool signutils.GenericCertificatePool) Option {
	return &rootcerts{pool}
}

func (o *rootcerts) ApplySigningOption(opts *Options) {
	opts.RootCerts = o.pool
}

////////////////////////////////////////////////////////////////////////////////

type privkey struct {
	name string
	key  interface{}
}

// PrivateKey provides an option requesting to use a dedicated private key
// for a dedicated signature name for a signing operation.
func PrivateKey(name string, key interface{}) Option {
	return &privkey{name, key}
}

func (o *privkey) ApplySigningOption(opts *Options) {
	if o.key == nil {
		return
	}
	if opts.Keys == nil {
		opts.Keys = signing.NewKeyRegistry()
	}
	opts.Keys.RegisterPrivateKey(o.name, o.key)
}

////////////////////////////////////////////////////////////////////////////////

type pubkey struct {
	name string
	key  interface{}
}

// PublicKey provides an option requesting to use a dedicated public key
// for a dedicated signature name for a verification operation.
func PublicKey(name string, key interface{}) Option {
	return &pubkey{name, key}
}

func (o *pubkey) ApplySigningOption(opts *Options) {
	if o.key == nil {
		return
	}
	if opts.Keys == nil {
		opts.Keys = signing.NewKeyRegistry()
	}
	opts.Keys.RegisterPublicKey(o.name, o.key)
}

////////////////////////////////////////////////////////////////////////////////

type tsaOpt struct {
	url string
	use *bool
}

// UseTSA enables the usage of a timestamp server authority.
func UseTSA(flag ...bool) Option {
	return &tsaOpt{use: utils.BoolP(utils.GetOptionFlag(flag...))}
}

// TSAUrl selects the TSA server URL to use, if TSA mode is enabled.
func TSAUrl(url string) Option {
	return &tsaOpt{url: url}
}

func (o *tsaOpt) ApplySigningOption(opts *Options) {
	if o.url != "" {
		opts.TSAUrl = o.url
	}
	if o.use != nil {
		opts.UseTSA = *o.use
	}
}

////////////////////////////////////////////////////////////////////////////////

type Options struct {
	Printer           common.Printer
	Update            bool
	Recursively       bool
	DigestMode        string
	Verify            bool
	SignAlgo          string
	Signer            signing.Signer
	Issuer            *pkix.Name
	VerifySignature   bool
	RootCerts         signutils.GenericCertificatePool
	HashAlgo          string
	Hasher            signing.Hasher
	Keys              signing.KeyRegistry
	Registry          signing.Registry
	Resolver          ocm.ComponentVersionResolver
	SkipAccessTypes   map[string]bool
	SignatureNames    []string
	NormalizationAlgo string
	Keyless           bool
	TSAUrl            string
	UseTSA            bool

	effectiveRegistry signing.Registry
}

var _ Option = (*Options)(nil)

func NewOptions(list ...Option) *Options {
	return (&Options{}).Eval(list...)
}

func (opts *Options) Eval(list ...Option) *Options {
	for _, o := range list {
		o.ApplySigningOption(opts)
	}
	return opts
}

func (o *Options) ApplySigningOption(opts *Options) {
	if o.Printer != nil {
		opts.Printer = o.Printer
	}
	if o.Keys != nil {
		opts.Keys = o.Keys
	}
	if o.Signer != nil {
		opts.Signer = o.Signer
	}
	if o.DigestMode != "" {
		opts.DigestMode = o.DigestMode
	}
	if o.VerifySignature {
		opts.VerifySignature = o.VerifySignature
	}
	if o.Hasher != nil {
		opts.Hasher = o.Hasher
	}
	if o.Registry != nil {
		opts.Registry = o.Registry
	}
	if o.Resolver != nil {
		opts.Resolver = o.Resolver
	}
	if len(o.SignatureNames) != 0 {
		opts.SignatureNames = o.SignatureNames
	}
	if o.SkipAccessTypes != nil {
		if opts.SkipAccessTypes == nil {
			opts.SkipAccessTypes = map[string]bool{}
		}
		for k, v := range o.SkipAccessTypes {
			opts.SkipAccessTypes[k] = v
		}
	}
	if o.Issuer != nil {
		opts.Issuer = o.Issuer
	}
	opts.Recursively = o.Recursively
	opts.Update = o.Update
	opts.Verify = o.Verify
	opts.Keyless = o.Keyless
	if o.NormalizationAlgo != "" {
		opts.NormalizationAlgo = o.NormalizationAlgo
	}
	if o.TSAUrl != "" {
		opts.TSAUrl = o.TSAUrl
	}
	if o.UseTSA {
		opts.UseTSA = o.UseTSA
	}
}

// Complete takes either nil, an ocm.ContextProvider or a signing.Registry.
// To be compatible with an older version the type has been changed to interface
// to support multiple variants.
func (o *Options) Complete(ctx interface{}) error {
	var reg signing.Registry

	if ctx == nil {
		ctx = ocm.DefaultContext()
	}

	switch t := ctx.(type) {
	case ocm.ContextProvider:
		reg = signingattr.Get(t.OCMContext())
	case signing.Registry:
		reg = t
	default:
		return fmt.Errorf("context argument (%T) is invalid", ctx)
	}

	o.Printer = common.AssurePrinter(o.Printer)

	if o.Registry == nil {
		o.Registry = reg
	}

	o.effectiveRegistry = o.Registry
	if o.Keys != nil && (o.Keys.HasKeys() || o.Keys.HasIssuers() || o.Keys.HasRootCertificates()) {
		o.effectiveRegistry = signing.RegistryWithPreferredKeys(o.Registry, o.Keys)
	}

	if o.RootCerts == nil && o.effectiveRegistry.HasRootCertificates() {
		o.RootCerts = o.effectiveRegistry.GetRootCertPool(true)
	}

	if o.RootCerts != nil {
		// check root certificates
		pool, err := signutils.GetCertPool(o.RootCerts, false)
		if err != nil {
			return err
		}
		o.RootCerts = pool
	}

	if o.SkipAccessTypes == nil {
		o.SkipAccessTypes = map[string]bool{}
	}

	if o.Signer == nil && o.SignAlgo != "" {
		o.Signer = o.Registry.GetSigner(o.SignAlgo)
		if o.Signer == nil {
			return errors.ErrUnknown(compdesc.KIND_SIGN_ALGORITHM, o.SignAlgo)
		}
	}
	if o.Signer != nil {
		if len(o.SignatureNames) == 0 {
			return errors.Newf("signature name required for signing")
		}
		priv, err := o.PrivateKey()
		if err != nil {
			return err
		}
		if priv == nil && !o.Keyless {
			return errors.ErrNotFound(compdesc.KIND_PRIVATE_KEY, o.SignatureNames[0])
		}
		if o.DigestMode == "" {
			o.DigestMode = DIGESTMODE_LOCAL
		}
	}
	if !o.Keyless {
		if o.Signer != nil && !o.VerifySignature {
			if pub := o.PublicKey(o.SignatureName()); pub != nil {
				o.VerifySignature = true
				if err := o.checkCert(pub, o.IssuerFor(o.SignatureName())); err != nil {
					return fmt.Errorf("public key not valid: %w", err)
				}
			}
		} else if o.VerifySignature {
			for _, n := range o.SignatureNames {
				pub := o.PublicKey(n)
				// don't check for public key here, anymore,
				// because the key might be provided via certificate together with
				// the signature. An early failure is therefore not possible anymore.
				if pub != nil {
					if err := o.checkCert(pub, o.IssuerFor(n)); err != nil {
						return fmt.Errorf("public key not valid: %w", err)
					}
				}
			}
		}
	}
	if o.NormalizationAlgo == "" {
		o.NormalizationAlgo = compdesc.JsonNormalisationV1
	}

	if o.Hasher == nil && o.HashAlgo != "" {
		o.Hasher = o.Registry.GetHasher(o.HashAlgo)
		if o.Hasher == nil {
			return errors.ErrUnknown(compdesc.KIND_HASH_ALGORITHM, o.HashAlgo)
		}
	}
	if o.Hasher == nil {
		o.Hasher = o.Registry.GetHasher(sha256.Algorithm)
	}
	return nil
}

func (o *Options) checkCert(data interface{}, name *pkix.Name) error {
	cert, pool, err := signutils.GetCertificate(data, false)
	if err != nil {
		return nil
	}
	err = signing.VerifyCertDN(pool, o.RootCerts, name, cert)
	if err != nil {
		if name != nil {
			return errors.Wrapf(err, "issuer [%s]", name)
		}
		return err
	}
	return nil
}

func (o *Options) DoUpdate() bool {
	return o.Update || o.DoSign()
}

func (o *Options) DoSign() bool {
	return o.Signer != nil && len(o.SignatureNames) > 0
}

func (o *Options) StoreLocally() bool {
	return o.DigestMode == DIGESTMODE_LOCAL
}

func (o *Options) DoVerify() bool {
	return o.VerifySignature
}

func (o *Options) SignatureName() string {
	if len(o.SignatureNames) > 0 {
		return o.SignatureNames[0]
	}
	return ""
}

func (o *Options) GetIssuer() *pkix.Name {
	if o.Issuer != nil {
		return o.Issuer
	}
	if o.effectiveRegistry != nil {
		return o.effectiveRegistry.GetIssuer(o.SignatureName())
	}
	return nil
}

func (o *Options) IssuerFor(name string) *pkix.Name {
	if o.Issuer != nil && name == o.SignatureName() {
		return o.Issuer
	}
	if o.effectiveRegistry != nil {
		return o.effectiveRegistry.GetIssuer(name)
	}
	return nil
}

func (o *Options) SignatureConfigured(name string) bool {
	for _, n := range o.SignatureNames {
		if n == name {
			return true
		}
	}
	return false
}

func (o *Options) PublicKey(sig string) signutils.GenericPublicKey {
	return o.effectiveRegistry.GetPublicKey(sig)
}

func (o *Options) PrivateKey() (signutils.GenericPrivateKey, error) {
	return signing.ResolvePrivateKey(o.effectiveRegistry, o.SignatureName())
}

func (o *Options) EffectiveTSAUrl() string {
	if o.UseTSA {
		if o.TSAUrl != "" {
			return o.TSAUrl
		}
		return o.effectiveRegistry.TSAUrl()
	}
	return ""
}

func (o *Options) Dup() *Options {
	opts := *o
	return &opts
}

func (o *Options) Nested() *Options {
	opts := o.Dup()
	opts.VerifySignature = false // TODO: may be we want a mode to verify signature if present
	if !opts.Recursively {
		opts.Update = opts.DoUpdate() && opts.DigestMode == DIGESTMODE_LOCAL
		opts.Signer = nil
	}
	opts.Printer = opts.Printer.AddGap("  ")
	return opts
}

func (o *Options) StopRecursion() *Options {
	opts := *o
	opts.Recursively = false
	opts.Signer = nil
	opts.Update = false
	return &opts
}

func (o *Options) WithDigestMode(mode string) *Options {
	if mode == "" || o.DigestMode == mode {
		return o
	}
	opts := *o
	opts.DigestMode = mode
	return &opts
}

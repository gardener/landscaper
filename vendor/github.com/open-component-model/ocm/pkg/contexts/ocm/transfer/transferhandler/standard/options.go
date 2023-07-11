// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package standard

import (
	"golang.org/x/exp/slices"

	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/transfer/transferhandler"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

type Options struct {
	recursive        *bool
	resourcesByValue *bool
	localByValue     *bool
	sourcesByValue   *bool
	keepGlobalAccess *bool
	stopOnExisting   *bool
	overwrite        *bool
	omitAccessTypes  utils.StringSet
	resolver         ocm.ComponentVersionResolver
}

var (
	_ transferhandler.TransferOption = (*Options)(nil)

	_ ResourcesByValueOption      = (*Options)(nil)
	_ LocalResourcesByValueOption = (*Options)(nil)
	_ SourcesByValueOption        = (*Options)(nil)
	_ RecursiveOption             = (*Options)(nil)
	_ ResolverOption              = (*Options)(nil)
	_ KeepGlobalAccessOption      = (*Options)(nil)
	_ OmitAccessTypesOption       = (*Options)(nil)
)

func (o *Options) ApplyTransferOption(target transferhandler.TransferOptions) error {
	if o.recursive != nil {
		if opts, ok := target.(RecursiveOption); ok {
			opts.SetRecursive(*o.recursive)
		}
	}
	if o.resourcesByValue != nil {
		if opts, ok := target.(ResourcesByValueOption); ok {
			opts.SetResourcesByValue(*o.resourcesByValue)
		}
	}
	if o.localByValue != nil {
		if opts, ok := target.(LocalResourcesByValueOption); ok {
			opts.SetLocalResourcesByValue(*o.localByValue)
		}
	}
	if o.sourcesByValue != nil {
		if opts, ok := target.(SourcesByValueOption); ok {
			opts.SetSourcesByValue(*o.sourcesByValue)
		}
	}
	if o.keepGlobalAccess != nil {
		if opts, ok := target.(KeepGlobalAccessOption); ok {
			opts.SetKeepGlobalAccess(*o.keepGlobalAccess)
		}
	}
	if o.stopOnExisting != nil {
		if opts, ok := target.(StopOnExistingVersionOption); ok {
			opts.SetStopOnExistingVersion(*o.stopOnExisting)
		}
	}
	if o.overwrite != nil {
		if opts, ok := target.(OverwriteOption); ok {
			opts.SetOverwrite(*o.overwrite)
		}
	}
	if o.omitAccessTypes != nil {
		if opts, ok := target.(OmitAccessTypesOption); ok {
			opts.SetOmittedAccessTypes(utils.StringMapKeys(o.omitAccessTypes)...)
		}
	}
	if o.resolver != nil {
		if opts, ok := target.(ResolverOption); ok {
			opts.SetResolver(o.resolver)
		}
	}
	return nil
}

func (o *Options) Apply(opts ...transferhandler.TransferOption) error {
	return transferhandler.ApplyOptions(o, opts...)
}

func (o *Options) SetOverwrite(overwrite bool) {
	o.overwrite = &overwrite
}

func (o *Options) IsOverwrite() bool {
	return transferhandler.AsBool(o.overwrite)
}

func (o *Options) SetRecursive(recursive bool) {
	o.recursive = &recursive
}

func (o *Options) IsRecursive() bool {
	return transferhandler.AsBool(o.recursive)
}

func (o *Options) SetResourcesByValue(resourcesByValue bool) {
	o.resourcesByValue = &resourcesByValue
}

func (o *Options) IsResourcesByValue() bool {
	return transferhandler.AsBool(o.resourcesByValue)
}

func (o *Options) SetLocalResourcesByValue(resourcesByValue bool) {
	o.localByValue = &resourcesByValue
}

func (o *Options) IsLocalResourcesByValue() bool {
	return transferhandler.AsBool(o.localByValue)
}

func (o *Options) SetSourcesByValue(sourcesByValue bool) {
	o.sourcesByValue = &sourcesByValue
}

func (o *Options) IsSourcesByValue() bool {
	return transferhandler.AsBool(o.sourcesByValue)
}

func (o *Options) SetKeepGlobalAccess(keepGlobalAccess bool) {
	o.keepGlobalAccess = &keepGlobalAccess
}

func (o *Options) IsKeepGlobalAccess() bool {
	return transferhandler.AsBool(o.keepGlobalAccess)
}

func (o *Options) SetResolver(resolver ocm.ComponentVersionResolver) {
	o.resolver = resolver
}

func (o *Options) GetResolver() ocm.ComponentVersionResolver {
	return o.resolver
}

func (o *Options) SetStopOnExistingVersion(stopOnExistingVersion bool) {
	o.stopOnExisting = &stopOnExistingVersion
}

func (o *Options) IsStopOnExistingVersion() bool {
	return transferhandler.AsBool(o.stopOnExisting)
}

func (o *Options) SetOmittedAccessTypes(list ...string) {
	o.omitAccessTypes = utils.StringSet{}
	for _, t := range list {
		o.omitAccessTypes.Add(t)
	}
}

func (o *Options) GetOmittedAccessTypes() []string {
	if o.omitAccessTypes == nil {
		return nil
	}
	return utils.StringMapKeys(o.omitAccessTypes)
}

func (o *Options) IsAccessTypeOmitted(t string) bool {
	if o.omitAccessTypes == nil {
		return false
	}
	if o.omitAccessTypes.Contains(t) {
		return true
	}
	k, _ := runtime.KindVersion(t)
	return o.omitAccessTypes.Contains(k)
}

///////////////////////////////////////////////////////////////////////////////

type OverwriteOption interface {
	SetOverwrite(bool)
	IsOverwrite() bool
}

type overwriteOption struct {
	overwrite bool
}

func (o *overwriteOption) ApplyTransferOption(to transferhandler.TransferOptions) error {
	if eff, ok := to.(OverwriteOption); ok {
		eff.SetOverwrite(o.overwrite)
		return nil
	} else {
		return errors.ErrNotSupported("overwrite")
	}
}

func Overwrite(args ...bool) transferhandler.TransferOption {
	return &overwriteOption{
		overwrite: utils.GetOptionFlag(args...),
	}
}

///////////////////////////////////////////////////////////////////////////////

type RecursiveOption interface {
	SetRecursive(bool)
	IsRecursive() bool
}

type recursiveOption struct {
	recursive bool
}

func (o *recursiveOption) ApplyTransferOption(to transferhandler.TransferOptions) error {
	if eff, ok := to.(RecursiveOption); ok {
		eff.SetRecursive(o.recursive)
		return nil
	} else {
		return errors.ErrNotSupported("recursive")
	}
}

func Recursive(args ...bool) transferhandler.TransferOption {
	return &recursiveOption{
		recursive: utils.GetOptionFlag(args...),
	}
}

///////////////////////////////////////////////////////////////////////////////

type ResourcesByValueOption interface {
	SetResourcesByValue(bool)
	IsResourcesByValue() bool
}

type LocalResourcesByValueOption interface {
	SetLocalResourcesByValue(bool)
	IsLocalResourcesByValue() bool
}

type resourcesByValueOption struct {
	flag bool
}

func (o *resourcesByValueOption) ApplyTransferOption(to transferhandler.TransferOptions) error {
	if eff, ok := to.(ResourcesByValueOption); ok {
		eff.SetResourcesByValue(o.flag)
		return nil
	} else {
		return errors.ErrNotSupported("resources by-value")
	}
}

func ResourcesByValue(args ...bool) transferhandler.TransferOption {
	return &resourcesByValueOption{
		flag: utils.GetOptionFlag(args...),
	}
}

type intrscsByValueOption struct {
	flag bool
}

func (o *intrscsByValueOption) ApplyTransferOption(to transferhandler.TransferOptions) error {
	if eff, ok := to.(LocalResourcesByValueOption); ok {
		eff.SetLocalResourcesByValue(o.flag)
		return nil
	} else {
		return errors.ErrNotSupported("resources by-value")
	}
}

func LocalResourcesByValue(args ...bool) transferhandler.TransferOption {
	return &intrscsByValueOption{
		flag: utils.GetOptionFlag(args...),
	}
}

///////////////////////////////////////////////////////////////////////////////

type SourcesByValueOption interface {
	SetSourcesByValue(bool)
	IsSourcesByValue() bool
}

type sourcesByValueOption struct {
	flag bool
}

func (o *sourcesByValueOption) ApplyTransferOption(to transferhandler.TransferOptions) error {
	if eff, ok := to.(SourcesByValueOption); ok {
		eff.SetSourcesByValue(o.flag)
		return nil
	} else {
		return errors.ErrNotSupported("sources by-value")
	}
}

func SourcesByValue(args ...bool) transferhandler.TransferOption {
	return &sourcesByValueOption{
		flag: utils.GetOptionFlag(args...),
	}
}

///////////////////////////////////////////////////////////////////////////////

type ResolverOption interface {
	GetResolver() ocm.ComponentVersionResolver
	SetResolver(ocm.ComponentVersionResolver)
}

type resolverOption struct {
	resolver ocm.ComponentVersionResolver
}

func (o *resolverOption) ApplyTransferOption(to transferhandler.TransferOptions) error {
	if eff, ok := to.(ResolverOption); ok {
		eff.SetResolver(o.resolver)
		return nil
	} else {
		return errors.ErrNotSupported("resolver")
	}
}

func Resolver(resolver ocm.ComponentVersionResolver) transferhandler.TransferOption {
	return &resolverOption{
		resolver: resolver,
	}
}

///////////////////////////////////////////////////////////////////////////////

type KeepGlobalAccessOption interface {
	SetKeepGlobalAccess(bool)
	IsKeepGlobalAccess() bool
}

type keepGlobalOption struct {
	flag bool
}

func (o *keepGlobalOption) ApplyTransferOption(to transferhandler.TransferOptions) error {
	if eff, ok := to.(KeepGlobalAccessOption); ok {
		eff.SetKeepGlobalAccess(o.flag)
		return nil
	} else {
		return errors.ErrNotSupported("keep-global-access")
	}
}

func KeepGlobalAccess(args ...bool) transferhandler.TransferOption {
	return &keepGlobalOption{
		flag: utils.GetOptionFlag(args...),
	}
}

///////////////////////////////////////////////////////////////////////////////

type StopOnExistingVersionOption interface {
	SetStopOnExistingVersion(bool)
	IsStopOnExistingVersion() bool
}

type stopOnExistingVersionOption struct {
	flag bool
}

func (o *stopOnExistingVersionOption) ApplyTransferOption(to transferhandler.TransferOptions) error {
	if eff, ok := to.(StopOnExistingVersionOption); ok {
		eff.SetStopOnExistingVersion(o.flag)
		return nil
	} else {
		return errors.ErrNotSupported("stop-on-existing")
	}
}

func StopOnExistingVersion(args ...bool) transferhandler.TransferOption {
	return &stopOnExistingVersionOption{
		flag: utils.GetOptionFlag(args...),
	}
}

///////////////////////////////////////////////////////////////////////////////

type OmitAccessTypesOption interface {
	SetOmittedAccessTypes(...string)
	GetOmittedAccessTypes() []string
}

type omitAccessTypesOption struct {
	add  bool
	list []string
}

func (o *omitAccessTypesOption) ApplyTransferOption(to transferhandler.TransferOptions) error {
	if eff, ok := to.(OmitAccessTypesOption); ok {
		if o.add {
			eff.SetOmittedAccessTypes(append(eff.GetOmittedAccessTypes(), o.list...)...)
		} else {
			eff.SetOmittedAccessTypes(o.list...)
		}
		return nil
	} else {
		return errors.ErrNotSupported("omit-access-types")
	}
}

func OmitAccessTypes(list ...string) transferhandler.TransferOption {
	return &omitAccessTypesOption{
		list: slices.Clone(list),
	}
}

func AddOmittedAccessTypes(list ...string) transferhandler.TransferOption {
	return &omitAccessTypesOption{
		add:  true,
		list: slices.Clone(list),
	}
}

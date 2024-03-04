// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/signingattr"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/signing/handlers/rsa"
)

func SignComponentVersion(cv ocm.ComponentVersionAccess, name string, optlist ...Option) (*metav1.DigestSpec, error) {
	var opts Options

	opts.Eval(
		SignatureName(name),
		Update(),
		Recursive(),
		VerifyDigests(),
	)
	opts.Eval(optlist...)

	if opts.VerifySignature {
		return nil, errors.Newf("impossible verification option set for signing")
	}
	if opts.Signer == nil {
		opts.Signer = signingattr.Get(cv.GetContext()).GetSigner(rsa.Algorithm)
	}
	err := opts.Complete(cv.GetContext())
	if err != nil {
		return nil, errors.Wrapf(err, "inconsistent options for signing")
	}
	return Apply(nil, nil, cv, &opts)
}

func VerifyComponentVersion(cv ocm.ComponentVersionAccess, name string, optlist ...Option) (*metav1.DigestSpec, error) {
	var opts Options
	if len(cv.GetDescriptor().Signatures) == 1 && name == "" {
		name = cv.GetDescriptor().Signatures[0].Name
	}

	opts.Eval(
		VerifyDigests(),
		VerifySignature(name),
		Recursive(),
	)
	opts.Eval(optlist...)

	if opts.Signer != nil {
		return nil, errors.Newf("impossible signer option set for verification")
	}
	err := opts.Complete(cv.GetContext())
	if err != nil {
		return nil, errors.Wrapf(err, "inconsistent options for verification")
	}
	return Apply(nil, nil, cv, &opts)
}

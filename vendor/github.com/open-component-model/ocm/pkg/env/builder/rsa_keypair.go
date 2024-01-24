// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"github.com/mandelsoft/filepath/pkg/filepath"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/signingattr"
	"github.com/open-component-model/ocm/pkg/signing/handlers/rsa"
	"github.com/open-component-model/ocm/pkg/signing/signutils"
	"github.com/open-component-model/ocm/pkg/utils"
)

// TODO: switch to context local setting.
func (b *Builder) RSAKeyPair(name ...string) {
	priv, pub, err := rsa.Handler{}.CreateKeyPair()
	b.failOn(err)
	reg := signingattr.Get(b.OCMContext())
	for _, n := range name {
		reg.RegisterPublicKey(n, pub)
		reg.RegisterPrivateKey(n, priv)
	}
}

func (b *Builder) ReadRSAKeyPair(name, path string) {
	reg := signingattr.Get(b.OCMContext())
	pubfound := false
	path, _ = utils.ResolvePath(path)
	if ok, _ := b.Exists(filepath.Join(path, "rsa.pub")); ok {
		pubbytes, err := b.ReadFile(filepath.Join(path, "rsa.pub"))
		b.failOn(err)
		pub, err := signutils.ParsePublicKey(pubbytes)
		b.failOn(err)
		reg.RegisterPublicKey(name, pub)
		pubfound = true
	}
	if ok, _ := b.Exists(filepath.Join(path, "rsa.priv")); ok {
		privbytes, err := b.ReadFile(filepath.Join(path, "rsa.priv"))
		b.failOn(err)
		priv, err := signutils.ParsePrivateKey(privbytes)
		b.failOn(err)
		reg.RegisterPrivateKey(name, priv)
		if !pubfound {
			pub, _, err := rsa.GetPublicKey(priv)
			b.failOn(err)
			reg.RegisterPublicKey(name, pub)
		}
	}
}

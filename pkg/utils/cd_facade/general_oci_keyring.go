package cd_facade

import (
	dockercreds "github.com/docker/cli/cli/config/credentials"
	"github.com/gardener/component-cli/ociclient/credentials"
)

type GeneralOciKeyring struct {
	keyring *credentials.GeneralOciKeyring
}

func New() *GeneralOciKeyring {
	return &GeneralOciKeyring{
		keyring: credentials.New(),
	}
}

func (r *GeneralOciKeyring) Get(resourceURl string) Auth {
	return r.keyring.Get(resourceURl)
}

func (r *GeneralOciKeyring) Add(store dockercreds.Store) error {
	return r.keyring.Add(store)
}

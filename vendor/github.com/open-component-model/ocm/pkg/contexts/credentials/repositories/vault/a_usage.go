// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package vault

import (
	"strings"

	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/vault/identity"
	"github.com/open-component-model/ocm/pkg/listformat"
)

func init() {
	info := cpi.DefaultContext.ConsumerIdentityMatchers().GetInfo(identity.CONSUMER_TYPE)
	idx := strings.Index(info.Description, "\n")
	desc := `
This repository type can be used to access credentials stored in a HashiCorp
Vault. 

It provides access to list of secrets stored under a dedicated path in
a vault namespace. This list can either explicitly be specified, or
it is taken from the metadata of a specified secret.

The following custom metadata attributes are evaluated:
- <code>` + CUSTOM_SECRETS + `</code> this attribute may contain a comma separated list of
  vault secrets, which should be exposed by this repository instance.
  The names are evaluated under the path prefix used for the repository.
- <code>` + CUSTOM_CONSUMERID + `</code> this attribute may contain a JSON encoded
  consumer id , this secret should be assigned to.
- <code>type</code> if no special attribute is defined this attribute 
  indicated to use the complete custom metadata as consumer id.

It uses the ` + identity.CONSUMER_TYPE + ` identity matcher and consumer type
to requests credentials for the access.
` + info.Description[idx:] + `

It requires the following credential attributes:

` + info.CredentialAttributes

	usage = desc
}

var usage string

var format = `
The repository specification supports the following fields:
` + listformat.FormatListElements("", listformat.StringElementDescriptionList{
	"serverURL", "*string* (required): the URL of the vault instance",
	"namespace", "*string* (optional): the namespace used to evaluate secrets",
	"secretsEngine", "*string* (optional): the secrets engine to use (default: secrets)",
	"path", "*string* (optional): the path prefix used to lookup secrets",
	"secrets", "*[]string* (optional): list of secrets",
	"propagateConsumerIdentity", "*bool*(optional): evaluate metadata for consumer id propagation",
}) + `
If the secrets list is empty, all secret entries found in the given path
is read.
`

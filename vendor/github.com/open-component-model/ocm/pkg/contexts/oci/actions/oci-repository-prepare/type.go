// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package oci_repository_prepare

import (
	"path"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/oci/identity"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/action/api"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

const Type = "oci.repository.prepare"

func init() {
	api.RegisterAction(Type, "Prepare the usage of a repository in an OCI registry.", usage,
		[]string{identity.ID_HOSTNAME, identity.ID_PORT, identity.ID_PATHPREFIX})

	api.RegisterType(api.NewActionType[*ActionSpecV1, *ActionResultV1](Type, "v1"))
}

var usage = `
The hostname of the target repository is used as selector. The action should
assure, that the requested repository is available on the target OCI registry.

Spec version v1 uses the following specification fields:
- <code>hostname</code> *string*: The  hostname of the OCI registry.
- <code>repository</code> *string*: The OCI repository name.
`

////////////////////////////////////////////////////////////////////////////////
// internal version

type ActionSpec = ActionSpecV1

type ActionResult = ActionResultV1

func Spec(host string, repo string) *ActionSpec {
	return &ActionSpec{
		ObjectVersionedType: runtime.ObjectVersionedType{runtime.TypeName(Type, "v1")},
		Hostname:            host,
		Repository:          repo,
	}
}

func Result(msg string) *ActionResult {
	return &ActionResult{
		CommonResult: api.CommonResult{
			ObjectVersionedType: runtime.ObjectVersionedType{runtime.TypeName(Type, "v1")},
			Message:             msg,
		},
	}
}

////////////////////////////////////////////////////////////////////////////////
// serialization formats

type ActionSpecV1 struct {
	runtime.ObjectVersionedType
	Hostname   string `json:"hostname"`
	Repository string `json:"repository"`
}

func (s *ActionSpecV1) Selector() api.Selector {
	return api.Selector(s.Hostname)
}

func (s *ActionSpecV1) GetConsumerAttributes() common.Properties {
	host, port, base := utils.SplitLocator(s.Hostname)
	return common.Properties{
		cpi.ID_TYPE:            identity.CONSUMER_TYPE,
		identity.ID_HOSTNAME:   host,
		identity.ID_PATHPREFIX: path.Join(base, s.Repository),
		identity.ID_PORT:       port,
	}
}

type ActionResultV1 struct {
	api.CommonResult `json:",inline"`
}

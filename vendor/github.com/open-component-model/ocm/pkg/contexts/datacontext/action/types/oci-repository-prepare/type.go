// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package oci_repository_prepare

import (
	"path"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/action/api"
	"github.com/open-component-model/ocm/pkg/contexts/oci/identity"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

const Type = "oci.repository.prepare"

func init() {
	api.RegisterAction(Type, &ActionSpec{}, &ActionResult{}, "Prepare the usage of a repository in an OCI registry.",
		identity.ID_HOSTNAME, identity.ID_PORT, identity.ID_PATHPREFIX)

	api.RegisterType(Type, "v1", api.NewActionTypeByProtoTypes(&ActionSpecV1{}, nil, &ActionResultV1{}, nil))
}

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
		identity.ID_TYPE:       identity.CONSUMER_TYPE,
		identity.ID_HOSTNAME:   host,
		identity.ID_PATHPREFIX: path.Join(base, s.Repository),
		identity.ID_PORT:       port,
	}
}

type ActionResultV1 struct {
	api.CommonResult `json:",inline"`
}

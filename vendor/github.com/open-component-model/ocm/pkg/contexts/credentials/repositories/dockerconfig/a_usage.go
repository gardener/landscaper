// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dockerconfig

import (
	"github.com/open-component-model/ocm/pkg/listformat"
)

var usage = `
This repository type can be used to access credentials stored in a file
following the docker config json format. It take into account the
credentials helper section, also. If enabled, the described
credentials will be automatically assigned to appropriate consumer ids.
`

var format = `The repository specification supports the following fields:
` + listformat.FormatListElements("", listformat.StringElementDescriptionList{
	"dockerConfigFile", "*string*: the file path to a docker config file",
	"dockerConfig", "*json*: an embedded docker config json",
	"propagateConsumerIdentity", "*bool*(optional): enable consumer id propagation",
})

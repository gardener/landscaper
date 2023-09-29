// Copyright 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package rsa_signingservice

import (
	"github.com/open-component-model/ocm/pkg/contexts/credentials/identity/hostpath"
)

const (
	CONSUMER_TYPE = "Signingserver.gardener.cloud"

	ID_HOSTNAME   = hostpath.ID_HOSTNAME
	ID_PORT       = hostpath.ID_PORT
	ID_PATHPREFIX = hostpath.ID_PATHPREFIX
	ID_SCHEME     = hostpath.ID_SCHEME
	CLIENT_CERT   = "clientCert"
	PRIVATE_KEY   = "privateKey"
	CA_CERTS      = "caCerts"
)

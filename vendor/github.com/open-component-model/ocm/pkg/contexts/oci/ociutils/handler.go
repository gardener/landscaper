// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociutils

import (
	"sync"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
)

type InfoHandler interface {
	Description(pr common.Printer, m cpi.ManifestAccess, config []byte)
	Info(m cpi.ManifestAccess, config []byte) interface{}
}

var (
	lock     sync.Mutex
	handlers = map[string]InfoHandler{}
)

func RegisterInfoHandler(mime string, h InfoHandler) {
	lock.Lock()
	defer lock.Unlock()
	handlers[mime] = h
}

func getHandler(mime string) InfoHandler {
	lock.Lock()
	defer lock.Unlock()
	return handlers[mime]
}

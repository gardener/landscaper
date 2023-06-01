// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"io"
	"sync"

	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
)

type ConfigType string

const (
	ContainerRegistry ConfigType = "container_registry"
)

type Credential interface {
	Name() string
	ConsumerIdentity() cpi.ConsumerIdentity
	Properties() cpi.Credentials
}

type Handler interface {
	ConfigType() ConfigType
	ParseConfig(io.Reader) ([]Credential, error)
}

var (
	handlers = map[ConfigType]Handler{}
	lock     sync.RWMutex
)

func RegisterHandler(h Handler) {
	lock.Lock()
	defer lock.Unlock()
	handlers[h.ConfigType()] = h
}

func GetHandler(configType ConfigType) Handler {
	lock.RLock()
	defer lock.RUnlock()
	return handlers[configType]
}

func GetHandlers() map[ConfigType]Handler {
	lock.RLock()
	defer lock.RUnlock()

	m := map[ConfigType]Handler{}
	for k, v := range handlers {
		m[k] = v
	}
	return m
}

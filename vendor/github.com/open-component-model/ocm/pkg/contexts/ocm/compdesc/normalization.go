// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compdesc

import (
	"sort"
	"sync"

	"github.com/open-component-model/ocm/pkg/errors"
)

// NormalisationAlgorithm types and versions the algorithm used for digest generation.
type NormalisationAlgorithm = string

const (
	JsonNormalisationV1 NormalisationAlgorithm = "jsonNormalisation/v1"
	JsonNormalisationV2 NormalisationAlgorithm = "jsonNormalisation/v2"
)

type Normalization interface {
	Normalize(cd *ComponentDescriptor) ([]byte, error)
}

type NormalizationAlgorithms struct {
	sync.RWMutex
	algos map[string]Normalization
}

func (n *NormalizationAlgorithms) Register(name string, norm Normalization) {
	n.Lock()
	defer n.Unlock()
	n.algos[name] = norm
}

func (n *NormalizationAlgorithms) Get(algo string) Normalization {
	n.RLock()
	defer n.RUnlock()
	return n.algos[algo]
}

func (n *NormalizationAlgorithms) Names() []string {
	n.RLock()
	defer n.RUnlock()
	names := []string{}
	for n := range n.algos {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (n *NormalizationAlgorithms) Normalize(cd *ComponentDescriptor, algo string) ([]byte, error) {
	n.RLock()
	defer n.RUnlock()

	norm := n.algos[algo]
	if norm == nil {
		return nil, errors.ErrUnknown("normalization algorithm", algo)
	}
	return norm.Normalize(cd)
}

var Normalizations = NormalizationAlgorithms{algos: map[string]Normalization{}}

func Normalize(cd *ComponentDescriptor, normAlgo string) ([]byte, error) {
	DefaultComponent(cd)
	return Normalizations.Normalize(cd, normAlgo)
}

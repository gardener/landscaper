// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"encoding/json"
)

type meta struct {
	Version string `json:"version"`
}

type transportConfig struct {
	Meta            meta                       `json:"meta"`
	Uploaders       []uploaderDefinition       `json:"uploaders"`
	Processors      []processorDefinition      `json:"processors"`
	Downloaders     []downloaderDefinition     `json:"downloaders"`
	ProcessingRules []processingRuleDefinition `json:"processingRules"`
}

type baseProcessorDefinition struct {
	Name string           `json:"name"`
	Type string           `json:"type"`
	Spec *json.RawMessage `json:"spec"`
}

type filterDefinition struct {
	Type string           `json:"type"`
	Spec *json.RawMessage `json:"spec"`
}

type downloaderDefinition struct {
	baseProcessorDefinition
	Filters []filterDefinition `json:"filters"`
}

type uploaderDefinition struct {
	baseProcessorDefinition
	Filters []filterDefinition `json:"filters"`
}

type processorDefinition struct {
	baseProcessorDefinition
}

type processorReference struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type processingRuleDefinition struct {
	Name       string
	Filters    []filterDefinition   `json:"filters"`
	Processors []processorReference `json:"processors"`
}

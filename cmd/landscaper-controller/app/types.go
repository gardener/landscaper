// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import "k8s.io/apimachinery/pkg/runtime"

// DeployersConfiguration describes additional configuration for deployers
type DeployersConfiguration struct {
	Deployers map[string]*runtime.RawExtension `json:"deployers"`
}

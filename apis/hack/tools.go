// +build tools

// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	_ "k8s.io/code-generator"
	_ "k8s.io/kube-openapi/cmd/openapi-gen"
)

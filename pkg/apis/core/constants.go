// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package core

const (
	ImportConfigEnvVarName = "IMPORT_CONFIG"

	ImportConfigPath = "/landscaper/import.yaml"

	// Annotations

	// Operations
	OperationAnnotation = "landscaper.gardener.cloud/operation"

	OperationReconcile = "reconcile"

	// InlineComponentDescriptorLabel is the label name used for nested inline component descriptors
	InlineComponentDescriptorLabel = "landscaper.gardener.cloud/component-descriptor"
)

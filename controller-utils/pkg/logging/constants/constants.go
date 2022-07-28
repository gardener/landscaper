// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package constants

const (
	// KeyReconciledResource is for 'namespace/name' of the resource which is currently being reconciled.
	KeyReconciledResource = "reconciledResource"
	// KeyReconciledResourceKind is for the kind of the resource which is being reconciled by this controller.
	KeyReconciledResourceKind = "reconciledResourceKind"
	// KeyOperation is for the current operation
	KeyOperation = "operation"
	// KeyMethod is for the currently executed method
	KeyMethod = "method"

	// MsgStartReconcile is the message which is displayed at the beginning of a new reconcile loop.
	MsgStartReconcile = "Starting reconcile"
	// MsgStartMethod is a message for logging the beginning of a method
	MsgStartMethod = "Starting method"
)

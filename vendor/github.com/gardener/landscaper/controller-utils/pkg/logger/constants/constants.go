// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package constants

const (
	// KeyReconciledResource points to 'namespace/name' of the resource which is currently being reconciled.
	KeyReconciledResource = "reconciledResource"

	// MsgStartReconcile is the message which is displayed at the beginning of a new reconcile loop.
	MsgStartReconcile = "Start reconcile"
)

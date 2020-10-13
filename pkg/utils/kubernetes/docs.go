// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

//go:generate mockgen -destination=./mock/client_mock.go sigs.k8s.io/controller-runtime/pkg/client Client,StatusWriter

package kubernetes

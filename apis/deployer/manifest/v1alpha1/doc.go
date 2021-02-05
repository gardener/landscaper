// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

// Package core is the internal version of the API.
// +k8s:deepcopy-gen=package,register
// +k8s:conversion-gen=github.com/gardener/landscaper/apis/deployer/manifest
// +k8s:openapi-gen=true
// +k8s:defaulter-gen=TypeMeta

// +groupName=manifest.deployer.landscaper.gardener.cloud
package v1alpha1

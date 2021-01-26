// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

//go:generate ../hack/generate-code.sh
//go:generate controller-gen output:crd:artifacts:config=../charts/landscaper/templates/crd crd:crdVersions=v1 object:headerFile="../hack/boilerplate.go.txt" paths="./core/v1alpha1"

package apis

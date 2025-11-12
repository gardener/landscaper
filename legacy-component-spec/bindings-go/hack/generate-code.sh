#!/bin/bash
#
# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

PROJECT_MOD_ROOT="github.com/gardener/landscaper/legacy-component-spec/bindings-go"

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

export GOFLAGS=-mod=vendor

echo "> Installing deepcopy-gen"
go install "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/cmd/deepcopy-gen

echo "> Generating deepcopy functions for Component Descriptor"

pushd ${PROJECT_ROOT}
"${GOPATH}"/bin/deepcopy-gen -i ./apis/v2 -O zz_generated_deepcopy --go-header-file ./hack/boilerplate.go.txt
popd
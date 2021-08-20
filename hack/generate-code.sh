#!/bin/bash
#
# Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

rm -f ${GOPATH}/bin/deepcopy-gen
rm -f ${GOPATH}/bin/defaulter-gen
rm -f ${GOPATH}/bin/conversion-gen

PROJECT_MOD_ROOT="github.com/gardener/landscaper"

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..
API_PROJECT_ROOT="${PROJECT_ROOT}"/apis

chmod +x ${API_PROJECT_ROOT}/vendor/k8s.io/code-generator/*

export GOFLAGS=-mod=vendor

echo "> Generating groups for Landscaper"
bash "${API_PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  deepcopy,defaulter,conversion \
  $PROJECT_MOD_ROOT/pkg/client \
  $PROJECT_MOD_ROOT/apis \
  $PROJECT_MOD_ROOT/apis \
  "core:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt"

echo "> Generating groups for Config"
bash "${API_PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  deepcopy,defaulter,conversion \
  $PROJECT_MOD_ROOT/pkg/client \
  $PROJECT_MOD_ROOT/apis \
  $PROJECT_MOD_ROOT/apis \
  "config:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt"

echo "> Generating groups for Deployers"
bash "${API_PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  deepcopy,defaulter,conversion \
  $PROJECT_MOD_ROOT/pkg/client \
  $PROJECT_MOD_ROOT/apis/deployer \
  $PROJECT_MOD_ROOT/apis/deployer \
  "utils/continuousreconcile:v1alpha1 utils/readinesschecks utils/managedresource helm:v1alpha1 container:v1alpha1 manifest:v1alpha1 manifest:v1alpha2 mock:v1alpha1 core:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt"

echo "> Generating openapi definitions"
go install "${API_PROJECT_ROOT}"/vendor/k8s.io/kube-openapi/cmd/openapi-gen
${GOPATH}/bin/openapi-gen "$@" \
  --v 1 \
  --logtostderr \
  --input-dirs=github.com/gardener/landscaper/apis/core/v1alpha1 \
  --input-dirs=github.com/gardener/landscaper/apis/config/v1alpha1 \
  --input-dirs=github.com/gardener/landscaper/apis/config \
  --input-dirs=github.com/gardener/landscaper/apis/deployer/core/v1alpha1 \
  --input-dirs=github.com/gardener/landscaper/apis/deployer/utils/readinesschecks \
  --input-dirs=github.com/gardener/landscaper/apis/deployer/utils/managedresource \
  --input-dirs=github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile/v1alpha1 \
  --input-dirs=github.com/gardener/landscaper/apis/deployer/helm/v1alpha1 \
  --input-dirs=github.com/gardener/landscaper/apis/deployer/manifest/v1alpha1 \
  --input-dirs=github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2 \
  --input-dirs=github.com/gardener/landscaper/apis/deployer/container/v1alpha1 \
  --input-dirs=github.com/gardener/landscaper/apis/deployer/mock/v1alpha1 \
  --input-dirs=github.com/gardener/component-spec/bindings-go/apis/v2 \
  --input-dirs=k8s.io/api/core/v1 \
  --input-dirs=k8s.io/apimachinery/pkg/apis/meta/v1 \
  --input-dirs=k8s.io/apimachinery/pkg/api/resource \
  --input-dirs=k8s.io/apimachinery/pkg/types \
  --input-dirs=k8s.io/apimachinery/pkg/runtime \
  --report-filename=${API_PROJECT_ROOT}/openapi/api_violations.report \
  --output-package=github.com/gardener/landscaper/apis/openapi \
  -h "${PROJECT_ROOT}/hack/boilerplate.go.txt"

echo
echo "NOTE: If you changed the API then consider updating the example manifests."

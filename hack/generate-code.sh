#!/bin/bash
#
# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

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
  "helm:v1alpha1 container:v1alpha1 manifest:v1alpha1 manifest:v1alpha2 mock:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt"

echo
echo "NOTE: If you changed the API then consider updating the example manifests."
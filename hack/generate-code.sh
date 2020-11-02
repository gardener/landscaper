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

chmod +x ${PROJECT_ROOT}/vendor/k8s.io/code-generator/*

echo "> Generating groups for Landscaper"
bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  deepcopy,defaulter,conversion \
  $PROJECT_MOD_ROOT/pkg/client \
  $PROJECT_MOD_ROOT/pkg/apis \
  $PROJECT_MOD_ROOT/pkg/apis \
  "core:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt"

echo "> Generating groups for Config"
bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  deepcopy,defaulter,conversion \
  $PROJECT_MOD_ROOT/pkg/client \
  $PROJECT_MOD_ROOT/pkg/apis \
  $PROJECT_MOD_ROOT/pkg/apis \
  "config:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt"

echo "> Generating groups for Deployers"
bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  deepcopy,defaulter,conversion \
  $PROJECT_MOD_ROOT/pkg/client \
  $PROJECT_MOD_ROOT/pkg/apis/deployer \
  $PROJECT_MOD_ROOT/pkg/apis/deployer \
  "helm:v1alpha1 container:v1alpha1 manifest:v1alpha1 manifest:v1alpha2 mock:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt"

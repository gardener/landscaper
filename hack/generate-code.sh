#!/bin/bash
#
# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

PROJECT_MOD_ROOT="github.com/gardener/landscaper"

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

chmod +x ${PROJECT_ROOT}/vendor/k8s.io/code-generator/*

echo "> Generating groups for Landscaper"
bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  deepcopy,defaulter \
  $PROJECT_MOD_ROOT/pkg/client \
  $PROJECT_MOD_ROOT/pkg/apis \
  $PROJECT_MOD_ROOT/pkg/apis \
  "core:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt"

echo "> Generating conversion Landscaper"
bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  conversion \
  $PROJECT_MOD_ROOT/pkg/core \
  $PROJECT_MOD_ROOT/pkg/apis \
  $PROJECT_MOD_ROOT/pkg/apis \
  core:v1alpha1 \
  --extra-peer-dirs=github.com/gardener/landscaper/pkg/apis/core,github.com/gardener/landscaper/pkg/apis/core/v1alpha1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/conversion,k8s.io/apimachinery/pkg/runtime \
  --go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt"
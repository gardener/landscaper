#!/bin/bash -eu
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

PROJECT_ROOT="$(realpath $(dirname $0)/..)"

function revendor() {
  go mod tidy
}

echo "Revendor legacy-component-spec/bindings-go module ..."
(
  cd "$PROJECT_ROOT/legacy-component-spec/bindings-go"
  revendor
)
echo "Revendor legacy-image-vector module ..."
(
  cd "$PROJECT_ROOT/legacy-image-vector"
  revendor
)
echo "Revendor legacy-component-cli module ..."
(
  cd "$PROJECT_ROOT/legacy-component-cli"
  revendor
)
echo "Revendor apis module ..."
(
  cd "$PROJECT_ROOT/apis"
  revendor
)
echo "Revendor controller-utils module ..."
(
  cd "$PROJECT_ROOT/controller-utils"
  revendor
)
echo "Revendor root module ..."
(
  cd "$PROJECT_ROOT"
  revendor
)

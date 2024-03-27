#!/bin/bash -eu
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

PROJECT_ROOT="$(realpath $(dirname $0)/..)"

function revendor() {
  go mod tidy
}

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

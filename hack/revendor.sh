#!/bin/bash -eu
#
# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

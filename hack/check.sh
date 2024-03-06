#!/bin/bash
#
# Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

PROJECT_ROOT="$(realpath $(dirname $0)/..)"

if [[ -z ${LOCALBIN:-} ]]; then
  LOCALBIN="$PROJECT_ROOT/bin"
fi
if [[ -z ${LINTER:-} ]]; then
  LINTER="$LOCALBIN/golangci-lint"
fi

GOLANGCI_LINT_CONFIG_FILE=""

landscaper_module_paths=()
apis_module_paths=()
controller_utils_module_paths=()
for arg in "$@"; do
  case $arg in
    --golangci-lint-config=*)
      GOLANGCI_LINT_CONFIG_FILE="-c ${arg#*=}"
      shift
      ;;
    $PROJECT_ROOT/apis/*)
      apis_module_paths+=("./$(realpath "--relative-base=$PROJECT_ROOT/apis" "$arg")")
      ;;
    $PROJECT_ROOT/controller-utils/*)
      controller_utils_module_paths+=("./$(realpath "--relative-base=$PROJECT_ROOT/controller-utils" "$arg")")
      ;;
    *)
      landscaper_module_paths+=("./$(realpath "--relative-base=$PROJECT_ROOT" "$arg")")
      ;;
  esac
done

echo "> Check"

echo "apis module: ${apis_module_paths[@]}"
(
  cd "$PROJECT_ROOT/apis"
  echo "  Executing golangci-lint"
  "$LINTER" run $GOLANGCI_LINT_CONFIG_FILE --timeout 10m "${apis_module_paths[@]}"
  echo "  Executing go vet"
  go vet "${apis_module_paths[@]}"
)

echo "controller-utils module: ${controller_utils_module_paths[@]}"
(
  cd "$PROJECT_ROOT/controller-utils"
  echo "  Executing golangci-lint"
  "$LINTER" run $GOLANGCI_LINT_CONFIG_FILE --timeout 10m "${controller_utils_module_paths[@]}"
  echo "  Executing go vet"
  go vet "${controller_utils_module_paths[@]}"
)

echo "root module: ${landscaper_module_paths[@]}"
echo "  Executing golangci-lint"
"$LINTER" run $GOLANGCI_LINT_CONFIG_FILE --timeout 10m "${landscaper_module_paths[@]}"
echo "  Executing go vet"
go vet "${landscaper_module_paths[@]}"

if [[ ${SKIP_FORMATTING_CHECK:-"false"} == "false" ]]; then
  echo "Checking formatting"
  "$PROJECT_ROOT/hack/format.sh" --verify "$@"
fi

echo "All checks successful"
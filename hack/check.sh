#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
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
legacy_component_cli_paths=()
legacy_component_spec_paths=()
legacy_image_vector_paths=()
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
      controller_utils_paths+=("./$(realpath "--relative-base=$PROJECT_ROOT/controller-utils" "$arg")")
      ;;
   $PROJECT_ROOT/legacy-component-cli/*)
      legacy_component_cli_paths+=("./$(realpath "--relative-base=$PROJECT_ROOT/legacy-component-cli" "$arg")")
      ;;
   $PROJECT_ROOT/legacy-component-spec/bindings-go/*)
      legacy_component_spec_paths+=("./$(realpath "--relative-base=$PROJECT_ROOT/legacy-component-spec/bindings-go" "$arg")")
      ;;
   $PROJECT_ROOT/legacy-image-vector/*)
      legacy_image_vector_paths+=("./$(realpath "--relative-base=$PROJECT_ROOT/legacy-image-vector" "$arg")")
      ;;
    *)
      landscaper_module_paths+=("./$(realpath "--relative-base=$PROJECT_ROOT" "$arg")")
      ;;
  esac
done

echo "> Check"

echo "legacy-component-cli module: ${legacy_component_cli_paths[@]}"
(
  cd "$PROJECT_ROOT/legacy-component-cli"
  echo "  Executing golangci-lint"
  "$LINTER" run $GOLANGCI_LINT_CONFIG_FILE --timeout 10m "${legacy_component_cli_paths[@]}"
  echo "  Executing go vet"
  go vet "${legacy_component_cli_paths[@]}"
)

echo "legacy-component-spec module: ${legacy_component_spec_paths[@]}"
(
  cd "$PROJECT_ROOT/legacy-component-spec/bindings-go"
  echo "  Executing golangci-lint"
  "$LINTER" run $GOLANGCI_LINT_CONFIG_FILE --timeout 10m "${legacy_component_spec_paths[@]}"
  echo "  Executing go vet"
  go vet "${legacy_component_spec_paths[@]}"
)

echo "legacy-image-vector module: ${legacy_image_vector_paths[@]}"
(
  cd "$PROJECT_ROOT/legacy-image-vector"
  echo "  Executing golangci-lint"
  "$LINTER" run $GOLANGCI_LINT_CONFIG_FILE --timeout 10m "${legacy_image_vector_paths[@]}"
  echo "  Executing go vet"
  go vet "${legacy_image_vector_paths[@]}"
)

echo "apis module: ${apis_module_paths[@]}"
(
  cd "$PROJECT_ROOT/apis"
  echo "  Executing golangci-lint"
  "$LINTER" run $GOLANGCI_LINT_CONFIG_FILE --timeout 10m "${apis_module_paths[@]}"
  echo "  Executing go vet"
  go vet "${apis_module_paths[@]}"
)

echo "controller-utils module: ${controller_utils_paths[@]}"
(
  cd "$PROJECT_ROOT/controller-utils"
  echo "  Executing golangci-lint"
  "$LINTER" run $GOLANGCI_LINT_CONFIG_FILE --timeout 10m "${controller_utils_paths[@]}"
  echo "  Executing go vet"
  go vet "${controller_utils_paths[@]}"
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
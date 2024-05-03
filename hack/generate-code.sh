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
if [[ -z ${CODE_GEN_SCRIPT:-} ]]; then
  CODE_GEN_SCRIPT="$LOCALBIN/kube_codegen.sh"
fi
if [[ -z ${CONTROLLER_GEN:-} ]]; then
  CONTROLLER_GEN="$LOCALBIN/controller-gen"
fi
if [[ -z ${API_REF_GEN:-} ]]; then
  API_REF_GEN="$LOCALBIN/crd-ref-docs"
fi
if [[ -z ${MOCKGEN:-} ]]; then
  MOCKGEN="$LOCALBIN/mockgen"
fi
LANDSCAPER_MODULE_PATH="github.com/gardener/landscaper"
API_MODULE_PATH="${LANDSCAPER_MODULE_PATH}"/apis

# Code generation expects this repo to lie under <whatever>/github.com/gardener/landscaper, so let's verify that this is the case.
src_path="$(realpath "$PROJECT_ROOT")"
for parent in $(tr '/' '\n' <<< $LANDSCAPER_MODULE_PATH | tac); do
  if [[ "$src_path" != */$parent ]]; then
    echo "error: code generation expects the landscaper repository to be contained into a folder structure matching its module path '$LANDSCAPER_MODULE_PATH'"
    echo "expected path element: $parent"
    echo "actual path element: ${src_path##*/}"
    exit 1
  fi
  src_path="${src_path%/$parent}"
done

rm -f ${GOPATH}/bin/deepcopy-gen
rm -f ${GOPATH}/bin/defaulter-gen
rm -f ${GOPATH}/bin/conversion-gen
rm -f ${GOPATH}/bin/openapi-gen

source "$CODE_GEN_SCRIPT"

echo "> Generating deepcopy/conversion/defaulter functions"
kube::codegen::gen_helpers \
  --input-pkg-root "$LANDSCAPER_MODULE_PATH" \
  --output-base "$src_path" \
  --boilerplate "${PROJECT_ROOT}/hack/boilerplate.go.txt"

 echo
 echo "> Generating openapi definitions"
 kube::codegen::gen_openapi \
   --input-pkg-root "$API_MODULE_PATH" \
   --output-pkg-root "$API_MODULE_PATH" \
   --output-base "$src_path" \
   --extra-pkgs "$API_MODULE_PATH/core/v1alpha1" \
   --extra-pkgs "$API_MODULE_PATH/config/v1alpha1" \
   --extra-pkgs "$API_MODULE_PATH/config" \
   --extra-pkgs "$API_MODULE_PATH/deployer/core/v1alpha1" \
   --extra-pkgs "$API_MODULE_PATH/deployer/utils/readinesschecks" \
   --extra-pkgs "$API_MODULE_PATH/deployer/utils/managedresource" \
   --extra-pkgs "$API_MODULE_PATH/deployer/utils/continuousreconcile" \
   --extra-pkgs "$API_MODULE_PATH/deployer/helm/v1alpha1" \
   --extra-pkgs "$API_MODULE_PATH/deployer/manifest/v1alpha1" \
   --extra-pkgs "$API_MODULE_PATH/deployer/manifest/v1alpha2" \
   --extra-pkgs "$API_MODULE_PATH/deployer/container/v1alpha1" \
   --extra-pkgs "$API_MODULE_PATH/deployer/mock/v1alpha1" \
   --extra-pkgs "github.com/gardener/component-spec/bindings-go/apis/v2" \
   --extra-pkgs "k8s.io/api/core/v1" \
   --extra-pkgs "k8s.io/apimachinery/pkg/apis/meta/v1" \
   --extra-pkgs "k8s.io/apimachinery/pkg/api/resource" \
   --extra-pkgs "k8s.io/apimachinery/pkg/types" \
   --extra-pkgs "k8s.io/apimachinery/pkg/runtime" \
   --report-filename "$src_path/$API_MODULE_PATH/openapi/api_violations.report" \
   --update-report \
   --boilerplate "${PROJECT_ROOT}/hack/boilerplate.go.txt"

echo
echo "> Generating CRDs"
"$CONTROLLER_GEN" crd paths="$PROJECT_ROOT/apis/..." output:crd:artifacts:config="$PROJECT_ROOT/apis/crds/manifests"

echo
echo "> Generating API reference"
"$API_REF_GEN" --renderer=markdown --source-path "$PROJECT_ROOT/apis/core/v1alpha1" --config "$PROJECT_ROOT/hack/api-reference/core-config.yaml" --output-path "$PROJECT_ROOT/docs/api-reference/core.md"

echo
echo "> Generating mock client"
"$MOCKGEN" "-destination=$PROJECT_ROOT/controller-utils/pkg/kubernetes/mock/client_mock.go" sigs.k8s.io/controller-runtime/pkg/client Client,StatusWriter
"$MOCKGEN" "-destination=$PROJECT_ROOT/pkg/landscaper/registry/components/mock/resolver_mock.go" github.com/gardener/component-spec/bindings-go/ctf ComponentResolver

echo
echo "NOTE: If you changed the API then consider updating the example manifests."

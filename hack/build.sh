#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors.
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

PROJECT_ROOT="$(realpath $(dirname $0)/..)"
if [[ -z ${COMPONENT_MAIN_PATH:-} ]]; then
  COMPONENT_MAIN_PATH="$COMPONENT"
fi

if [[ -z ${EFFECTIVE_VERSION:-} ]]; then
  EFFECTIVE_VERSION=$("$PROJECT_ROOT/hack/get-version.sh")
fi

echo "> Building binaries for component '$COMPONENT' ..."
(
  cd "$PROJECT_ROOT"
  for pf in ${PLATFORMS//,/ }; do
    echo "  > Building binary for $pf ..."
    os=${pf%/*}
    arch=${pf#*/}
    CGO_ENABLED=0 GOOS=$os GOARCH=$arch \
      go build -a -o "bin/${COMPONENT}-${os}.${arch}" \
      -ldflags "-X github.com/gardener/landscaper/pkg/version.GitVersion=$EFFECTIVE_VERSION \
                -X github.com/gardener/landscaper/pkg/version.gitTreeState=$([ -z git status --porcelain 2>/dev/null ] && echo clean || echo dirty) \
                -X github.com/gardener/landscaper/pkg/version.gitCommit=$(git rev-parse --verify HEAD) \
                -X github.com/gardener/landscaper/pkg/version.buildDate=$(date --rfc-3339=seconds | sed 's/ /T/')" \
      "cmd/$COMPONENT_MAIN_PATH/main.go"
  done
)

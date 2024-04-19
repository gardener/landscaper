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
if [[ -z ${REGISTRY:-} ]]; then
  REGISTRY="$LOCALBIN/registry"
fi

REGISTRY_DIR="$(dirname ${REGISTRY})"
REGISTRY_ARCHIVE="${REGISTRY_DIR}/registry.tar.gz"
REGISTRY_REPO_DIR="$(mktemp -d)"

git clone --quiet  --depth=1 --branch="$1" https://github.com/distribution/distribution.git "${REGISTRY_REPO_DIR}" 2>/dev/null
(
  cd "${REGISTRY_REPO_DIR}"
  make bin/registry 2>/dev/null
)
cp "${REGISTRY_REPO_DIR}/bin/registry" "${REGISTRY}"
rm -rf "${REGISTRY_REPO_DIR}"

chmod +x "${REGISTRY}"

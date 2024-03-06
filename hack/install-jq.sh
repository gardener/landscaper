#!/bin/bash
#
# SPDX-FileCopyrightText: 2018 SAP SE or an SAP affiliate company and Gardener contributors.
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

PROJECT_ROOT="$(realpath $(dirname $0)/..)"
if [[ -z ${LOCALBIN:-} ]]; then
  LOCALBIN="$PROJECT_ROOT/bin"
fi
if [[ -z ${JQ:-} ]]; then
  JQ="$LOCALBIN/jq"
fi

JQ_VERSION="$1"

os="linux64"
if [[ $(uname -o) == "Darwin" ]]; then
  os="osx-amd64"
fi
curl -sfL "https://github.com/stedolan/jq/releases/download/jq-${JQ_VERSION}/jq-${os}" --output "${LOCALBIN}/jq"
chmod +x "${LOCALBIN}/jq"
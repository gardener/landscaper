#!/bin/bash
#
# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

# test OCI registry currently doesnt support darwin/arm64 / force amd64
ARCH_ARG=$(go env GOARCH)
if [[ $(go env GOOS) == "darwin" && $(go env GOARCH) == "arm64" ]]; then
  ARCH_ARG="amd64"
fi

mkdir -p ${PROJECT_ROOT}/tmp/test/registry
curl -sfL "https://storage.googleapis.com/gardener-public/test/oci-registry/registry-$(go env GOOS)-${ARCH_ARG}" --output ${PROJECT_ROOT}/tmp/test/registry/registry
chmod +x ${PROJECT_ROOT}/tmp/test/registry/registry

GO111MODULE=off go get golang.org/x/tools/cmd/goimports

curl -sfL "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh" | sh -s -- -b $(go env GOPATH)/bin v1.53.3

# install jq (needed for documentation index generation)
if ! jq --version &>/dev/null; then
  os="linux64"
  if [[ $(uname -o) == "Darwin" ]]; then
    os="osx-amd64"
  fi
  tmpdir=$(mktemp -d)
  curl -sfL "https://github.com/stedolan/jq/releases/download/jq-1.6/jq-${os}" --output "${tmpdir}/jq"
  chmod +x "${tmpdir}/jq"
  # try to copy to /usr/local/bin, modify PATH as a workaround
  if ! cp "${tmpdir}/jq" /usr/local/bin/jq >/dev/null; then
    export PATH=${tmpdir}/jq:$PATH
  fi
fi

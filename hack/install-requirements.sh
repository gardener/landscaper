#!/bin/bash
#
# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

# OCI Registry

REGISTRY_LOCAL_DIR="${PROJECT_ROOT}/tmp/test/registry"
REGISTRY_LOCAL_BINARY="${REGISTRY_LOCAL_DIR}/registry"

if [[ ! -f ${REGISTRY_LOCAL_BINARY} || ! $(${REGISTRY_LOCAL_BINARY} --version) ]]; then
  # test OCI registry currently doesnt support darwin/arm64 / force amd64
  ARCH_ARG=$(go env GOARCH)
  GO_OS=$(go env GOOS)
  if [[ ${GO_OS} == "darwin" && ${ARCH_ARG} == "arm64" ]]; then
    ARCH_ARG="amd64"
  fi

  mkdir -p ${PROJECT_ROOT}/tmp/test/registry
  REGISTRY_VERSION="2.8.3"
  REGISTRY_DOWNLOAD_FILENAME="registry_${REGISTRY_VERSION}_${GO_OS}_${ARCH_ARG}.tar.gz"
  REGISTRY_DOWNLOAD_FILEPATH="${PROJECT_ROOT}/tmp/test/${REGISTRY_DOWNLOAD_FILENAME}"
  REGISTRY_URL="https://github.com/distribution/distribution/releases/download/v${REGISTRY_VERSION}/${REGISTRY_DOWNLOAD_FILENAME}"
  echo "Downloading registry from REGISTRY_DOWNLOAD_URL to \"${REGISTRY_DOWNLOAD_FILEPATH}\""
  set +e
  curl_rc=$(curl -sfL "${REGISTRY_URL}" --output ${REGISTRY_DOWNLOAD_FILEPATH})
  set -e
  if [[ ${curl_rc} == 0 ]]; then
    echo "Extracting file \"registry\" from \"${REGISTRY_DOWNLOAD_FILEPATH}\" to \"${PROJECT_ROOT}/tmp/test/registry\""
    tar -C ${PROJECT_ROOT}/tmp/test/registry -xzf ${REGISTRY_DOWNLOAD_FILEPATH} registry
  else
    echo "Can't download registry \"${REGISTRY_DOWNLOAD_FILENAME}\", clone repository and build."
    if [[ ! -d ${PROJECT_ROOT}/tmp/test/distribution ]]; then
      echo "git clone --depth 1 --branch v${REGISTRY_VERSION} https://github.com/distribution/distribution.git"
      # git clone --quiet --branch v${REGISTRY_VERSION} https://github.com/distribution/distribution.git ${PROJECT_ROOT}/tmp/test/distribution
      git clone --quiet  https://github.com/distribution/distribution.git ${PROJECT_ROOT}/tmp/test/distribution
    else
      echo "Reposisory github.com/distribution already downloaed"
    fi

    echo "CURRENT_DIR=${CURRENT_DIR}"
    cd ${PROJECT_ROOT}/tmp/test/distribution
    make bin/registry
    cp bin/registry ${REGISTRY_LOCAL_BINARY}
    cd ${CURRENT_DIR}
  fi

  # old deleted location
  # curl -sfL "https://storage.googleapis.com/gardener-public/test/oci-registry/registry-$(go env GOOS)-${ARCH_ARG}" --output ${PROJECT_ROOT}/tmp/test/registry/registry
  chmod +x ${PROJECT_ROOT}/tmp/test/registry/registry
else
    echo "OCI registry already installed with version: $(${REGISTRY_LOCAL_BINARY} --version)"
fi

GO111MODULE=off go get golang.org/x/tools/cmd/goimports

curl -sfL "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh" | sh -s -- -b $(go env GOPATH)/bin v1.54.2

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

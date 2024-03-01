#!/bin/bash
#
# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

if [[ $EFFECTIVE_VERSION == "" ]]; then
  EFFECTIVE_VERSION=$(cat $PROJECT_ROOT/VERSION)
fi

if [[ $BUILD_OS == "" ]]; then
  BUILD_OS="linux"
fi

if [[ $BUILD_ARCH == "" ]]; then
  BUILD_ARCH="amd64"
fi

if [[ $OUT_DIR == "" ]]; then
  OUT_DIR="${GOPATH}/bin"
fi

echo "> Install $EFFECTIVE_VERSION / ${BUILD_OS}-${BUILD_ARCH}"

CGO_ENABLED=0 GOOS=${BUILD_OS} GOARCH=${BUILD_ARCH} GO111MODULE=on \
  go build -mod=vendor -v -o $OUT_DIR \
  -ldflags "-X github.com/gardener/landscaper/pkg/version.GitVersion=$EFFECTIVE_VERSION \
            -X github.com/gardener/landscaper/pkg/version.gitTreeState=$([ -z git status --porcelain 2>/dev/null ] && echo clean || echo dirty) \
            -X github.com/gardener/landscaper/pkg/version.gitCommit=$(git rev-parse --verify HEAD) \
            -X github.com/gardener/landscaper/pkg/version.buildDate=$(date --rfc-3339=seconds | sed 's/ /T/')" \
  ${PROJECT_ROOT}/cmd/...

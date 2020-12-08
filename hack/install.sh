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

CGO_ENABLED=0 GOOS=$(go env GOOS) GOARCH=$(go env GOARCH) GO111MODULE=on \
  go install -mod=vendor \
  -ldflags "-X github.com/gardener/landscaper/pkg/version.gitVersion=$EFFECTIVE_VERSION \
            -X github.com/gardener/landscaper/pkg/version.gitTreeState=$([ -z git status --porcelain 2>/dev/null ] && echo clean || echo dirty) \
            -X github.com/gardener/landscaper/pkg/version.gitCommit=$(git rev-parse --verify HEAD) \
            -X github.com/gardener/landscaper/pkg/version.buildDate=$(date --rfc-3339=seconds | sed 's/ /T/')" \
  ${PROJECT_ROOT}/cmd/...

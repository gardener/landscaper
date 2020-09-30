#!/bin/bash
#
# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

if [[ $EFFECTIVE_VERSION == "" ]]; then
  EFFECTIVE_VERSION=$(cat $PROJECT_ROOT/VERSION)
fi

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on \
  go install -mod=vendor \
  -ldflags "-X github.com/gardener/landscaper/pkg/version.gitVersion=$EFFECTIVE_VERSION \
            -X github.com/gardener/landscaper/pkg/version.gitTreeState=$([ -z git status --porcelain 2>/dev/null ] && echo clean || echo dirty) \
            -X github.com/gardener/landscaper/pkg/version.gitCommit=$(git rev-parse --verify HEAD) \
            -X github.com/gardener/landscaper/pkg/version.buildDate=$(date --rfc-3339=seconds | sed 's/ /T/')" \
  ${PROJECT_ROOT}/cmd/...

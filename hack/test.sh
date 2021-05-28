#!/bin/bash
#
# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

go test -mod=vendor ${PROJECT_ROOT}/cmd/... \
                    ${PROJECT_ROOT}/pkg/... \
                    ${PROJECT_ROOT}/test/framework/... \
                    ${PROJECT_ROOT}/test/utils/... \
                    ${PROJECT_ROOT}/test/landscaper/...
cd ${PROJECT_ROOT}/apis && GO111MODULE=on go test ./...

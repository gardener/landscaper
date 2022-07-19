#!/bin/bash
#
# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

# On MacOS there is a strange race condition
# between port allocation of envtest suites when go test
# runs all the tests in parallel without any limits (spins up around 10+ environments).
#
# To avoid flakes, set we're setting the go-test parallel flag
# to limit the number of parallel executions.
#
# TODO: check the controller-runtime for root-cause and real mitigation
# https://github.com/kubernetes-sigs/controller-runtime/pull/1567
if [[ "${OSTYPE}" == "darwin"* ]]; then
  P_FLAG="-p=1"
fi

go test -mod=vendor -race -coverprofile=${PROJECT_ROOT}/coverage.main.out -covermode=atomic ${P_FLAG} \
                    ${PROJECT_ROOT}/cmd/... \
                    ${PROJECT_ROOT}/pkg/... \
                    ${PROJECT_ROOT}/test/framework/... \
                    ${PROJECT_ROOT}/test/utils/... \
                    ${PROJECT_ROOT}/test/landscaper/...
EXIT_STATUS_MAIN_TEST=$?

cd ${PROJECT_ROOT}/apis && GO111MODULE=on go test -coverprofile=${PROJECT_ROOT}/coverage.api.out -covermode=atomic ./...
EXIT_STATUS_API_TEST=$?

cd ${PROJECT_ROOT}/controller-utils && GO111MODULE=on go test -coverprofile=${PROJECT_ROOT}/coverage.controller-utils.out -covermode=atomic ./...
EXIT_STATUS_CONTROLLER_UTILS_TEST=$?

! (( EXIT_STATUS_MAIN_TEST || EXIT_STATUS_API_TEST || EXIT_STATUS_CONTROLLER_UTILS_TEST ))

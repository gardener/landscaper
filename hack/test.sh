#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
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

CGO_ENABLED=1 go test -race -coverprofile=${PROJECT_ROOT}/coverage.main.out -covermode=atomic ${P_FLAG} \
                    ${PROJECT_ROOT}/cmd/... \
                    ${PROJECT_ROOT}/pkg/... \
                    ${PROJECT_ROOT}/test/framework/... \
                    ${PROJECT_ROOT}/test/utils/... \
                    ${PROJECT_ROOT}/test/landscaper/...
EXIT_STATUS_MAIN_TEST=$?
go tool cover -html=${PROJECT_ROOT}/coverage.main.out -o ${PROJECT_ROOT}/coverage.main.html

cd ${PROJECT_ROOT}/apis && GO111MODULE=on go test -coverprofile=${PROJECT_ROOT}/coverage.api.out -covermode=atomic ./...
EXIT_STATUS_API_TEST=$?
go tool cover -html=${PROJECT_ROOT}/coverage.api.out -o ${PROJECT_ROOT}/coverage.api.html

cd ${PROJECT_ROOT}/controller-utils && GO111MODULE=on go test -coverprofile=${PROJECT_ROOT}/coverage.controller-utils.out -covermode=atomic ./...
EXIT_STATUS_CONTROLLER_UTILS_TEST=$?
go tool cover -html=${PROJECT_ROOT}/coverage.controller-utils.out -o ${PROJECT_ROOT}/coverage.controller-utils.html

! (( EXIT_STATUS_MAIN_TEST || EXIT_STATUS_API_TEST || EXIT_STATUS_CONTROLLER_UTILS_TEST ))

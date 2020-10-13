#!/bin/bash
#
# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

curl -sfL "https://install.goreleaser.com/github.com/golangci/golangci-lint.sh" | sh -s -- -b $(go env GOPATH)/bin v1.31.0

GO111MODULE=off go get golang.org/x/tools/cmd/goimports

echo "> Download Kubernetes test binaries"
TEST_BIN_DIR=${PROJECT_ROOT}/tmp/test/bin
KUBEBUILDER_VER=2.3.1

os=$(go env GOOS)
arch=$(go env GOARCH)

mkdir -p ${TEST_BIN_DIR}

curl -L https://go.kubebuilder.io/dl/${KUBEBUILDER_VER}/${os}/${arch} | tar -xz -C /tmp/
mv /tmp/kubebuilder_${KUBEBUILDER_VER}_${os}_${arch}/bin/* ${TEST_BIN_DIR}/

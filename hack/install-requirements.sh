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

curl -sfL "https://install.goreleaser.com/github.com/golangci/golangci-lint.sh" | sh -s -- -b $(go env GOPATH)/bin v1.24.0

echo "> Download Kubernetes test binaries"
TEST_BIN_DIR=${PROJECT_ROOT}/tmp/test/bin
K8S_VERSION=1.18.0
ETCD_VER=v3.3.11

os=$(go env GOOS)
arch=$(go env GOARCH)

mkdir -p ${TEST_BIN_DIR}
curl -LO https://storage.googleapis.com/kubernetes-release/release/v${K8S_VERSION}/bin/${os}/${arch}/kubectl
chmod +x kubectl
mv kubectl ${TEST_BIN_DIR}/kubectl

curl -LO https://storage.googleapis.com/kubernetes-release/release/v${K8S_VERSION}/bin/${os}/${arch}/kube-apiserver
chmod +x kube-apiserver
mv kube-apiserver ${TEST_BIN_DIR}/kube-apiserver

if [[ $os == "darwin" ]]; then
  curl -L https://storage.googleapis.com/etcd/${ETCD_VER}/etcd-${ETCD_VER}-${os}-${arch}.zip -o /tmp/etcd-${ETCD_VER}-${os}-${arch}.zip
  unzip /tmp/etcd-${ETCD_VER}-${os}-${arch}.zip -d /tmp
  mv /tmp/etcd-${ETCD_VER}-${os}-${arch}/etcd ${TEST_BIN_DIR}/etcd
  rm /tmp/etcd-${ETCD_VER}-${os}-${arch}.zip
  rm -r /tmp/etcd-${ETCD_VER}-${os}-${arch}
else
  curl -L https://storage.googleapis.com/etcd/${ETCD_VER}/etcd-${ETCD_VER}-${os}-${arch}.tar.gz -o /tmp/etcd-${ETCD_VER}-${os}-${arch}.tar.gz
  tar xzvf /tmp/etcd-${ETCD_VER}-${os}-${arch}.tar.gz /tmp/etcd-${ETCD_VER}-${os}-${arch}/etcd -C /tmp
  mv /tmp/etcd-${ETCD_VER}-${os}-${arch}/etcd ${TEST_BIN_DIR}/etcd
fi
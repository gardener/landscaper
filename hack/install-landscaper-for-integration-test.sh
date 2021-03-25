#!/bin/sh
#
# Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

# install bash for the get version command
if ! which bash 1>/dev/null; then
  echo "Installing bash... "
  apk add --no-cache bash
fi

if ! which openssl 1>/dev/null; then
  echo "Installing openssl... "
  apk add openssl
fi

if ! which curl 1>/dev/null; then
  echo "Installing curl... "
  apk add curl
fi

if ! which git 1>/dev/null; then
  echo "Installing git... "
  apk add --no-cache git
fi

if ! which kubectl 1>/dev/null; then
  echo "Kubectl is not installed, trying to install it..."
  curl -LO https://dl.k8s.io/release/v1.20.0/bin/linux/amd64/kubectl
  mv ./kubectl /usr/local/bin/kubectl
  chmod +x /usr/local/bin/kubectl
fi

if ! which helm 1>/dev/null; then
  echo "Helm 3 is not installed, trying to install it..."
  export DESIRED_VERSION=v3.5.1
  curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
fi

VERSION="$(${CURRENT_DIR}/get-version.sh)"

if [[ -z "VERSION" ]]; then
  echo "No version defined"
  exit 1
fi

echo "> Installing Landscaper version ${VERSION}"

printf "
landscaper:
  deployers:
  - container
  - helm
  - manifest
  - mock
  deployItemPickupTimeout: 10s
" > /tmp/values.yaml

touch /tmp/registry-values.yaml
if [[ -f "$TM_SHARED_PATH/docker.config" ]]; then
  printf "
landscaper:
  registryConfig:
    allowPlainHttpRegistries: false
    insecureSkipVerify: true
    secrets:
      default: $(cat "$TM_SHARED_PATH/docker.config")
  " > /tmp/registry-values.yaml
fi

export KUBECONFIG="${TM_KUBECONFIG_PATH}/${CLUSTER_NAME}.config"
helm upgrade --install --create-namespace ls -n ls-system ./charts/landscaper -f /tmp/values.yaml -f /tmp/registry-values.yaml --set "image.tag=${VERSION}"

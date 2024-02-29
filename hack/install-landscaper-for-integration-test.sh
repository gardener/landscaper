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
  curl -LO https://dl.k8s.io/release/v1.26.0/bin/linux/amd64/kubectl
  mv ./kubectl /usr/local/bin/kubectl
  chmod +x /usr/local/bin/kubectl
fi

if ! which helm 1>/dev/null; then
  echo "Helm 3 is not installed, trying to install it..."
  export DESIRED_VERSION=v3.7.0
  curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
fi

# in the testmachinery the version should be set by the EFFECTIVE_VERSION
VERSION="$(${CURRENT_DIR}/get-version.sh)"

if [[ -z "VERSION" ]]; then
  echo "No version defined"
  exit 1
fi

echo "> Installing Landscaper version ${VERSION}"

tmp_dir="$(mktemp -d)"

printf "
landscaper:
  landscaper:
    deployItemTimeouts:
      pickup: 30s
      abort: 30s
    useOCMLib: true
" > "/${tmp_dir}/landscaper-values.yaml"

touch /tmp/registry-values.yaml
if [[ -f "$TM_SHARED_PATH/docker.config" ]]; then
  printf "
landscaper:
  landscaper:
    registryConfig:
      allowPlainHttpRegistries: false
      insecureSkipVerify: true
      secrets:
        default: $(cat "$TM_SHARED_PATH/docker.config")
  " > "/${tmp_dir}/registry-values.yaml"
fi

export KUBECONFIG="${TM_KUBECONFIG_PATH}/${CLUSTER_NAME}.config"
helm upgrade --install --create-namespace -n ls-system landscaper ./charts/landscaper -f "/${tmp_dir}/landscaper-values.yaml" -f "/${tmp_dir}/registry-values.yaml" --set "landscaper.image.tag=${VERSION}"

landscaper_ready=false
retries_left=20

while [ "$landscaper_ready" = false ]; do
  kubectl get customresourcedefinitions.apiextensions.k8s.io installations.landscaper.gardener.cloud
  if [ "$?" = 0 ]; then
    landscaper_ready=true
  fi

  if [ "retries_left" == 0 ]; then
    >&2 echo "landscaper is not ready after max retries"
    exit 1
  fi

  retries_left="$((${retries_left}-1))"
  sleep 1
done

printf "
deployer:
  verbosityLevel: debug
" > "/${tmp_dir}/deployer-values.yaml"

helm upgrade --install -n ls-system manifest-deployer ./charts/manifest-deployer -f "/${tmp_dir}/deployer-values.yaml" --set "image.tag=${VERSION}"
helm upgrade --install -n ls-system helm-deployer ./charts/helm-deployer -f "/${tmp_dir}/deployer-values.yaml" --set "image.tag=${VERSION}"
helm upgrade --install -n ls-system container-deployer ./charts/container-deployer -f "/${tmp_dir}/deployer-values.yaml" --set "image.tag=${VERSION}"
helm upgrade --install -n ls-system mock-deployer ./charts/mock-deployer -f "/${tmp_dir}/deployer-values.yaml" --set "image.tag=${VERSION}"

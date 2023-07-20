#!/bin/bash
#
# Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

SOURCE_PATH="$(dirname $0)/.."
oci_images=$@
REPO_CTX="eu.gcr.io/sap-gcp-cp-k8s-stable-hub/landscaper"
CA_PATH="$(mktemp -d)"
BASE_DEFINITION_PATH="${CA_PATH}/component-descriptor.yaml"

if [[ $EFFECTIVE_VERSION == "" ]]; then
  EFFECTIVE_VERSION="$(${SOURCE_PATH}/hack/get-version.sh)"
fi

if ! which component-cli 1>/dev/null; then
  echo -n "component-cli is required to generate the component descriptors"
  echo -n "Trying to installing it..."
  curl -L https://github.com/gardener/component-cli/releases/download/$(curl -s https://api.github.com/repos/gardener/component-cli/releases/latest | jq -r '.tag_name')/componentcli-$(go env GOOS)-$(go env GOARCH).gz | gzip -d > $(go env GOPATH)/bin/component-cli
  chmod +x $(go env GOPATH)/bin/component-cli

  if ! which component-cli 1>/dev/null; then
    echo -n "component-cli was successfully installed but the binary cannot be found"
    echo -n "Try adding the \$GOPATH/bin to your \$PATH..."
    exit 1
  fi
fi
if ! which jq 1>/dev/null; then
  echo -n "jq canot be found"
  exit 1
fi

echo "> Generate Component Descriptor ${EFFECTIVE_VERSION}"
echo "> Creating base definition"
component-cli ca create "${CA_PATH}" \
    --component-name=github.com/gardener/landscaper/landscaper \
    --component-version=${EFFECTIVE_VERSION} \
    --repo-ctx=${REPO_CTX}

# add oci images
#for image in "${oci_images[@]}"
#do
#  echo "Adding ${image} to component descriptor"
#  cat <<EOF | component-cli ca resources add "${CA_PATH}" -
#name: ${image}
#version: 'v0.0.1'
#type: 'ociImage'
#relation: 'external'
#access:
#  type: 'ociRegistry'
#  imageReference: ${image}
#EOF
#done

echo "> Creating ctf"
CTF_PATH=./gen/ctf.tar
mkdir -p ./gen
[ -e $CTF_PATH ] && rm ${CTF_PATH}
CTF_PATH=${CTF_PATH} BASE_DEFINITION_PATH=${BASE_DEFINITION_PATH} CURRENT_COMPONENT_REPOSITORY=${REPO_CTX} bash $SOURCE_PATH/.ci/component_descriptor

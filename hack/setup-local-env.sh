#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -e

CLUSTER_NAME="test"
K3D_INTERNAL_HOST="host.k3d.internal"

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

echo "Setup local Landscaper development environment"

if ! which k3d 1>/dev/null; then
  echo "K3d is not installed, see https://k3d.io/ how to install it..."
  exit 1
fi


found_cluster=$(k3d cluster list -o json | jq ".[] | select(.name==\"${CLUSTER_NAME}\").name")
if [[ -z $found_cluster ]]; then
  echo "Creating k3d cluster ..."
  k3d cluster create $CLUSTER_NAME --config ${CURRENT_DIR}/resources/k3d-cluster-config.yaml
else
  echo "k3d cluster ${CLUSTER_NAME} already exists. Skipping..."
fi

echo "Note: the new cluster context is automatically added to your current scope"

echo "Adding k3d host entry ..."
echo "# Local k3d cluster" >> /etc/hosts
echo "0.0.0.0 ${K3D_INTERNAL_HOST}" >> /etc/hosts

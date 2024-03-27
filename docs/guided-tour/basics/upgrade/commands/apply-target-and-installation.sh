#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

COMPONENT_DIR="$(dirname $0)/.."

source ${COMPONENT_DIR}/commands/settings

# create namespace "example" on the Landscaper resource cluster
kubectl create ns example --kubeconfig="${LS_DATA_KUBECONFIG}"

# create target
landscaper-cli targets create kubernetes-cluster \
  --name my-cluster \
  --namespace example \
  --target-kubeconfig "${TARGET_KUBECONFIG}" \
  | kubectl apply -f - --kubeconfig="${LS_DATA_KUBECONFIG}"

# create installation
kubectl apply -f "${COMPONENT_DIR}/installation/installation-1.0.0.yaml" --kubeconfig="${LS_DATA_KUBECONFIG}"

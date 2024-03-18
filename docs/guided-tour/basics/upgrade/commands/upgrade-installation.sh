#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

COMPONENT_DIR="$(dirname $0)/.."

source ${COMPONENT_DIR}/commands/settings


# upgrade installation
kubectl apply -f "${COMPONENT_DIR}/installation/installation-1.0.1.yaml" --kubeconfig="${LS_DATA_KUBECONFIG}"

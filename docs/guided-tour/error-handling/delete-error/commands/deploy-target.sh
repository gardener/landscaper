#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit

COMPONENT_DIR="$(dirname $0)/.."
cd "${COMPONENT_DIR}"
COMPONENT_DIR="$(pwd)"
echo "COMPONENT_DIR: ${COMPONENT_DIR}"

source "${COMPONENT_DIR}/commands/settings"

TMP_DIR=`mktemp -d`
echo "TMP_DIR: ${TMP_DIR}"

echo "creating target"
echo "target cluster kubeconfig: $TARGET_CLUSTER_KUBECONFIG_PATH"
outputFile="${TMP_DIR}/target.yaml"
export namespace="${NAMESPACE}"
export kubeconfig=`sed 's/^/      /' $TARGET_CLUSTER_KUBECONFIG_PATH`
inputFile="${COMPONENT_DIR}/installation/target.yaml.tpl"
envsubst < ${inputFile} > ${outputFile}
kubectl apply -f ${outputFile} --kubeconfig="${RESOURCE_CLUSTER_KUBECONFIG_PATH}"

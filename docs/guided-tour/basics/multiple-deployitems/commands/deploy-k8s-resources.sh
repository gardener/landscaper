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

echo "creating target 1"
echo "target cluster kubeconfig: ${TARGET_CLUSTER_KUBECONFIG_PATH_1}"
outputFile="${TMP_DIR}/target-1.yaml"
export name="my-cluster-1"
export namespace="${NAMESPACE}"
export kubeconfig_path="${TARGET_CLUSTER_KUBECONFIG_PATH_1}"
inputFile="${COMPONENT_DIR}/installation/target.yaml.tpl"
envsubst < ${inputFile} > ${outputFile}
kubectl apply -f "${outputFile}" --kubeconfig="${RESOURCE_CLUSTER_KUBECONFIG_PATH}"

echo "creating target 2"
echo "target cluster kubeconfig: ${TARGET_CLUSTER_KUBECONFIG_PATH_2}"
outputFile="${TMP_DIR}/target-2.yaml"
export name="my-cluster-2"
export namespace="${NAMESPACE}"
export kubeconfig_path="${TARGET_CLUSTER_KUBECONFIG_PATH_2}"
inputFile="${COMPONENT_DIR}/installation/target.yaml.tpl"
envsubst < ${inputFile} > ${outputFile}
kubectl apply -f "${outputFile}" --kubeconfig="${RESOURCE_CLUSTER_KUBECONFIG_PATH}"

echo "creating installation"
outputFile="${TMP_DIR}/installation.yaml.tpl"
export namespace="${NAMESPACE}"
inputFile=""${COMPONENT_DIR}/installation/installation.yaml.tpl"
envsubst < ${inputFile} > ${outputFile}
kubectl apply -f ${outputFile} --kubeconfig="${RESOURCE_CLUSTER_KUBECONFIG_PATH}"

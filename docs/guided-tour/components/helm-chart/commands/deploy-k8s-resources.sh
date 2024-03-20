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

echo "creating context"
outputFile="${TMP_DIR}/context.yaml"
mako-render "${COMPONENT_DIR}/installation/context.yaml.tpl" \
  --var namespace="${NAMESPACE}" \
  --var contextName="${CONTEXT_NAME}" \
  --var repoBaseUrl="${REPO_BASE_URL}" \
  --output-file=${outputFile}
kubectl apply -f ${outputFile}

echo "creating target"
echo "target cluster kubeconfig: $TARGET_CLUSTER_KUBECONFIG_PATH"
 outputFile="${TMP_DIR}/target.yaml"
 mako-render "${COMPONENT_DIR}/installation/target.yaml.tpl" \
   --var namespace="${NAMESPACE}" \
   --var targetName="${TARGET_NAME}" \
   --var kubeconfig_path="${TARGET_CLUSTER_KUBECONFIG_PATH}" \
   --output-file=${outputFile}
 kubectl apply -f ${outputFile}

echo "creating installation"
outputFile="${TMP_DIR}/installation.yaml"
mako-render "${COMPONENT_DIR}/installation/installation.yaml.tpl" \
  --var namespace="${NAMESPACE}" \
  --var installationName=${INSTALLATION_NAME} \
  --var contextName="${CONTEXT_NAME}" \
   --var targetName="${TARGET_NAME}" \
  --output-file=${outputFile}
kubectl apply -f ${outputFile}

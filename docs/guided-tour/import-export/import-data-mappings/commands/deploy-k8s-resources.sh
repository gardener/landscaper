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
export namespace="${NAMESPACE}"
export repoBaseUrl="${REPO_BASE_URL}"
inputFile="${COMPONENT_DIR}/installation/context.yaml.tpl"
envsubst < ${inputFile} > ${outputFile}
kubectl apply -f ${outputFile}

echo "creating target"
echo "target cluster kubeconfig: $TARGET_CLUSTER_KUBECONFIG_PATH"
outputFile="${TMP_DIR}/target.yaml"
export namespace="${NAMESPACE}"
export kubeconfig=`sed 's/^/      /' $TARGET_CLUSTER_KUBECONFIG_PATH`
inputFile="${COMPONENT_DIR}/installation/target.yaml.tpl"
envsubst < ${inputFile} > ${outputFile}
kubectl apply -f ${outputFile}

echo "creating dataobject my-release-name"
outputFile="${TMP_DIR}/dataobject-name.yaml"
export namespace="${NAMESPACE}"
inputFile="${COMPONENT_DIR}/installation/dataobject-name.yaml.tpl"
envsubst < ${inputFile} > ${outputFile}
kubectl apply -f ${outputFile}

echo "creating dataobject my-release-namespace"
outputFile="${TMP_DIR}/dataobject-namespace.yaml"
export namespace="${NAMESPACE}"
inputFile="${COMPONENT_DIR}/installation/dataobject-namespace.yaml.tpl"
envsubst < ${inputFile} > ${outputFile}
kubectl apply -f ${outputFile}

echo "creating installation"
outputFile="${TMP_DIR}/installation.yaml"
export namespace="${NAMESPACE}"
inputFile="${COMPONENT_DIR}/installation/installation.yaml.tpl"
envsubst < ${inputFile} > ${outputFile}
kubectl apply -f ${outputFile}

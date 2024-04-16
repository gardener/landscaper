#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit

COMPONENT_DIR="$(dirname $0)/.."
cd "${COMPONENT_DIR}"
COMPONENT_DIR="$(pwd)"
echo compdir ${COMPONENT_DIR}

source ${COMPONENT_DIR}/commands/settings

TMP_DIR=`mktemp -d`
echo tempdir ${TMP_DIR}

outputFile="${TMP_DIR}/context.yaml"
export namespace="${NAMESPACE}"
export repoBaseUrl="${REPO_BASE_URL}"
inputFile="${COMPONENT_DIR}/installation/context.yaml.tpl"
envsubst < ${inputFile} > ${outputFile}
kubectl delete -f ${outputFile}

outputFile="${TMP_DIR}/dataobject.yaml"
export namespace="${NAMESPACE}"
inputFile="${COMPONENT_DIR}/installation/dataobject.yaml.tpl"
envsubst < ${inputFile} > ${outputFile}
kubectl delete -f ${outputFile}


echo "Reading file $TARGET_CLUSTER_KUBECONFIG_PATH"

array=("blue" "green" "yellow" "orange" "red")
# Iterate over the array
for color in "${array[@]}"
do
   outputFile="${TMP_DIR}/target-${color}.yaml"
   export namespace="${NAMESPACE}"
   export color="${color}"
   export kubeconfig_path="${TARGET_CLUSTER_KUBECONFIG_PATH}"
   inputFile="${COMPONENT_DIR}/installation/target.yaml.tpl"
   envsubst < ${inputFile} > ${outputFile}
   kubectl delete -f ${outputFile}
done

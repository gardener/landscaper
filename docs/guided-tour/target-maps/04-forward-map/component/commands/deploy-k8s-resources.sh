#!/bin/bash
#
# Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit

COMPONENT_DIR="$(dirname $0)/.."
cd "${COMPONENT_DIR}"
COMPONENT_DIR="$(pwd)"
echo compdir ${COMPONENT_DIR}

source ${COMPONENT_DIR}/commands/settings

TMP_DIR=`mktemp -d`
echo tempdir ${TMP_DIR}

outputFile="${TMP_DIR}/context.yaml"
mako-render "${COMPONENT_DIR}/installation/context.yaml.tpl" \
  --var namespace="${NAMESPACE}" \
  --var repoBaseUrl="${REPO_BASE_URL}" \
  --output-file=${outputFile}
kubectl apply -f ${outputFile}

outputFile="${TMP_DIR}/dataobject.yaml"
mako-render "${COMPONENT_DIR}/installation/dataobject.yaml.tpl" \
  --var namespace="${NAMESPACE}" \
  --output-file=${outputFile}
kubectl apply -f ${outputFile}

echo "Reading file $TARGET_CLUSTER_KUBECONFIG_PATH"

array=("blue" "green" "yellow" "orange" "red")
# Iterate over the array
for color in "${array[@]}"
do
   outputFile="${TMP_DIR}/target-${color}.yaml"
   mako-render "${COMPONENT_DIR}/installation/target.yaml.tpl" \
     --var namespace="${NAMESPACE}" \
     --var color="${color}" \
     --var kubeconfig_path="${TARGET_CLUSTER_KUBECONFIG_PATH}" \
     --output-file=${outputFile}
   kubectl apply -f ${outputFile}
done

outputFile="${TMP_DIR}/installation.yaml"
mako-render "${COMPONENT_DIR}/installation/installation.yaml.tpl" \
  --var namespace="${NAMESPACE}" \
  --var targetNamespace="${TARGET_NAMESPACE}" \
  --output-file=${outputFile}
kubectl apply -f ${outputFile}

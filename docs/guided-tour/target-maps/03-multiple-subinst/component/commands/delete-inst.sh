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

outputFile="${TMP_DIR}/installation.yaml"
mako-render "${COMPONENT_DIR}/installation/installation.yaml.tpl" \
  --var namespace="${NAMESPACE}" \
  --var targetNamespace="${TARGET_NAMESPACE}" \
  --output-file=${outputFile}
kubectl delete -f ${outputFile}




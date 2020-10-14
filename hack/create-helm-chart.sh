#!/bin/bash
#
# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e
CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..
CHART_NAME=$1
CHART_PATH=$2

if [[ $EFFECTIVE_VERSION == "" ]]; then
  EFFECTIVE_VERSION=$(cat $PROJECT_ROOT/VERSION)
fi

if [[ -z "$CHART_PATH" ]]; then
  echo "CHART_PATH is undefined: create-helm-chat.sh [chart-name] [chart path] "
fi
if [[ -z "$CHART_NAME" ]]; then
  echo "CHART_NAME is undefined: create-helm-chat.sh [chart-name] [chart path]"
fi

if ! which openssl 1>/dev/null; then
  echo -n "Installing openssl... "
  apk add openssl
fi

echo -n "Installing helm... "
export DESIRED_VERSION=v3.2.1
curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash

cli.py config attribute --cfg-type container_registry --cfg-name gcr-readwrite --key password > /tmp/serviceaccount.yaml

echo "> Creating helm chart ${CHART_NAME}:${EFFECTIVE_VERSION} from $CHART_PATH"

# update version and appVersion
sed -i -e "s/0.1.0/${EFFECTIVE_VERSION}/" ${PROJECT_ROOT}/${CHART_PATH}/Chart.yaml

export HELM_EXPERIMENTAL_OCI=1
helm registry login eu.gcr.io -u _json_key -p "$(cat /tmp/serviceaccount.yaml)"

helm chart save ${PROJECT_ROOT}/${CHART_PATH} ${CHART_NAME}:${EFFECTIVE_VERSION}
helm chart push ${CHART_NAME}:${EFFECTIVE_VERSION}

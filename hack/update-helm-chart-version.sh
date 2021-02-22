#!/bin/bash
#
# Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..
CHART_ROOT="${PROJECT_ROOT}/charts"

if [[ -n $1 ]]; then
  EFFECTIVE_VERSION=$1
elif [[ $EFFECTIVE_VERSION == "" ]]; then
  EFFECTIVE_VERSION=$(cat $PROJECT_ROOT/VERSION)
fi

CHARTLIST=$(find $CHART_ROOT -maxdepth 2 -type f -name "Chart.yaml")

echo "Updating version and appVersion of Helm Charts to $EFFECTIVE_VERSION"

for chart in $CHARTLIST; do
    sed -i -e "s/^appVersion:.*/appVersion: ${EFFECTIVE_VERSION}/" $chart
done

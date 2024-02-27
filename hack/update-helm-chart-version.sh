#!/bin/bash
#
# Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

PROJECT_ROOT="$(realpath $(dirname $0)/..)"
CHART_ROOT="${PROJECT_ROOT}/charts"

if [[ -n $1 ]]; then
  EFFECTIVE_VERSION=$1
elif [[ -z ${EFFECTIVE_VERSION:-} ]]; then
  EFFECTIVE_VERSION=$("$PROJECT_ROOT/hack/get-version.sh")
fi

CHARTLIST=$(find $CHART_ROOT -maxdepth 10 -type f -name "Chart.yaml")

echo "Updating version and appVersion of Helm Charts to $EFFECTIVE_VERSION"

for chart in $CHARTLIST; do
    echo "Updating chart ${chart} with version ${EFFECTIVE_VERSION}"
    sed -i -e "s/^appVersion:.*/appVersion: ${EFFECTIVE_VERSION}/" $chart
    sed -i -e "s/version:.*/version: ${EFFECTIVE_VERSION}/" $chart

done

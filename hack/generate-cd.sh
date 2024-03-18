#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

PROJECT_ROOT="$(realpath $(dirname $0)/..)"
if [[ -z ${LOCALBIN:-} ]]; then
  LOCALBIN="$PROJECT_ROOT/bin"
fi
if [[ -z ${OCM:-} ]]; then
  OCM="$LOCALBIN/ocm"
fi

if [ -z $1 ]; then
  echo "provider argument is required"
  exit 1
fi

EFFECTIVE_VERSION="$(${PROJECT_ROOT}/hack/get-version.sh)"

echo -n "> Updating helm chart version"
${PROJECT_ROOT}/hack/update-helm-chart-version.sh

echo "> Create Component Version ${EFFECTIVE_VERSION}"

PROVIDER=$1
COMPONENT_ARCHIVE_PATH="$(mktemp -d)/ctf"
COMMIT_SHA=$(git rev-parse HEAD)

LANDSCAPER_CHART_PATH="${PROJECT_ROOT}/charts/landscaper"
LANDSCAPER_CONTROLLER_RBAC_CHART_PATH="${PROJECT_ROOT}/charts/landscaper/charts/rbac"
LANDSCAPER_CONTROLLER_DEPLOYMENT_CHART_PATH="${PROJECT_ROOT}/charts/landscaper/charts/landscaper"
LANDSCAPER_AGENT_CHART_PATH="${PROJECT_ROOT}/charts/landscaper-agent"
HELM_DEPLOYER_CHART_PATH="${PROJECT_ROOT}/charts/helm-deployer"
MANIFEST_DEPLOYER_CHART_PATH="${PROJECT_ROOT}/charts/manifest-deployer"
CONTAINER_DEPLOYER_CHART_PATH="${PROJECT_ROOT}/charts/container-deployer"
MOCK_DEPLOYER_CHART_PATH="${PROJECT_ROOT}/charts/mock-deployer"

"$OCM" add componentversions --create --file ${COMPONENT_ARCHIVE_PATH} ${PROJECT_ROOT}/.landscaper/components.yaml \
  -- VERSION=${EFFECTIVE_VERSION} \
     COMMIT_SHA=${COMMIT_SHA} \
     PROVIDER=landscaper.gardener.cloud \
     LANDSCAPER_CHART_PATH=${LANDSCAPER_CHART_PATH} \
     LANDSCAPER_CONTROLLER_RBAC_CHART_PATH=${LANDSCAPER_CONTROLLER_RBAC_CHART_PATH} \
     LANDSCAPER_CONTROLLER_DEPLOYMENT_CHART_PATH=${LANDSCAPER_CONTROLLER_DEPLOYMENT_CHART_PATH} \
     LANDSCAPER_AGENT_CHART_PATH=${LANDSCAPER_AGENT_CHART_PATH} \
     HELM_DEPLOYER_CHART_PATH=${HELM_DEPLOYER_CHART_PATH} \
     MANIFEST_DEPLOYER_CHART_PATH=${MANIFEST_DEPLOYER_CHART_PATH} \
     CONTAINER_DEPLOYER_CHART_PATH=${CONTAINER_DEPLOYER_CHART_PATH} \
     MOCK_DEPLOYER_CHART_PATH=${MOCK_DEPLOYER_CHART_PATH}

echo "> Transfer Component version ${EFFECTIVE_VERSION} to ${PROVIDER}"
"$OCM" transfer ctf --copy-resources --recursive --overwrite ${COMPONENT_ARCHIVE_PATH} ${PROVIDER}

echo "> Remote Component Version Landscaper"
"$OCM" get componentversion --repo OCIRegistry::${PROVIDER} "github.com/gardener/landscaper:${EFFECTIVE_VERSION}" -o yaml

echo "> Remote Component Version Helm Deployer"
"$OCM" get componentversion --repo OCIRegistry::${PROVIDER} "github.com/gardener/landscaper/helm-deployer:${EFFECTIVE_VERSION}" -o yaml

echo "> Remote Component Version Manifest Deployer"
"$OCM" get componentversion --repo OCIRegistry::${PROVIDER} "github.com/gardener/landscaper/manifest-deployer:${EFFECTIVE_VERSION}" -o yaml

echo "> Remote Component Version Container Deployer"
"$OCM" get componentversion --repo OCIRegistry::${PROVIDER} "github.com/gardener/landscaper/container-deployer:${EFFECTIVE_VERSION}" -o yaml

echo "> Remote Component Version Mock Deployer"
"$OCM" get componentversion --repo OCIRegistry::${PROVIDER} "github.com/gardener/landscaper/mock-deployer:${EFFECTIVE_VERSION}" -o yaml

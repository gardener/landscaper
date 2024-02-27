#!/bin/bash
#
# Copyright (c) 2024 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

if [ -z $1 ]; then
  echo "provider argument is required"
  exit 1
fi

SOURCE_PATH="$(realpath $(dirname $0)/..)"
EFFECTIVE_VERSION="$(${SOURCE_PATH}/hack/get-version.sh)"

echo -n "> Updating helm chart version"
${SOURCE_PATH}/hack/update-helm-chart-version.sh ${EFFECTIVE_VERSION}

echo "> Create Component Version ${EFFECTIVE_VERSION}"

PROVIDER=$1
COMPONENT_ARCHIVE_PATH="$(mktemp -d)/ctf"
COMMIT_SHA=$(git rev-parse HEAD)

LANDSCAPER_CHART_PATH="${SOURCE_PATH}/charts/landscaper"
LANDSCAPER_CONTROLLER_RBAC_CHART_PATH="${SOURCE_PATH}/charts/landscaper/charts/rbac"
LANDSCAPER_CONTROLLER_DEPLOYMENT_CHART_PATH="${SOURCE_PATH}/charts/landscaper/charts/landscaper"
LANDSCAPER_AGENT_CHART_PATH="${SOURCE_PATH}/charts/landscaper-agent"
HELM_DEPLOYER_CHART_PATH="${SOURCE_PATH}/charts/helm-deployer"
MANIFEST_DEPLOYER_CHART_PATH="${SOURCE_PATH}/charts/manifest-deployer"
CONTAINER_DEPLOYER_CHART_PATH="${SOURCE_PATH}/charts/container-deployer"
MOCK_DEPLOYER_CHART_PATH="${SOURCE_PATH}/charts/mock-deployer"

ocm add componentversions --create --file ${COMPONENT_ARCHIVE_PATH} ${SOURCE_PATH}/.landscaper/components.yaml \
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
ocm transfer ctf --copy-resources --recursive --overwrite ${COMPONENT_ARCHIVE_PATH} ${PROVIDER}

echo "> Remote Component Version Landscaper"
ocm get componentversion --repo OCIRegistry::${PROVIDER} "github.com/gardener/landscaper:${EFFECTIVE_VERSION}" -o yaml

echo "> Remote Component Version Helm Deployer"
ocm get componentversion --repo OCIRegistry::${PROVIDER} "github.com/gardener/landscaper/helm-deployer:${EFFECTIVE_VERSION}" -o yaml

echo "> Remote Component Version Manifest Deployer"
ocm get componentversion --repo OCIRegistry::${PROVIDER} "github.com/gardener/landscaper/manifest-deployer:${EFFECTIVE_VERSION}" -o yaml

echo "> Remote Component Version Container Deployer"
ocm get componentversion --repo OCIRegistry::${PROVIDER} "github.com/gardener/landscaper/container-deployer:${EFFECTIVE_VERSION}" -o yaml

echo "> Remote Component Version Mock Deployer"
ocm get componentversion --repo OCIRegistry::${PROVIDER} "github.com/gardener/landscaper/mock-deployer:${EFFECTIVE_VERSION}" -o yaml

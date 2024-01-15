#!/bin/bash
#
# Copyright (c) 2024 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

if ! which ocm 1>/dev/null; then
  curl -s https://ocm.software/install.sh | bash
fi

if ! which docker 1>/dev/null; then
  curl -L -o - https://download.docker.com/linux/static/stable/x86_64/docker-18.06.3-ce.tgz | tar zxvf - --strip 1 -C /usr/bin docker/docker
fi

SOURCE_PATH="$(realpath $(dirname $0)/..)"
EFFECTIVE_VERSION="$(${SOURCE_PATH}/hack/get-version.sh)"

echo "> Building docker images for version ${EFFECTIVE_VERSION}"
${SOURCE_PATH}/hack/prepare-docker-builder.sh

LANDSCAPER_CONTROLLER_IMAGE_PATH="landscaper-controller"
LANDSCAPER_WEBHOOKS_SERVER_IMAGE_PATH="landscaper-webhooks-server"
LANDSCAPER_AGENT_IMAGE_PATH="landscaper-agent"

HELM_DEPLOYER_CONTROLLER_IMAGE_PATH="helm-deployer-controller"

MANIFEST_DEPLOYER_CONTROLLER_IMAGE_PATH="manifest-deployer-controller"

CONTAINER_DEPLOYER_CONTROLLER_IMAGE_PATH="container-deployer-controller"
CONTAINER_DEPLOYER_INIT_IMAGE_PATH="container-deployer-init"
CONTAINER_DEPLOYER_WAIT_IMAGE_PATH="container-deployer-wait"

MOCK_DEPLOYER_CONTROLLER_IMAGE_PATH="mock-deployer-controller"

PLATFORM="linux/amd64"

docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${PLATFORM} -t ${LANDSCAPER_CONTROLLER_IMAGE_PATH}:${EFFECTIVE_VERSION} -f Dockerfile --target landscaper-controller .
docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${PLATFORM} -t ${LANDSCAPER_WEBHOOKS_SERVER_IMAGE_PATH}:${EFFECTIVE_VERSION} -f Dockerfile --target landscaper-webhooks-server .
docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${PLATFORM} -t ${LANDSCAPER_AGENT_IMAGE_PATH}:${EFFECTIVE_VERSION} -f Dockerfile --target landscaper-agent .
docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${PLATFORM} -t ${HELM_DEPLOYER_CONTROLLER_IMAGE_PATH}:${EFFECTIVE_VERSION} -f Dockerfile --target helm-deployer-controller .
docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${PLATFORM} -t ${MANIFEST_DEPLOYER_CONTROLLER_IMAGE_PATH}:${EFFECTIVE_VERSION} -f Dockerfile --target manifest-deployer-controller .
docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${PLATFORM} -t ${MOCK_DEPLOYER_CONTROLLER_IMAGE_PATH}:${EFFECTIVE_VERSION} -f Dockerfile --target mock-deployer-controller .
docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${PLATFORM} -t ${CONTAINER_DEPLOYER_CONTROLLER_IMAGE_PATH}:${EFFECTIVE_VERSION} -f Dockerfile --target container-deployer-controller .
docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${PLATFORM} -t ${CONTAINER_DEPLOYER_INIT_IMAGE_PATH}:${EFFECTIVE_VERSION} -f Dockerfile --target container-deployer-init .
docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${PLATFORM} -t ${CONTAINER_DEPLOYER_WAIT_IMAGE_PATH}:${EFFECTIVE_VERSION} -f Dockerfile --target container-deployer-wait .

echo -n "> Updating helm chart version"
${SOURCE_PATH}/hack/update-helm-chart-version.sh ${EFFECTIVE_VERSION}

echo "> Create Component Version ${EFFECTIVE_VERSION}"

PROVIDER="europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper"
COMPONENT_ARCHIVE_PATH="$(mktemp -d)/ctf"
COMMIT_SHA=$(git rev-parse HEAD)

LANDSCAPER_COMPONENT_NAME="github.com/gardener/landscaper"
LANDSCAPER_CHART_PATH="${SOURCE_PATH}/charts/landscaper"
LANDSCAPER_CONTROLLER_RBAC_CHART_PATH="${SOURCE_PATH}/charts/landscaper/charts/rbac"
LANDSCAPER_CONTROLLER_DEPLOYMENT_CHART_PATH="${SOURCE_PATH}/charts/landscaper/charts/landscaper"
LANDSCAPER_AGENT_CHART_PATH="${SOURCE_PATH}/charts/landscaper-agent"

HELM_DEPLOYER_COMPONENT_NAME="github.com/gardener/landscaper/helm-deployer"
HELM_DEPLOYER_CHART_PATH="${SOURCE_PATH}/charts/helm-deployer"

MANIFEST_DEPLOYER_COMPONENT_NAME="github.com/gardener/landscaper/manifest-deployer"
MANIFEST_DEPLOYER_CHART_PATH="${SOURCE_PATH}/charts/manifest-deployer"

CONTAINER_DEPLOYER_COMPONENT_NAME="github.com/gardener/landscaper/container-deployer"
CONTAINER_DEPLOYER_CHART_PATH="${SOURCE_PATH}/charts/container-deployer"

MOCK_DEPLOYER_COMPONENT_NAME="github.com/gardener/landscaper/mock-deployer"
MOCK_DEPLOYER_CHART_PATH="${SOURCE_PATH}/charts/mock-deployer"


ocm add componentversions --create --file ${COMPONENT_ARCHIVE_PATH} ${SOURCE_PATH}/.landscaper/components.yaml \
  -- VERSION=${EFFECTIVE_VERSION} \
     COMMIT_SHA=${COMMIT_SHA} \
     LANDSCAPER_COMPONENT_NAME=${LANDSCAPER_COMPONENT_NAME} \
     HELM_DEPLOYER_COMPONENT_NAME=${HELM_DEPLOYER_COMPONENT_NAME} \
     MANIFEST_DEPLOYER_COMPONENT_NAME=${MANIFEST_DEPLOYER_COMPONENT_NAME} \
     MOCK_DEPLOYER_COMPONENT_NAME=${MOCK_DEPLOYER_COMPONENT_NAME} \
     CONTAINER_DEPLOYER_COMPONENT_NAME=${CONTAINER_DEPLOYER_COMPONENT_NAME} \
     COMPONENT_REPO=${LANDSCAPER_COMPONENT_NAME} \
     PROVIDER=${PROVIDER} \
     LANDSCAPER_CONTROLLER_IMAGE_PATH=${LANDSCAPER_CONTROLLER_IMAGE_PATH} \
     LANDSCAPER_WEBHOOKS_SERVER_IMAGE_PATH=${LANDSCAPER_WEBHOOKS_SERVER_IMAGE_PATH} \
     LANDSCAPER_AGENT_IMAGE_PATH=${LANDSCAPER_AGENT_IMAGE_PATH} \
     LANDSCAPER_CHART_PATH=${LANDSCAPER_CHART_PATH} \
     LANDSCAPER_CONTROLLER_RBAC_CHART_PATH=${LANDSCAPER_CONTROLLER_RBAC_CHART_PATH} \
     LANDSCAPER_CONTROLLER_DEPLOYMENT_CHART_PATH=${LANDSCAPER_CONTROLLER_DEPLOYMENT_CHART_PATH} \
     LANDSCAPER_AGENT_CHART_PATH=${LANDSCAPER_AGENT_CHART_PATH} \
     HELM_DEPLOYER_CHART_PATH=${HELM_DEPLOYER_CHART_PATH} \
     HELM_DEPLOYER_CONTROLLER_IMAGE_PATH=${HELM_DEPLOYER_CONTROLLER_IMAGE_PATH} \
     MANIFEST_DEPLOYER_CHART_PATH=${MANIFEST_DEPLOYER_CHART_PATH} \
     MANIFEST_DEPLOYER_CONTROLLER_IMAGE_PATH=${MANIFEST_DEPLOYER_CONTROLLER_IMAGE_PATH} \
     CONTAINER_DEPLOYER_CHART_PATH=${CONTAINER_DEPLOYER_CHART_PATH} \
     CONTAINER_DEPLOYER_CONTROLLER_IMAGE_PATH=${CONTAINER_DEPLOYER_CONTROLLER_IMAGE_PATH} \
     CONTAINER_DEPLOYER_INIT_IMAGE_PATH=${CONTAINER_DEPLOYER_INIT_IMAGE_PATH} \
     CONTAINER_DEPLOYER_WAIT_IMAGE_PATH=${CONTAINER_DEPLOYER_WAIT_IMAGE_PATH} \
     MOCK_DEPLOYER_CHART_PATH=${MOCK_DEPLOYER_CHART_PATH} \
     MOCK_DEPLOYER_CONTROLLER_IMAGE_PATH=${MANIFEST_DEPLOYER_CONTROLLER_IMAGE_PATH}

echo "> Transfer Component version ${EFFECTIVE_VERSION} to ${PROVIDER}"
ocm transfer ctf --copy-resources --recursive --overwrite ${COMPONENT_ARCHIVE_PATH} ${PROVIDER}

echo "> Remote Component Version Landscaper"
ocm get componentversion --repo OCIRegistry::${PROVIDER} ${LANDSCAPER_COMPONENT_NAME}:${EFFECTIVE_VERSION} -o yaml

echo "> Remote Component Version Helm Deployer"
ocm get componentversion --repo OCIRegistry::${PROVIDER} ${HELM_DEPLOYER_COMPONENT_NAME}:${EFFECTIVE_VERSION} -o yaml

echo "> Remote Component Version Manifest Deployer"
ocm get componentversion --repo OCIRegistry::${PROVIDER} ${MANIFEST_DEPLOYER_COMPONENT_NAME}:${EFFECTIVE_VERSION} -o yaml

echo "> Remote Component Version Container Deployer"
ocm get componentversion --repo OCIRegistry::${PROVIDER} ${CONTAINER_DEPLOYER_COMPONENT_NAME}:${EFFECTIVE_VERSION} -o yaml

echo "> Remote Component Version Mock Deployer"
ocm get componentversion --repo OCIRegistry::${PROVIDER} ${MOCK_DEPLOYER_COMPONENT_NAME}:${EFFECTIVE_VERSION} -o yaml

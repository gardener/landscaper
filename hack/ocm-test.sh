#!/bin/bash

set -e

SOURCE_PATH="$(realpath $(dirname $0)/..)"
EFFECTIVE_VERSION="$(${SOURCE_PATH}/hack/get-version.sh)"
DOCKER_BUILDER_NAME="ls-multiarch"

echo "> Building docker images for version ${EFFECTIVE_VERSION}"
${SOURCE_PATH}/hack/prepare-docker-builder.sh

LANDSCAPER_CONTROLLER_IMAGE_PATH="landscaper-controller"
LANDSCAPER_WEBHOOKS_SERVER_IMAGE_PATH="landscaper-webhooks-server"
LANDSCAPER_AGENT_IMAGE_PATH="landscaper-agent"

HELM_DEPLOYER_CONTROLLER_IMAGE_PATH="helm-deployer-controller"

#PLATFORM="linux/amd64"
PLATFORM="linux/arm64"

#docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${PLATFORM} -t ${LANDSCAPER_CONTROLLER_IMAGE_PATH}:${EFFECTIVE_VERSION} -f Dockerfile --target landscaper-controller .
#docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${PLATFORM} -t ${LANDSCAPER_WEBHOOKS_SERVER_IMAGE_PATH}:${EFFECTIVE_VERSION} -f Dockerfile --target landscaper-webhooks-server .
#docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${PLATFORM} -t ${LANDSCAPER_AGENT_IMAGE_PATH}:${EFFECTIVE_VERSION} -f Dockerfile --target landscaper-agent .
#docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${PLATFORM} -t ${HELM_DEPLOYER_CONTROLLER_IMAGE_PATH}:${EFFECTIVE_VERSION} -f Dockerfile --target helm-deployer-controller .


echo -n "> Updating helm chart version"
${SOURCE_PATH}/hack/update-helm-chart-version.sh ${EFFECTIVE_VERSION}

echo "> Create Landscaper Component Version ${EFFECTIVE_VERSION}"

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

ocm add componentversions --create --file ${COMPONENT_ARCHIVE_PATH} ${SOURCE_PATH}/.landscaper/components.yaml \
  -- VERSION=${EFFECTIVE_VERSION} \
     COMMIT_SHA=${COMMIT_SHA} \
     LANDSCAPER_COMPONENT_NAME=${LANDSCAPER_COMPONENT_NAME} \
     HELM_DEPLOYER_COMPONENT_NAME=${HELM_DEPLOYER_COMPONENT_NAME} \
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
     HELM_DEPLOYER_CONTROLLER_IMAGE_PATH=${HELM_DEPLOYER_CONTROLLER_IMAGE_PATH}

echo "> Transfer Component version ${EFFECTIVE_VERSION} to ${PROVIDER}"
ocm transfer ctf --copy-resources --recursive --overwrite ${COMPONENT_ARCHIVE_PATH} ${PROVIDER}

echo "> Remote Component Version"
ocm get componentversion --repo OCIRegistry::${PROVIDER} ${LANDSCAPER_COMPONENT_NAME}:${EFFECTIVE_VERSION} -o yaml

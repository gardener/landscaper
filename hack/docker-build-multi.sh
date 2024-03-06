#!/bin/bash
#
# Copyright (c) 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

PROJECT_ROOT="$(realpath $(dirname $0)/..)"
if [[ -z ${EFFECTIVE_VERSION:-} ]]; then
  EFFECTIVE_VERSION=$("$PROJECT_ROOT/hack/get-version.sh")
fi

DOCKER_BUILDER_NAME="ls-multiarch-builder"
if ! docker buildx ls | grep "$DOCKER_BUILDER_NAME" >/dev/null; then
	docker buildx create --name "$DOCKER_BUILDER_NAME"
fi

for pf in ${PLATFORMS//,/ }; do
  echo "> Building docker images for $pf in version $EFFECTIVE_VERSION ..."
	os=${pf%/*}
	arch=${pf#*/}
	docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${pf} -t landscaper-controller:${EFFECTIVE_VERSION}-${os}-${arch} -f Dockerfile --target landscaper-controller "${PROJECT_ROOT}"
	docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${pf} -t landscaper-webhooks-server:${EFFECTIVE_VERSION}-${os}-${arch} -f Dockerfile --target landscaper-webhooks-server "${PROJECT_ROOT}"
	docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${pf} -t container-deployer-controller:${EFFECTIVE_VERSION}-${os}-${arch} -f Dockerfile --target container-deployer-controller "${PROJECT_ROOT}"
	docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${pf} -t container-deployer-init:${EFFECTIVE_VERSION}-${os}-${arch} -f Dockerfile --target container-deployer-init "${PROJECT_ROOT}"
	docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${pf} -t container-deployer-wait:${EFFECTIVE_VERSION}-${os}-${arch} -f Dockerfile --target container-deployer-wait "${PROJECT_ROOT}"
	docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${pf} -t helm-deployer-controller:${EFFECTIVE_VERSION}-${os}-${arch} -f Dockerfile --target helm-deployer-controller "${PROJECT_ROOT}"
	docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${pf} -t manifest-deployer-controller:${EFFECTIVE_VERSION}-${os}-${arch} -f Dockerfile --target manifest-deployer-controller "${PROJECT_ROOT}"
	docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${pf} -t mock-deployer-controller:${EFFECTIVE_VERSION}-${os}-${arch} -f Dockerfile --target mock-deployer-controller "${PROJECT_ROOT}"
done

docker buildx rm "$DOCKER_BUILDER_NAME"